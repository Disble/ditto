//go:build experiment

package perfbench_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Disble/ditto/internal/gosourcefile"
	"github.com/Disble/ditto/internal/schemata"
	"github.com/Disble/ditto/viruses/comparison"
)

const (
	expectedMutants                           = 12
	expectedOrdinaryDriverStarts              = 13
	expectedModuleScopeDriverStarts           = 1
	expectedOrdinaryPackageTestExecutions     = 52
	expectedPackageOnlyPackageTestExecutions  = 13
	expectedModuleScopePackageTestExecutions  = 52
	maximumModuleScopeToOrdinaryDurationRatio = 0.25
)

type experimentMode string

const (
	ordinaryMode    experimentMode = "A ordinary"
	packageOnlyMode experimentMode = "B package-only"
	moduleScopeMode experimentMode = "C module-scope"
)

type experimentFixture struct {
	root       string
	goTool     string
	logPath    string
	selectors  []int
	addresses  []string
	sentinelAt string
}

type experimentResult struct {
	mode                  experimentMode
	driverStarts          int
	packageTestExecutions int
	verdicts              []string
	addresses             []string
	duration              time.Duration
}

// TestModuleScopeExperiment carries out docs/experiments/module-scope-runner.md
// without entering the normal performance gate. It must be run only from a
// complete disposable copy of this tree with .git omitted:
//
//	go test -tags=experiment ./internal/perfbench -run '^TestModuleScopeExperiment$' -count=1
func TestModuleScopeExperiment(t *testing.T) {
	confirmExperimentSource(t)
	goTool := resolveGoTool(t)

	fixture := writeModuleScopeFixture(t, goTool)
	instrumentFixture(t, &fixture)

	// The warm-up is deliberately not evidence. It gives the toolchain and file
	// cache their first work before the three pre-registered interleaved rounds.
	warmup := runExperimentRound(t, fixture, []experimentMode{ordinaryMode, packageOnlyMode, moduleScopeMode})
	t.Logf("discarded warm-up: %s", describeRound(warmup))

	orders := [][]experimentMode{
		{ordinaryMode, packageOnlyMode, moduleScopeMode},
		{packageOnlyMode, moduleScopeMode, ordinaryMode},
		{moduleScopeMode, ordinaryMode, packageOnlyMode},
	}

	for round, order := range orders {
		results := runExperimentRound(t, fixture, order)
		ordinary := results[ordinaryMode]
		moduleScope := results[moduleScopeMode]
		ratio := float64(moduleScope.duration) / float64(ordinary.duration)

		t.Logf("round %d order %s: %s; C/A=%.4f", round+1, modeOrder(order), describeRound(results), ratio)
		assertExperimentExpectations(t, results, fixture)

		if ratio > maximumModuleScopeToOrdinaryDurationRatio {
			t.Fatalf("H3 refuted in round %d: C/A = %.4f, want at most %.2f", round+1, ratio, maximumModuleScopeToOrdinaryDurationRatio)
		}
	}

}

func confirmExperimentSource(t *testing.T) {
	t.Helper()

	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate the experiment source")
	}

	root := filepath.Clean(filepath.Join(filepath.Dir(testFile), "../.."))
	module, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("read tool module identity: %v", err)
	}
	if !strings.Contains(string(module), "module github.com/Disble/ditto\n") {
		t.Fatalf("unexpected tool module identity in %s: %q", root, module)
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); !os.IsNotExist(err) {
		t.Fatalf("experiment must run from a disposable source copy without .git; %s still has .git", root)
	}

	source, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("read experiment source identity: %v", err)
	}
	sum := sha256.Sum256(source)
	t.Logf("source identity: module=%s test=%s sha256=%x", root, testFile, sum)
}

func resolveGoTool(t *testing.T) string {
	t.Helper()

	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("resolve Go toolchain: %v", err)
	}
	goTool, err = filepath.Abs(goTool)
	if err != nil {
		t.Fatalf("resolve absolute Go toolchain path: %v", err)
	}
	if info, err := os.Stat(goTool); err != nil || info.IsDir() {
		t.Fatalf("resolved Go toolchain %q is unusable: %v", goTool, err)
	}

	return goTool
}

func writeModuleScopeFixture(t *testing.T, goTool string) experimentFixture {
	t.Helper()

	root := t.TempDir()
	const module = "example.invalid/module-scope-fixture"
	writeExperimentFile(t, root, "go.mod", "module "+module+"\n\ngo 1.27\n")
	writeExperimentFile(t, root, "pkg0/values.go", fixtureValuesSource())
	writeExperimentFile(t, root, "pkg0/values_test.go", packageZeroTests())
	writeExperimentFile(t, root, "pkg1/values_test.go", packageOneTests(module))
	writeExperimentFile(t, root, "pkg2/pkg2_test.go", packageMarkerTests("pkg2"))
	writeExperimentFile(t, root, "pkg3/pkg3_test.go", packageMarkerTests("pkg3"))

	logPath := filepath.Join(root, "package-executions.log")
	moduleBytes, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("read fixture module identity: %v", err)
	}
	if string(moduleBytes) != "module "+module+"\n\ngo 1.27\n" {
		t.Fatalf("unexpected fixture module identity: %q", moduleBytes)
	}

	return experimentFixture{root: root, goTool: goTool, logPath: logPath}
}

func fixtureValuesSource() string {
	var source strings.Builder
	source.WriteString("package pkg0\n\n")
	for site := range expectedMutants {
		fmt.Fprintf(&source, "func Gate%d(value, threshold int) bool {\n\treturn value > threshold\n}\n", site)
	}
	return source.String()
}

func packageZeroTests() string {
	return `package pkg0

import (
	"os"
	"testing"
)

func TestLocalComparisonSites(t *testing.T) {
	for _, gate := range []func(int, int) bool{Gate1, Gate2, Gate3, Gate4, Gate5} {
		if gate(1, 1) {
			t.Fatal("local comparison mutation survived")
		}
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	recordPackageExecution("pkg0")
	os.Exit(code)
}

func recordPackageExecution(name string) {
	file, err := os.OpenFile(os.Getenv("DITTO_EXPERIMENT_LOG"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		panic(err)
	}
	if _, err := file.WriteString(name + "\n"); err != nil {
		panic(err)
	}
	if err := file.Close(); err != nil {
		panic(err)
	}
}
`
}

func packageOneTests(module string) string {
	return fmt.Sprintf(`package pkg1

import (
	"os"
	"testing"

	"%s/pkg0"
)

func TestDependentPackageSeesTheScopeSentinel(t *testing.T) {
	if pkg0.Gate0(1, 1) {
		t.Fatal("dependent package killed the scope sentinel")
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	recordPackageExecution("pkg1")
	os.Exit(code)
}

func recordPackageExecution(name string) {
	file, err := os.OpenFile(os.Getenv("DITTO_EXPERIMENT_LOG"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		panic(err)
	}
	if _, err := file.WriteString(name + "\n"); err != nil {
		panic(err)
	}
	if err := file.Close(); err != nil {
		panic(err)
	}
}
`, module)
}

func packageMarkerTests(name string) string {
	return fmt.Sprintf(`package %s

import (
	"os"
	"testing"
)

func TestMarker(t *testing.T) {}

func TestMain(m *testing.M) {
	code := m.Run()
	recordPackageExecution(%q)
	os.Exit(code)
}

func recordPackageExecution(name string) {
	file, err := os.OpenFile(os.Getenv("DITTO_EXPERIMENT_LOG"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		panic(err)
	}
	if _, err := file.WriteString(name + "\n"); err != nil {
		panic(err)
	}
	if err := file.Close(); err != nil {
		panic(err)
	}
}
`, name, name)
}

func writeExperimentFile(t *testing.T, root, name, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("create fixture directory for %s: %v", name, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture file %s: %v", name, err)
	}
}

func instrumentFixture(t *testing.T, fixture *experimentFixture) {
	t.Helper()

	path := filepath.Join(fixture.root, "pkg0", "values.go")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture source: %v", err)
	}

	infected := gosourcefile.New("pkg0/values.go", original).Incubate(comparison.New())
	if len(infected) != expectedMutants {
		t.Fatalf("fixture produced %d comparison mutants, want %d", len(infected), expectedMutants)
	}

	mutated := make([][]byte, len(infected))
	addresses := make([]string, len(infected))
	for i, infection := range infected {
		mutation := infection.Mutate()
		mutated[i] = mutation.Mutated()
		addresses[i] = mutation.Address()
	}

	planned := schemata.Plan(original, mutated)
	if len(planned.Selector) != expectedMutants || slices.Contains(planned.Selector, 0) {
		t.Fatalf("real gosourcefile/schemata pipeline selectors = %v, want twelve non-zero selectors", planned.Selector)
	}
	if err := os.WriteFile(path, planned.Instrumented, 0o600); err != nil {
		t.Fatalf("write instrumented fixture: %v", err)
	}

	fixture.selectors = planned.Selector
	fixture.addresses = addresses
	fixture.sentinelAt = addresses[0]
}

func runExperimentRound(t *testing.T, fixture experimentFixture, order []experimentMode) map[experimentMode]experimentResult {
	t.Helper()

	results := make(map[experimentMode]experimentResult, len(order))
	for _, mode := range order {
		resetPackageExecutions(t, fixture.logPath)
		var result experimentResult
		switch mode {
		case ordinaryMode:
			result = runOrdinary(t, fixture)
		case packageOnlyMode:
			result = runPackageOnly(t, fixture)
		case moduleScopeMode:
			result = runModuleScope(t, fixture)
		default:
			t.Fatalf("unknown experiment mode %q", mode)
		}
		results[mode] = result
	}
	return results
}

func runOrdinary(t *testing.T, fixture experimentFixture) experimentResult {
	t.Helper()

	started := time.Now()
	verdicts := make([]string, 0, len(fixture.selectors))
	for index, selector := range append([]int{0}, fixture.selectors...) {
		before := readPackageExecutions(t, fixture.logPath)
		output, err := runExperimentCommand(fixture.root, fixture.logPath, selector, fixture.goTool, "test", "-count=1", "./...")
		assertPackageExecutionGrowth(t, ordinaryMode, index, before, readPackageExecutions(t, fixture.logPath), 4)
		if index == 0 && err != nil {
			t.Fatalf("ordinary baseline failed: %s", output)
		}
		if index > 0 {
			verdicts = append(verdicts, verdict(fixture.addresses[index-1], err != nil))
		}
	}
	return experimentResult{
		mode: ordinaryMode, driverStarts: len(fixture.selectors) + 1,
		packageTestExecutions: readPackageExecutions(t, fixture.logPath),
		verdicts:              verdicts, addresses: append([]string(nil), fixture.addresses...), duration: time.Since(started),
	}
}

func runPackageOnly(t *testing.T, fixture experimentFixture) experimentResult {
	t.Helper()

	binaryDir := t.TempDir()
	binary := filepath.Join(binaryDir, packageTestBinaryName("pkg0"))
	started := time.Now()
	if output, err := runExperimentCommand(fixture.root, fixture.logPath, 0, fixture.goTool, "test", "-c", "-o", binary, "./pkg0"); err != nil {
		t.Fatalf("package-only compile failed: %s", output)
	}

	verdicts := runBinarySelections(t, fixture, packageOnlyMode, map[string]string{"pkg0": binary}, 1)
	return experimentResult{
		mode: packageOnlyMode, driverStarts: 1,
		packageTestExecutions: readPackageExecutions(t, fixture.logPath),
		verdicts:              verdicts, addresses: append([]string(nil), fixture.addresses...), duration: time.Since(started),
	}
}

func runModuleScope(t *testing.T, fixture experimentFixture) experimentResult {
	t.Helper()

	binaryDir := t.TempDir()
	started := time.Now()
	if output, err := runExperimentCommand(fixture.root, fixture.logPath, 0, fixture.goTool, "test", "-c", "-o", binaryDir, "./..."); err != nil {
		t.Fatalf("module-scope compile failed: %s", output)
	}

	binaries := map[string]string{}
	for _, name := range []string{"pkg0", "pkg1", "pkg2", "pkg3"} {
		binary := filepath.Join(binaryDir, packageTestBinaryName(name))
		if _, err := os.Stat(binary); err != nil {
			t.Fatalf("module-scope compile did not produce %s: %v", binary, err)
		}
		binaries[name] = binary
	}

	verdicts := runBinarySelections(t, fixture, moduleScopeMode, binaries, 4)
	return experimentResult{
		mode: moduleScopeMode, driverStarts: 1,
		packageTestExecutions: readPackageExecutions(t, fixture.logPath),
		verdicts:              verdicts, addresses: append([]string(nil), fixture.addresses...), duration: time.Since(started),
	}
}

func packageTestBinaryName(packageName string) string {
	name := packageName + ".test"
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func runBinarySelections(t *testing.T, fixture experimentFixture, mode experimentMode, binaries map[string]string, expectedExecutions int) []string {
	t.Helper()

	verdicts := make([]string, 0, len(fixture.selectors))
	for index, selector := range append([]int{0}, fixture.selectors...) {
		before := readPackageExecutions(t, fixture.logPath)
		killed := false
		for _, packageName := range []string{"pkg0", "pkg1", "pkg2", "pkg3"} {
			binary, included := binaries[packageName]
			if !included {
				continue
			}
			output, err := runExperimentCommand(filepath.Join(fixture.root, packageName), fixture.logPath, selector,
				binary, "-test.count=1", "-test.timeout=10m")
			if index == 0 && err != nil {
				t.Fatalf("%s baseline failed in %s: %s", packageName, fixture.root, output)
			}
			killed = killed || err != nil
		}
		assertPackageExecutionGrowth(t, mode, index, before, readPackageExecutions(t, fixture.logPath), expectedExecutions)
		if index > 0 {
			verdicts = append(verdicts, verdict(fixture.addresses[index-1], killed))
		}
	}
	return verdicts
}

func runExperimentCommand(directory, logPath string, selector int, tool string, arguments ...string) (string, error) {
	command := exec.Command(tool, arguments...) //nolint:gosec,noctx // tool is resolved and every argument is fixture-controlled
	command.Dir = directory
	command.Env = experimentEnvironment(logPath, selector)
	output, err := command.CombinedOutput()
	return string(output), err
}

func experimentEnvironment(logPath string, selector int) []string {
	remove := []string{
		"GIT_DIR=", "GIT_INDEX_FILE=", "GIT_WORK_TREE=", "GIT_OBJECT_DIRECTORY=", "GIT_COMMON_DIR=",
		"DITTO_MUTANT=", "DITTO_EXPERIMENT_LOG=",
	}
	environment := make([]string, 0, len(os.Environ())+2)
	for _, entry := range os.Environ() {
		if !hasExperimentPrefix(entry, remove) {
			environment = append(environment, entry)
		}
	}
	return append(environment, fmt.Sprintf("DITTO_MUTANT=%d", selector), "DITTO_EXPERIMENT_LOG="+logPath)
}

func hasExperimentPrefix(value string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func resetPackageExecutions(t *testing.T, logPath string) {
	t.Helper()

	if err := os.WriteFile(logPath, nil, 0o600); err != nil {
		t.Fatalf("reset package execution log: %v", err)
	}
}

func readPackageExecutions(t *testing.T, logPath string) int {
	t.Helper()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read package execution log: %v", err)
	}
	return len(strings.Fields(string(content)))
}

func verdict(address string, killed bool) string {
	if killed {
		return address + "=killed"
	}
	return address + "=survived"
}

func assertExperimentExpectations(t *testing.T, results map[experimentMode]experimentResult, fixture experimentFixture) {
	t.Helper()

	ordinary := results[ordinaryMode]
	packageOnly := results[packageOnlyMode]
	moduleScope := results[moduleScopeMode]

	assertExact(t, ordinary.mode, "Go driver starts", ordinary.driverStarts, expectedOrdinaryDriverStarts)
	assertExact(t, ordinary.mode, "package-test executions", ordinary.packageTestExecutions, expectedOrdinaryPackageTestExecutions)
	assertExact(t, packageOnly.mode, "Go driver starts", packageOnly.driverStarts, expectedModuleScopeDriverStarts)
	assertExact(t, packageOnly.mode, "package-test executions", packageOnly.packageTestExecutions, expectedPackageOnlyPackageTestExecutions)
	assertExact(t, moduleScope.mode, "Go driver starts", moduleScope.driverStarts, expectedModuleScopeDriverStarts)
	assertExact(t, moduleScope.mode, "package-test executions", moduleScope.packageTestExecutions, expectedModuleScopePackageTestExecutions)

	if !slices.Equal(ordinary.addresses, fixture.addresses) || !slices.Equal(moduleScope.addresses, ordinary.addresses) {
		t.Fatalf("H1 refuted: ordered mutant addresses differ: A=%v C=%v", ordinary.addresses, moduleScope.addresses)
	}
	if !slices.Equal(moduleScope.verdicts, ordinary.verdicts) {
		t.Fatalf("H1 refuted: ordered verdicts differ: A=%v C=%v", ordinary.verdicts, moduleScope.verdicts)
	}

	killed := 0
	for _, got := range ordinary.verdicts {
		if strings.HasSuffix(got, "=killed") {
			killed++
		}
	}
	if killed != 6 || len(ordinary.verdicts)-killed != 6 {
		t.Fatalf("fixture control produced %d killed and %d survived, want 6 and 6: %v", killed, len(ordinary.verdicts)-killed, ordinary.verdicts)
	}

	for index := range ordinary.verdicts {
		if index == 0 {
			if ordinary.verdicts[index] != fixture.sentinelAt+"=killed" || packageOnly.verdicts[index] != fixture.sentinelAt+"=survived" {
				t.Fatalf("H2 refuted: scope sentinel %s is A=%s B=%s", fixture.sentinelAt, ordinary.verdicts[index], packageOnly.verdicts[index])
			}
			continue
		}
		if packageOnly.verdicts[index] != ordinary.verdicts[index] {
			t.Fatalf("H2 refuted: package-only disagrees beyond sentinel at %s: A=%s B=%s", fixture.addresses[index], ordinary.verdicts[index], packageOnly.verdicts[index])
		}
	}
}

// TestExactCounterRefusalRejectsCounterDrift keeps the experiment's deliberate
// failure control in the checked source rather than depending on a temporary
// source edit in a disposable copy.
func TestExactCounterRefusalRejectsCounterDrift(t *testing.T) {
	got := exactCounterRefusal(moduleScopeMode, "package-test executions", 52, 51)
	const want = "C module-scope package-test executions = 52, want exactly 51"
	if got != want {
		t.Fatalf("counter-drift refusal = %q, want %q", got, want)
	}
	t.Logf("counter-drift refusal: %s", got)
}

func assertExact(t *testing.T, mode experimentMode, counter string, got, want int) {
	t.Helper()
	if refusal := exactCounterRefusal(mode, counter, got, want); refusal != "" {
		t.Fatal(refusal)
	}
}

func assertPackageExecutionGrowth(t *testing.T, mode experimentMode, selection, before, after, want int) {
	t.Helper()
	counter := fmt.Sprintf("selection %d package-test execution growth", selection)
	if refusal := exactCounterRefusal(mode, counter, after-before, want); refusal != "" {
		t.Fatal(refusal)
	}
}

func exactCounterRefusal(mode experimentMode, counter string, got, want int) string {
	if got == want {
		return ""
	}
	return fmt.Sprintf("%s %s = %d, want exactly %d", mode, counter, got, want)
}

func modeOrder(order []experimentMode) string {
	parts := make([]string, len(order))
	for i, mode := range order {
		parts[i] = string(mode)
	}
	return strings.Join(parts, " -> ")
}

func describeRound(results map[experimentMode]experimentResult) string {
	parts := make([]string, 0, len(results))
	for _, mode := range []experimentMode{ordinaryMode, packageOnlyMode, moduleScopeMode} {
		result := results[mode]
		parts = append(parts, fmt.Sprintf("%s drivers=%d packages=%d duration=%s verdicts=%v addresses=%v",
			mode, result.driverStarts, result.packageTestExecutions, result.duration, result.verdicts, result.addresses))
	}
	return strings.Join(parts, "; ")
}

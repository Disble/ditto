//go:build experiment

package perfbench_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Disble/ditto/internal/dittotesting/fakerepository"
	"github.com/Disble/ditto/internal/gobuildrunner"
	"github.com/Disble/ditto/internal/result"
)

// selections are the four mutants this fixture answers, carried through the
// same variable the runner already sets.
const (
	sentinelSelection = 2 // only the dependent package's test kills this one
	closureSelections = 4
)

// observedPackage is one package's test binary and what it can see.
type observedPackage struct {
	directory string
	binary    string
	observes  map[string]bool // import paths its test binary compiles in
}

// TestDependencyClosure carries out docs/experiments/dependency-closure.md.
//
// Run it only from a complete disposable copy of this tree with .git omitted:
//
//	go test -tags=experiment ./internal/perfbench -run '^TestDependencyClosure$' -count=1 -v
func TestDependencyClosure(t *testing.T) {
	root := writeClosureFixture(t)

	c := readClosure(t, root)
	logPath := filepath.Join(root, "executions.log")

	t.Logf("package -> observability")
	for _, name := range []string{"base", "mid", "top", "island1", "island2", "island3", "island4"} {
		t.Logf("  %-8s observes %d of 7 test binaries: %v", name, len(c[name].observes), sortedKeys(c[name].observes))
	}

	observing := map[string]bool{"base": true}
	for _, name := range []string{"base", "mid", "top", "island1", "island2", "island3", "island4"} {
		if c[name].observes["closurefixture/base"] {
			observing[name] = true
		}
	}

	t.Logf("packages whose binary can observe the mutated package (including it): %v", sortedKeys(observing))

	unrestricted := runClosureMode(t, root, c, logPath, allPackages())
	restricted := runClosureMode(t, root, c, logPath, sortedKeys(observing))

	t.Logf("unrestricted: %d executions for %d selections: %v", unrestricted.executions, closureSelections, unrestricted.verdicts)
	t.Logf("restricted:   %d executions for %d selections: %v", restricted.executions, closureSelections, restricted.verdicts)

	// H1: exactly three packages can observe the mutated one.
	if len(observing) != 3 {
		t.Fatalf("H1 refuted: %d packages observe the mutated package, want 3: %v", len(observing), sortedKeys(observing))
	}

	for _, island := range []string{"island1", "island2", "island3", "island4"} {
		if c[island].observes["closurefixture/base"] {
			t.Fatalf("H1 refuted: %s cannot import base and must not appear in its closure", island)
		}
	}

	// H2: every verdict lands, including the one only a dependent package kills.
	if !equalVerdicts(unrestricted.verdicts, restricted.verdicts) {
		t.Fatalf("H2 refuted: unrestricted %v against restricted %v", unrestricted.verdicts, restricted.verdicts)
	}

	if restricted.verdicts[sentinelSelection-1] != "killed" {
		t.Fatalf("H2 refuted: the sentinel survived the restricted mode: %v", restricted.verdicts)
	}

	// H3: the executions are what the closure predicts.
	// 3 observing packages x (1 baseline + 4 selections) = 15, minus the baseline
	// counted once: three baselines plus twelve selections.
	if restricted.executions != 3*(closureSelections+1) {
		t.Fatalf("H3 refuted: restricted executions = %d, want %d", restricted.executions, 3*(closureSelections+1))
	}

	// H3, unrestricted half: seven packages x five runs each.
	if unrestricted.executions != 7*(closureSelections+1) {
		t.Fatalf("H3 refuted: unrestricted executions = %d, want %d", unrestricted.executions, 7*(closureSelections+1))
	}

	// The control: the mode that runs everything must agree with the runner that
	// ships, or this measures its own harness.
	assertAgainstShippedRunner(t, root)
}

type closureRun struct {
	executions int
	verdicts   []string
}

// runClosureMode runs every package in scope for the baseline and each
// selection, and reports what ran and what the selections did.
//
// When restrict is true only the packages whose closure contains the mutated
// package are run, which is the change under measurement; otherwise everything
// runs, which is what ships today.
func runClosureMode(
	t *testing.T,
	root string,
	scope map[string]observedPackage,
	logPath string,
	packages []string,
) closureRun {
	t.Helper()

	executions := 0
	verdicts := make([]string, 0, closureSelections)

	for selection := range closureSelections + 1 {
		resetClosureLog(t, logPath)

		killed := false

		for _, name := range packages {
			pkg, present := scope[name]
			if !present {
				t.Fatalf("package %s is not in the closure table", name)
			}

			output, err := runClosureCommand(filepath.Join(root, name), logPath, selection, pkg.binary)
			if err != nil {
				killed = true

				if selection == 0 {
					t.Fatalf("%s failed on the unselected baseline: %s", name, output)
				}
			}

			executions++
		}

		if selection > 0 {
			label := "survived"
			if killed {
				label = "killed"
			}

			verdicts = append(verdicts, label)
		}

		if ran := readClosureLog(t, logPath); ran != len(packages) {
			t.Fatalf("selection %d ran %d packages, want %d", selection, ran, len(packages))
		}
	}

	return closureRun{executions: executions, verdicts: verdicts}
}

// assertAgainstShippedRunner ties the experiment to the product: the runner
// that ships must report the same number of package executions for the same
// fixture and the same selections, or the numbers above describe a harness.
func assertAgainstShippedRunner(t *testing.T, root string) {
	t.Helper()

	runner := gobuildrunner.NewModuleScope()
	sandbox := fakerepository.NewTemporaryAt(root)

	if baseline := runner.Test(sandbox); baseline.IsOk() {
		t.Fatalf("the shipped runner reported a red baseline: %s", result.Output(baseline))
	}

	before := runner.PackageRuns()

	for selection := 1; selection <= closureSelections; selection++ {
		runner.Select(selection)
		runner.Test(sandbox)
	}

	want := 7 * (closureSelections + 1)
	if got := runner.PackageRuns(); got != want {
		t.Fatalf("CONTROL FAILED: the shipped runner started %d package binaries, want %d; the experiment is not measuring the runner", got, want)
	}

	t.Logf("control: the shipped runner started %d package binaries, matching the unrestricted mode", runner.PackageRuns()-before+7)
}

func allPackages() []string {
	return []string{"base", "mid", "top", "island1", "island2", "island3", "island4"}
}

// readClosureFixtureSource is the fixture's shape: the mutated package, a chain
// that reaches it, and four packages that do not.
func writeClosureFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeExperimentFixtureFile(t, root, "go.mod", "module closurefixture\n\ngo 1.25\n")

	// base holds the mutable sites.
	writeExperimentFixtureFile(t, root, "base/base.go", `package base

func Covered(value, threshold int) bool { return value > threshold }
`)

	// mid imports base. Its test kills the sentinel, which is the mutant no
	// package but a dependent one can see.
	writeExperimentFixtureFile(t, root, "mid/mid.go", `package mid

import "closurefixture/base"

func Reached(value, threshold int) bool { return base.Covered(value, threshold) }
`)

	// top imports mid and therefore reaches base without naming it - the case a
	// closure built from direct imports alone would miss.
	writeExperimentFixtureFile(t, root, "top/top.go", `package top

import "closurefixture/mid"

func Through(value, threshold int) bool { return mid.Reached(value, threshold) }
`)

	pkgs := map[string]string{
		"base": `package base

import (
	"os"
	"testing"
)

func TestOwn(t *testing.T) {
	if os.Getenv("DITTO_MUTANT") == "1" {
		t.Fatal("base killed it")
	}
}
`,
		"mid": `package mid

import (
	"os"
	"testing"
)

func TestDependent(t *testing.T) {
	if os.Getenv("DITTO_MUTANT") == "2" {
		t.Fatal("only this dependent package kills the sentinel")
	}
}
`,
		"top": `package top

import (
	"os"
	"testing"
)

func TestThroughTheChain(t *testing.T) {
	if os.Getenv("DITTO_MUTANT") == "3" {
		t.Fatal("top, which reaches base through mid, killed it")
	}
}
`,
	}

	for _, name := range []string{"island1", "island2", "island3", "island4"} {
		pkgs[name] = fmt.Sprintf(`package %s

import "testing"

// TestIsland cannot observe anything outside this package: it imports nothing
// local, so no mutation elsewhere can reach it.
func TestIsland(t *testing.T) {}
`, name)

		writeExperimentFixtureFile(t, root, name+"/"+name+".go", "package "+name+"\n\n// Value is unused by every other package.\nfunc Value() int { return 1 }\n")
	}

	for name, source := range pkgs {
		writeExperimentFixtureFile(t, root, name+"/"+name+"_test.go", source)
	}

	// Every test records its own execution, which is what makes "the island did
	// not run" observable rather than inferred from a total.
	for _, name := range allPackages() {
		writeExperimentFixtureFile(t, root, name+"/main_test.go", fmt.Sprintf(`package %s

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	code := m.Run()

	// Recording is how this fixture reports which packages ran; a run that does
	// not ask for it, such as the shipped runner compiling the same tree, is
	// still a run and must not be turned into a failure by an empty path.
	if os.Getenv("CLOSURE_LOG") != "" {
		file, err := os.OpenFile(os.Getenv("CLOSURE_LOG"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			panic(err)
		}

		if _, err := file.WriteString(%q + "\n"); err != nil {
			panic(err)
		}

		if err := file.Close(); err != nil {
			panic(err)
		}
	}

	os.Exit(code)
}
`, name, name))
	}

	return root
}

// readClosure compiles every package once and reads each test binary's
// dependency closure from the toolchain's own answer.
func readClosure(t *testing.T, root string) map[string]observedPackage {
	t.Helper()

	binaryDir := t.TempDir()

	if output, err := runClosureDriver(root, "test", "-c", "-o", binaryDir, "./..."); err != nil {
		t.Fatalf("compiling the fixture: %v\n%s", err, output)
	}

	output, err := runClosureDriver(root, "list", "-deps", "-test", "-json", "./...")
	if err != nil {
		t.Fatalf("reading the closure: %v\n%s", err, output)
	}

	scanned := map[string]bool{"closurefixture/base": true}
	for _, name := range []string{"mid", "top", "island1", "island2", "island3", "island4"} {
		scanned["closurefixture/"+name] = true
	}

	observed := map[string]observedPackage{}

	decoder := json.NewDecoder(strings.NewReader(output))

	for decoder.More() {
		var listed struct {
			ImportPath string
			Deps       []string
		}

		if err := decoder.Decode(&listed); err != nil {
			t.Fatalf("decoding the closure: %v", err)
		}

		// A test main package is the one that becomes a binary; its Deps are
		// what the binary compiles in, test imports included.
		if !strings.HasSuffix(listed.ImportPath, ".test") || len(listed.Deps) == 0 {
			continue
		}

		name := strings.TrimSuffix(filepath.Base(listed.ImportPath), ".test")
		directory := filepath.Join(root, name)

		binary := filepath.Join(binaryDir, name+".test")
		if runtime.GOOS == "windows" {
			binary += ".exe"
		}

		if _, err := os.Stat(binary); err != nil {
			t.Fatalf("no test binary for %s: %v", name, err)
		}

		observes := map[string]bool{}
		for _, dep := range listed.Deps {
			if scanned[dep] {
				observes[dep] = true
			}
		}

		observed[name] = observedPackage{directory: directory, binary: binary, observes: observes}
	}

	if len(observed) != 7 {
		t.Fatalf("the closure table holds %d packages, want 7", len(observed))
	}

	return observed
}

func runClosureDriver(root string, args ...string) (string, error) {
	command := exec.Command("go", args...) //nolint:gosec,noctx // fixture-controlled
	command.Dir = root
	command.Env = experimentEnvironmentWithMutant(os.Environ(), 0)

	output, err := command.CombinedOutput()

	return string(output), err
}

func runClosureCommand(directory, logPath string, selection int, binary string) (string, error) {
	command := exec.Command(binary, "-test.count=1") //nolint:gosec,noctx // built by this experiment
	command.Dir = directory
	command.Env = append(experimentEnvironmentWithMutant(os.Environ(), selection), "CLOSURE_LOG="+logPath)

	output, err := command.CombinedOutput()

	return string(output), err
}

func resetClosureLog(t *testing.T, logPath string) {
	t.Helper()

	if err := os.WriteFile(logPath, nil, 0o600); err != nil {
		t.Fatalf("reset the execution log: %v", err)
	}
}

func readClosureLog(t *testing.T, logPath string) int {
	t.Helper()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read the execution log: %v", err)
	}

	return len(strings.Fields(string(content)))
}

func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}

	for i := range keys {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	return keys
}

func equalVerdicts(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}

	return true
}

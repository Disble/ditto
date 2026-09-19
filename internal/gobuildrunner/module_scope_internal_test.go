package gobuildrunner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Disble/ditto/internal/dittotesting/fakerepository"
	"github.com/Disble/ditto/internal/result"
	"github.com/Disble/ditto/internal/verdict"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Binary naming is pure; actual Windows process execution belongs to the
// repository's existing Windows CI matrix.
func TestModuleTestBinaryName(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name       string
		importPath string
		goos       string
		want       string
	}{
		{name: "POSIX", importPath: "example.test/internal/calc", goos: "linux", want: "calc.test"},
		{name: "Windows", importPath: "example.test/internal/calc", goos: "windows", want: "calc.test.exe"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := moduleTestBinaryName(tt.importPath, tt.goos); got != tt.want {
				t.Fatalf("moduleTestBinaryName(%q, %q) = %q, want %q", tt.importPath, tt.goos, got, tt.want)
			}
		})
	}
}

func TestModuleScopeRunnerRunsTheCompleteDefaultScope(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the Go toolchain")
	}

	root := moduleFixture(t, map[string]string{
		"subject/subject.go": `package subject

import "os"

func Changed() bool { return os.Getenv("DITTO_MUTANT") != "1" }
`,
		"subject/subject_test.go": `package subject

import (
	"os"
	"testing"
)

func TestOwnPackageStillRuns(t *testing.T) {
	f, err := os.OpenFile("../order", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil { t.Fatal(err) }
	defer f.Close()
	if _, err := f.WriteString("subject:" + os.Getenv("DITTO_MUTANT") + "\n"); err != nil { t.Fatal(err) }
}
`,
		"dependent/dependent_test.go": `package dependent

import (
	"os"
	"testing"

	"fixture/subject"
)

func TestDependentPackageKillsTheSentinel(t *testing.T) {
	f, err := os.OpenFile("../order", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil { t.Fatal(err) }
	defer f.Close()
	if _, err := f.WriteString("dependent:" + os.Getenv("DITTO_MUTANT") + "\n"); err != nil { t.Fatal(err) }
	if !subject.Changed() { t.Fatal("dependent test killed the sentinel") }
}
`,
	})
	runner := NewModuleScope()

	baseline := runner.Test(fakerepository.NewTemporaryAt(root))
	require.False(t, baseline.IsOk(), "the unselected baseline must be green")
	runner.Select(1)
	outcome := runner.Test(fakerepository.NewTemporaryAt(root))

	assert.True(t, outcome.IsOk(), "a dependent package must kill the selected sentinel")
	assert.True(t, runner.Built())
	assert.Equal(t, 1, runner.Discoveries())
	assert.Equal(t, 2, runner.ToolchainStarts())
	assert.Equal(t, 1, runner.Compilations())
	assert.Equal(t, 2, runner.Selections())
	assert.Equal(t, 4, runner.PackageRuns())

	order, err := os.ReadFile(filepath.Join(root, "order"))
	require.NoError(t, err)
	assert.Equal(t, "dependent:0\nsubject:0\ndependent:1\nsubject:1\n", string(order))
	assert.Contains(t, string(order), "dependent:1\nsubject:1\n", "the later sorted subject package ran after dependent killed the sentinel")
}

// TestModuleScopeRunnerBuildsCollidingPackagesInSeparateBatches replaces
// TestModuleScopeRunnerRejectsDuplicateBinaryNames, which owned the refusal of
// a module whose packages share a test-binary basename. A measurement on ditto
// itself contradicted that written claim: three packages in scope produced
// `ditto.test.exe`, the scope refused with errBinaryNameCollision, and
// GatedLaboratory fell back to every file — 0 of 850 mutants gated. The same
// two-calc fixture must now build, plan two batches, start both binaries and
// reach both tests.
func TestModuleScopeRunnerBuildsCollidingPackagesInSeparateBatches(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the Go toolchain")
	}

	root := moduleFixture(t, map[string]string{
		"first/calc/calc.go": "package calc\n",
		"first/calc/calc_test.go": `package calc

import (
	"os"
	"testing"
)

func TestOne(t *testing.T) {
	f, err := os.OpenFile("../../order", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil { t.Fatal(err) }
	defer f.Close()
	if _, err := f.WriteString("first\n"); err != nil { t.Fatal(err) }
}
`,
		"second/calc/calc.go": "package calc\n",
		"second/calc/calc_test.go": `package calc

import (
	"os"
	"testing"
)

func TestTwo(t *testing.T) {
	f, err := os.OpenFile("../../order", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil { t.Fatal(err) }
	defer f.Close()
	if _, err := f.WriteString("second\n"); err != nil { t.Fatal(err) }
}
`,
	})
	runner := NewModuleScope()

	outcome := runner.Test(fakerepository.NewTemporaryAt(root))

	assert.False(t, outcome.IsOk(), "the unselected baseline of both packages must pass")
	assert.True(t, runner.Built(), "a colliding module must compile instead of refusing")
	assert.Equal(t, 1, runner.Discoveries())
	assert.Equal(t, 3, runner.ToolchainStarts(), "one discovery plus one compile invocation per batch")
	assert.Equal(t, 2, runner.Compilations(), "the two calc packages cannot share one output directory")
	assert.Equal(t, 1, runner.Selections())
	assert.Equal(t, 2, runner.PackageRuns(), "both colliding packages must still run")

	order, err := os.ReadFile(filepath.Join(root, "order"))
	require.NoError(t, err)
	assert.Equal(t, "first\nsecond\n", string(order), "both test binaries started, in sorted package order")
}

func TestPlanCompileBatchesSeparatesCollidingNames(t *testing.T) {
	t.Parallel()

	packages := []modulePackage{
		{importPath: "fixture/first/calc", hasTests: true},
		{importPath: "fixture/second/calc", hasTests: true},
	}

	batches := planCompileBatches(packages, "linux")

	require.Len(t, batches, 2)
	require.Len(t, batches[0], 1)
	require.Len(t, batches[1], 1)
	assert.Equal(t, "fixture/first/calc", batches[0][0].importPath, "sorted order fills the first batch first")
	assert.Equal(t, "fixture/second/calc", batches[1][0].importPath)
}

func TestPlanCompileBatchesKeepsOneBatchForDistinctNames(t *testing.T) {
	t.Parallel()

	packages := []modulePackage{
		{importPath: "fixture/alpha", hasTests: true},
		{importPath: "fixture/beta", hasTests: true},
		{importPath: "fixture/gamma", hasTests: true},
	}

	batches := planCompileBatches(packages, "linux")

	require.Len(t, batches, 1, "a module without collisions must stay one compilation")
	require.Len(t, batches[0], 3)
	assert.Equal(t, []string{"fixture/alpha", "fixture/beta", "fixture/gamma"},
		[]string{batches[0][0].importPath, batches[0][1].importPath, batches[0][2].importPath},
		"sorted-by-import-path order is preserved inside the batch")
}

func TestPlanCompileBatchesIsDeterministic(t *testing.T) {
	t.Parallel()

	packages := []modulePackage{
		{importPath: "fixture/alpha", hasTests: true},
		{importPath: "fixture/first/calc", hasTests: true},
		{importPath: "fixture/library", hasTests: false},
		{importPath: "fixture/omega", hasTests: true},
		{importPath: "fixture/second/calc", hasTests: true},
	}

	first := planCompileBatches(packages, "linux")
	second := planCompileBatches(packages, "linux")

	assert.Equal(t, first, second, "same input, same batches")
}

func TestPlanCompileBatchesTreatsCaseOnlyDifferencesAsCollidingOnWindows(t *testing.T) {
	t.Parallel()

	packages := []modulePackage{
		{importPath: "fixture/Calc", hasTests: true},
		{importPath: "fixture/calc", hasTests: true},
	}

	assert.Len(t, planCompileBatches(packages, "windows"), 2,
		"on Windows the two names are one file, so the packages cannot share a batch")
	assert.Len(t, planCompileBatches(packages, "linux"), 1,
		"on a case-sensitive filesystem the names are distinct files")
}

// TestPlanCompileBatchesIncludesPackagesWithoutTests replaces
// TestPlanCompileBatchesPlansNothingWithoutTestPackages, which owned the
// opposite behaviour: it asserted that packages without tests belong to no
// batch. The compiled set must equal the configured ./... scope instead —
// `go list` does not type-check, so an untested package with a type error only
// fails the release if it is among the compile arguments.
func TestPlanCompileBatchesIncludesPackagesWithoutTests(t *testing.T) {
	t.Parallel()

	packages := []modulePackage{
		{importPath: "fixture/alpha", hasTests: true},
		{importPath: "fixture/library", hasTests: false},
	}

	batches := planCompileBatches(packages, "linux")

	require.Len(t, batches, 1)
	require.Len(t, batches[0], 2, "the union of batch arguments must be the whole discovered scope")
	assert.Equal(t, "fixture/alpha", batches[0][0].importPath)
	assert.Equal(t, "fixture/library", batches[0][1].importPath)
}

// The toolchain refuses duplicate basenames per argument list even when neither
// package has test files — measured: `go test -c -o <dir> ./p/lib1 ./q/lib1`
// exits 1 with "cannot write test binary lib1.test for multiple packages". So
// the collision key covers every package, tested or not.
func TestPlanCompileBatchesSeparatesCollidingNamesAmongUntestedPackages(t *testing.T) {
	t.Parallel()

	packages := []modulePackage{
		{importPath: "fixture/first/calc", hasTests: false},
		{importPath: "fixture/second/calc", hasTests: false},
	}

	batches := planCompileBatches(packages, "linux")

	require.Len(t, batches, 2)
	assert.Equal(t, "fixture/first/calc", batches[0][0].importPath)
	assert.Equal(t, "fixture/second/calc", batches[1][0].importPath)
}

// TestPlanCompileBatchesDittoShape encodes the measured shape that forced the
// batching: on ditto itself the scope held three packages named ditto and two
// named dittotesting. Each colliding package lands in its own batch in sorted
// order; non-colliding packages share the earliest batch free of their name.
func TestPlanCompileBatchesDittoShape(t *testing.T) {
	t.Parallel()

	packages := []modulePackage{
		{importPath: "github.com/Disble/ditto", hasTests: true},
		{importPath: "github.com/Disble/ditto/cmd/ditto", hasTests: true},
		{importPath: "github.com/Disble/ditto/dittotesting", hasTests: true},
		{importPath: "github.com/Disble/ditto/internal/ditto", hasTests: true},
		{importPath: "github.com/Disble/ditto/internal/dittotesting", hasTests: true},
	}

	batches := planCompileBatches(packages, "linux")

	require.Len(t, batches, 3)
	assert.Equal(t, []string{"github.com/Disble/ditto", "github.com/Disble/ditto/dittotesting"},
		[]string{batches[0][0].importPath, batches[0][1].importPath})
	assert.Equal(t, []string{"github.com/Disble/ditto/cmd/ditto", "github.com/Disble/ditto/internal/dittotesting"},
		[]string{batches[1][0].importPath, batches[1][1].importPath})
	assert.Equal(t, []string{"github.com/Disble/ditto/internal/ditto"},
		[]string{batches[2][0].importPath})
}

func TestValidateBatchesRefusesADuplicatedBatch(t *testing.T) {
	t.Parallel()

	batches := [][]modulePackage{
		{
			{importPath: "fixture/first/calc", hasTests: true},
			{importPath: "fixture/second/calc", hasTests: true},
		},
	}

	err := validateBatches(batches, "linux")

	require.ErrorIs(t, err, errBinaryNameCollision)
	assert.ErrorContains(t, err, "fixture/first/calc")
	assert.ErrorContains(t, err, "fixture/second/calc")
	assert.ErrorContains(t, err, "calc.test")
}

// The invariant covers untested packages too: one batch directory receives one
// `go test -c` argument list, and the toolchain refuses a duplicate basename in
// one argument list even without test files anywhere.
func TestValidateBatchesRefusesDuplicateNamesAmongUntestedPackages(t *testing.T) {
	t.Parallel()

	batches := [][]modulePackage{
		{
			{importPath: "fixture/first/calc", hasTests: false},
			{importPath: "fixture/second/calc", hasTests: false},
		},
	}

	require.ErrorIs(t, validateBatches(batches, "linux"), errBinaryNameCollision)
}

// TestValidateBatchesAcceptsDistinctBatches previously owned the opposite
// untested-package behaviour: its library case demonstrated that packages
// without tests are invisible to validation. They now carry collision keys and
// are validated like every other package; this case keeps them in the batch to
// pin that distinct names among tested and untested packages pass together.
func TestValidateBatchesAcceptsDistinctBatches(t *testing.T) {
	t.Parallel()

	batches := [][]modulePackage{
		{
			{importPath: "fixture/alpha", hasTests: true},
			{importPath: "fixture/beta", hasTests: true},
			{importPath: "fixture/library", hasTests: false},
			{importPath: "fixture/other", hasTests: false},
		},
		{
			{importPath: "fixture/second/calc", hasTests: true},
		},
	}

	assert.NoError(t, validateBatches(batches, "linux"))
}

// TestModuleScopeRunnerFailsClosedWhenUntestedPackageHasATypeError is the guard
// for the measured scope-fidelity gap: `go list -deps -test -json ./...` does
// not type-check, so discovery passes a module the configured `go test ./...`
// refuses, and the old planner — which compiled only packages with tests —
// reported Built=true against a module the ordinary command rejects.
func TestModuleScopeRunnerFailsClosedWhenUntestedPackageHasATypeError(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the Go toolchain")
	}

	root := moduleFixture(t, map[string]string{
		"good/good.go":      "package good\n\nfunc Value() int { return 1 }\n",
		"good/good_test.go": "package good\n\nimport \"testing\"\n\nfunc TestGood(t *testing.T) {}\n",
		// No test files, imported by nothing: discovery cannot see this failure.
		"broken/broken.go": "package broken\n\nfunc Broken() int { return \"not an int\" }\n",
	})
	runner := NewModuleScope()

	outcome := runner.Test(fakerepository.NewTemporaryAt(root))

	assert.True(t, outcome.IsOk(), "the compiled set must equal the configured ./... scope")
	assert.False(t, runner.Built())
	assert.Equal(t, 1, runner.Discoveries())
	assert.GreaterOrEqual(t, runner.Compilations(), 1, "at least one compile attempt was made")
	assert.Equal(t, 0, runner.PackageRuns())
	assert.Contains(t, outcome.String(), "broken", "the failure must name the offending package")
}

// TestModuleScopeRunnerBuildsWithoutAnyTestPackages replaces the
// errNoTestPackages refusal. Measured: `go test -c -o <dir>` with packages that
// have no test files exits 0 with `[no test files]`, and the ordinary command
// answers "every mutant survived" for the same tree — so the module scope
// answers the same: one compilation, no binaries, Built.
func TestModuleScopeRunnerBuildsWithoutAnyTestPackages(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the Go toolchain")
	}

	root := moduleFixture(t, map[string]string{
		"library/library.go": "package library\n\nfunc Value() int { return 1 }\n",
	})
	runner := NewModuleScope()

	outcome := runner.Test(fakerepository.NewTemporaryAt(root))

	assert.False(t, outcome.IsOk(), "nothing ran, so every mutant survived")
	assert.True(t, runner.Built())
	assert.Equal(t, 1, runner.Discoveries())
	assert.Equal(t, 2, runner.ToolchainStarts(), "one discovery plus the single batch invocation")
	assert.Equal(t, 1, runner.Compilations(), "measured: go test -c with untested packages exits 0")
	assert.Equal(t, 0, runner.PackageRuns())
}

func moduleFixture(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n\ngo 1.25\n"), 0o600))

	for name, source := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
		require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
	}

	return root
}

// TestModuleScopeRunnerCarriesAVerdictReason is the guard for the gap measured
// in docs/experiments/module-path-verdict-reason.md.
//
// A package test binary cannot emit `go test -json` — that flag belongs to the
// driver that starts the binary, not to the binary — so a module-path kill
// arrived as plain text, internal/verdict saw no stream, and the reason was
// Unknown. That is not a cosmetic loss: internal/confirminglaboratory re-runs a
// kill only when the reason is Assertion, so `--confirm-kills` silently never
// fired on the gated path.
func TestModuleScopeRunnerCarriesAVerdictReason(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the Go toolchain")
	}

	root := moduleFixture(t, map[string]string{
		"subject/subject.go": "package subject\n\nimport \"os\"\n\nfunc Changed() bool { return os.Getenv(\"DITTO_MUTANT\") != \"1\" }\n",
		"subject/subject_test.go": `package subject

import (
	"os"
	"testing"
)

func TestKilledByTheMutant(t *testing.T) {
	if os.Getenv("DITTO_MUTANT") == "1" {
		t.Fatal("the mutant was active, so this test failed")
	}
}
`,
	})
	runner := NewModuleScope()
	sandbox := fakerepository.NewTemporaryAt(root)

	if baseline := runner.Test(sandbox); baseline.IsOk() {
		t.Fatalf("the baseline was red, so nothing below is a mutant's verdict: %s", result.Output(baseline))
	}

	runner.Select(1)

	outcome := runner.Test(sandbox)
	require.True(t, outcome.IsOk(), "the selected mutant survived, so there is no kill to read a reason from")

	assert.Equal(t, verdict.Assertion, verdict.ReasonOf(result.Output(outcome)),
		"a module-path kill must carry the same reason the ordinary command carries, or --confirm-kills never re-runs one")

	assert.Equal(t, 1, runner.ConverterStarts(),
		"only a failing package needs converting; a green selection must start no converter at all")
}

func TestModuleScopeRunnerStartsNoConverterForAGreenSelection(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the Go toolchain")
	}

	root := moduleFixture(t, map[string]string{
		"subject/subject.go":      "package subject\n\nfunc Value() int { return 1 }\n",
		"subject/subject_test.go": "package subject\n\nimport \"testing\"\n\nfunc TestValue(t *testing.T) {}\n",
	})
	runner := NewModuleScope()
	sandbox := fakerepository.NewTemporaryAt(root)

	outcome := runner.Test(sandbox)

	assert.False(t, outcome.IsOk(), "the unselected baseline is green")
	assert.Equal(t, 0, runner.ConverterStarts(), "a green selection needs no reason, so it pays for no conversion")
}

// TestModuleScopeRunnerScopesToTheObservablePackages is the guard for
// docs/experiments/dependency-closure.md.
//
// A package test binary compiles its own package plus the transitive closure of
// what it imports, so a binary without the mutated package in that closure holds
// no code that can refer to anything the mutation changed. Running it can only
// cost time.
//
// This is not the package-only defect returning. That one ran the mutated
// package and nothing else, so a mutant only a dependent package could kill
// survived; this runs it plus everything that can reach it.
func TestModuleScopeRunnerScopesToTheObservablePackages(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the Go toolchain")
	}

	root := moduleFixture(t, map[string]string{
		"base/base.go": "package base\n\nfunc Covered(value, threshold int) bool { return value > threshold }\n",
		"mid/mid.go":   "package mid\n\nimport \"fixture/base\"\n\nfunc Reached(value, threshold int) bool { return base.Covered(value, threshold) }\n",
		// top reaches base through mid without ever naming it, which is the case
		// a closure built from direct imports alone would miss.
		"top/top.go": "package top\n\nimport \"fixture/mid\"\n\nfunc Through(value, threshold int) bool { return mid.Reached(value, threshold) }\n",
		// island imports nothing local and cannot observe anything outside itself.
		"island/island.go":      "package island\n\nfunc Value() int { return 1 }\n",
		"base/base_test.go":     "package base\n\nimport (\n\t\"os\"\n\t\"testing\"\n)\n\nfunc TestOwn(t *testing.T) {\n\tif os.Getenv(\"DITTO_MUTANT\") == \"1\" {\n\t\tt.Fatal(\"base killed it\")\n\t}\n}\n",
		"mid/mid_test.go":       "package mid\n\nimport (\n\t\"os\"\n\t\"testing\"\n)\n\nfunc TestDependent(t *testing.T) {\n\tif os.Getenv(\"DITTO_MUTANT\") == \"2\" {\n\t\tt.Fatal(\"only this dependent package kills the sentinel\")\n\t}\n}\n",
		"top/top_test.go":       "package top\n\nimport \"testing\"\n\nfunc TestThroughTheChain(t *testing.T) {}\n",
		"island/island_test.go": "package island\n\nimport \"testing\"\n\nfunc TestIsland(t *testing.T) {}\n",
	})
	runner := NewModuleScope()
	sandbox := fakerepository.NewTemporaryAt(root)

	if baseline := runner.Test(sandbox); baseline.IsOk() {
		t.Fatalf("the baseline was red: %s", result.Output(baseline))
	}

	assert.Equal(t, 4, runner.PackageRuns(), "unscoped, every package with tests runs once for the baseline")

	runner.ScopeTo("base")

	before := runner.PackageRuns()

	runner.Select(2)

	outcome := runner.Test(sandbox)
	require.True(t, outcome.IsOk(), "the sentinel must still be killed")

	assert.Equal(t, 3, runner.PackageRuns()-before,
		"only base, mid and top can observe a mutation in base; island must not run")
	assert.Equal(t, verdict.Assertion, verdict.ReasonOf(result.Output(outcome)))
}

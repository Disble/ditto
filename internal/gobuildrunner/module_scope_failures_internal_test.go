package gobuildrunner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Disble/ditto/internal/dittotesting/fakerepository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The controlled toolchain must exit before the Go test runner sees the
// toolchain arguments; otherwise it would run this package's tests recursively.
//
// TestMain rather than init, because this package forbids init functions and
// the self-exec stub is the same idea with the framework's own entry point: the
// variable is unset for every ordinary run, so m.Run is what decides there.
func TestMain(m *testing.M) {
	if os.Getenv("DITTO_MODULE_SCOPE_FAILURE_TOOLCHAIN") == "" {
		os.Exit(m.Run())
	}

	os.Exit(runFailureToolchain())
}

func runFailureToolchain() int {
	args := os.Args[1:]
	if len(args) == 0 {
		return 0
	}

	switch args[0] {
	case "list":
		if os.Getenv("DITTO_MODULE_SCOPE_FAILURE_EMPTY") != "" {
			// A module whose ./... matches nothing: the toolchain exits 0 with
			// no package records at all.
			return 0
		}

		listed := goListPackage{
			ImportPath:  "fixture/has_tests",
			Dir:         filepath.Join(os.Getenv("DITTO_MODULE_SCOPE_FAILURE_ROOT"), "has_tests"),
			TestGoFiles: []string{"has_tests_test.go"},
		}

		encoded, err := json.Marshal(listed)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)

			return 2
		}

		// Written rather than printed: this package forbids the fmt print family
		// so that a debug statement cannot become the product's output.
		_, _ = os.Stdout.Write(append(encoded, '\n'))
	case "test":
		// A successful compiler exit with no binary is the contract under test.
	}

	return 0
}

func TestModuleScopeRunnerDoesNotExpectOrRunABinaryForPackagesWithoutTests(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the Go toolchain")
	}

	root := moduleFixture(t, map[string]string{
		"library/library.go": "package library\n\nfunc Value() int { return 1 }\n",
		"checked/checked.go": "package checked\n",
		"checked/checked_test.go": `package checked

import "testing"

func TestChecked(t *testing.T) {}
`,
	})
	runner := NewModuleScope()

	outcome := runner.Test(fakerepository.NewTemporaryAt(root))

	assert.False(t, outcome.IsOk(), "the tested package passed")
	assert.True(t, runner.Built(), "a package without tests must not require a binary")
	assert.Equal(t, 1, runner.Discoveries(), "the complete module scope is discovered")
	assert.Equal(t, 1, runner.Compilations(), "the complete module scope is compiled once")
	assert.Equal(t, 1, runner.PackageRuns(), "only the package with tests starts a binary")
}

func TestModuleScopeRunnerReturnsGoListDiagnostics(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the Go toolchain")
	}

	root := moduleFixture(t, map[string]string{
		"broken/first.go":  "package first\n",
		"broken/second.go": "package second\n",
	})
	runner := NewModuleScope()

	outcome := runner.Test(fakerepository.NewTemporaryAt(root))

	assert.True(t, outcome.IsOk(), "discovery must fail closed")
	assert.False(t, runner.Built())
	assert.Equal(t, 1, runner.Discoveries())
	assert.Equal(t, 1, runner.ToolchainStarts())
	assert.Equal(t, 0, runner.Compilations())
	assert.Equal(t, 0, runner.PackageRuns())
	assert.Contains(t, outcome.String(), "found packages first")
	assert.Contains(t, outcome.String(), "second")
	assert.NotEqual(t, "ditto: discover module packages: exit status 1", strings.TrimSpace(outcome.String()))
}

func TestModuleScopeRunnerBuildFailureDoesNotRunPackages(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the Go toolchain")
	}

	root := moduleFixture(t, map[string]string{
		"broken/broken.go": "package broken\n\nfunc Broken() { this is not Go }\n",
		"broken/broken_test.go": `package broken

import "testing"

func TestBroken(t *testing.T) {}
`,
	})
	runner := NewModuleScope()

	outcome := runner.Test(fakerepository.NewTemporaryAt(root))

	assert.True(t, outcome.IsOk(), "a module build failure must fail closed")
	assert.False(t, runner.Built())
	assert.Equal(t, 1, runner.Compilations())
	assert.Equal(t, 0, runner.PackageRuns())
	assert.Contains(t, outcome.String(), "broken.go")
	assert.Contains(t, outcome.String(), "syntax error")
}

func TestModuleScopeRunnerSuccessfulBuildWithoutExpectedBinaryDoesNotRunPackages(t *testing.T) {
	root := t.TempDir()
	runner := NewModuleScope()
	runner.toolchain = moduleScopeFailureToolchain(t, root)

	outcome := runner.Test(fakerepository.NewTemporaryAt(root))

	assert.True(t, outcome.IsOk(), "a missing test binary must fail closed")
	assert.False(t, runner.Built())
	assert.Equal(t, 1, runner.Discoveries())
	assert.Equal(t, 1, runner.Compilations())
	assert.Equal(t, 0, runner.PackageRuns())
	assert.Contains(t, outcome.String(), "fixture/has_tests")
	assert.Contains(t, outcome.String(), moduleTestBinaryName("fixture/has_tests", runtime.GOOS))
}

// TestModuleScopeRunnerFailsClosedWhenDiscoveryYieldsNoPackages guards the
// empty-scope path: discovery that returns no module packages at all must fail
// closed with a named error, never reach `go test -c` with no package
// arguments.
func TestModuleScopeRunnerFailsClosedWhenDiscoveryYieldsNoPackages(t *testing.T) {
	root := t.TempDir()
	runner := NewModuleScope()
	runner.toolchain = moduleScopeFailureToolchain(t, root)
	t.Setenv("DITTO_MODULE_SCOPE_FAILURE_EMPTY", "1")

	outcome := runner.Test(fakerepository.NewTemporaryAt(root))

	assert.True(t, outcome.IsOk(), "an empty scope must fail closed")
	assert.False(t, runner.Built())
	assert.Equal(t, 1, runner.Discoveries())
	assert.Equal(t, 1, runner.ToolchainStarts(), "only the discovery runs; no compile invocation is started")
	assert.Equal(t, 0, runner.Compilations(), "no batch is compiled when nothing was discovered")
	assert.Equal(t, 0, runner.PackageRuns())
	assert.Contains(t, outcome.String(), "ditto: module scope discovered no packages")
}

func TestModuleScopeRunnerContinuesAfterARedBaselinePackage(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the Go toolchain")
	}

	root := moduleFixture(t, map[string]string{
		"alpha/alpha_test.go": `package alpha

import (
	"fmt"
	"os"
	"testing"
)

func TestFirstSortedPackageFails(t *testing.T) {
	fmt.Fprintln(os.Stdout, "alpha:0")
	if err := os.WriteFile("../order", []byte("alpha:0\n"), 0o600); err != nil { t.Fatal(err) }
	t.Fatal("alpha failed")
}
`,
		"omega/omega_test.go": `package omega

import (
	"fmt"
	"os"
	"testing"
)

func TestLaterSortedPackageStillRuns(t *testing.T) {
	fmt.Fprintln(os.Stdout, "omega:0")
	f, err := os.OpenFile("../order", os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil { t.Fatal(err) }
	defer f.Close()
	if _, err := f.WriteString("omega:0\n"); err != nil { t.Fatal(err) }
}
`,
	})
	runner := NewModuleScope()

	outcome := runner.Test(fakerepository.NewTemporaryAt(root))

	assert.True(t, outcome.IsOk(), "a red baseline must return failure")
	assert.True(t, runner.Built())
	assert.Equal(t, 2, runner.PackageRuns(), "the later package must run after the first fails")
	assert.Contains(t, outcome.String(), "alpha:0")
	assert.Contains(t, outcome.String(), "omega:0")

	order, err := os.ReadFile(filepath.Join(root, "order"))
	require.NoError(t, err)
	assert.Equal(t, "alpha:0\nomega:0\n", string(order), "exact per-package execution evidence")
}

func moduleScopeFailureToolchain(t *testing.T, root string) string {
	t.Helper()

	toolchain, err := os.Executable()
	require.NoError(t, err)
	t.Setenv("DITTO_MODULE_SCOPE_FAILURE_TOOLCHAIN", "1")
	t.Setenv("DITTO_MODULE_SCOPE_FAILURE_ROOT", root)

	return toolchain
}

// Shared environment stripping remains covered by
// internal/cmdtestrunner.TestCMDTestRunner's git-environment assertion. These
// module-scope tests keep that shared behavior out of a second internal test.

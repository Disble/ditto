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

func TestModuleScopeRunnerRejectsDuplicateBinaryNames(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the Go toolchain")
	}

	root := moduleFixture(t, map[string]string{
		"first/calc/calc.go":       "package calc\n",
		"first/calc/calc_test.go":  "package calc\nimport \"testing\"\nfunc TestOne(t *testing.T) {}\n",
		"second/calc/calc.go":      "package calc\n",
		"second/calc/calc_test.go": "package calc\nimport \"testing\"\nfunc TestTwo(t *testing.T) {}\n",
	})
	runner := NewModuleScope()

	outcome := runner.Test(fakerepository.NewTemporaryAt(root))

	assert.True(t, outcome.IsOk())
	assert.False(t, runner.Built())
	assert.Equal(t, 1, runner.Discoveries())
	assert.Equal(t, 1, runner.ToolchainStarts())
	assert.Equal(t, 0, runner.Compilations())
	assert.Equal(t, 1, runner.Selections())
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

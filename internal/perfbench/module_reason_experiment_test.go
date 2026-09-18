//go:build experiment

package perfbench_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Disble/ditto/internal/dittotesting/fakerepository"
	"github.com/Disble/ditto/internal/gobuildrunner"
	"github.com/Disble/ditto/internal/result"
	dittoverdict "github.com/Disble/ditto/internal/verdict"
)

// TestModulePathVerdictReason carries out docs/experiments/module-path-verdict-reason.md.
//
// Run it only from a complete disposable copy of this tree with .git omitted:
//
//	go test -tags=experiment ./internal/perfbench -run '^TestModulePathVerdictReason$' -count=1 -v
func TestModulePathVerdictReason(t *testing.T) {
	root := writeReasonFixture(t)

	t.Run("control: the ordinary -json command reports a reason", func(t *testing.T) {
		output := ordinaryJSONRun(t, root)

		t.Logf("ordinary output (first 400 bytes): %.400s", output)
		t.Logf("ordinary reason: %s", dittoverdict.ReasonOf(output))

		if got := dittoverdict.ReasonOf(output); got != dittoverdict.Assertion {
			t.Fatalf("CONTROL FAILED: the ordinary path reported %q, want %q; the instrument is wrong, not the module path", got, dittoverdict.Assertion)
		}
	})

	var moduleOutput string

	t.Run("module path: what a kill carries", func(t *testing.T) {
		moduleOutput = moduleScopeRun(t, root)

		t.Logf("module output (first 400 bytes): %.400s", moduleOutput)
		t.Logf("module reason: %s", dittoverdict.ReasonOf(moduleOutput))
	})

	t.Run("module path through test2json: the ceiling of the fix", func(t *testing.T) {
		converted := test2json(t, root, moduleOutput)

		t.Logf("converted output (first 400 bytes): %.400s", converted)
		t.Logf("converted reason: %s", dittoverdict.ReasonOf(converted))
	})

	t.Run("module path fails closed on a package that does not compile", func(t *testing.T) {
		broken := writeBrokenReasonFixture(t)
		runner := gobuildrunner.NewModuleScope()

		outcome := runner.Test(fakerepository.NewTemporaryAt(broken))

		t.Logf("broken fixture: built=%v reason=%s output=%.300s", runner.Built(), dittoverdict.ReasonOf(result.Output(outcome)), result.Output(outcome))

		if runner.Built() {
			t.Fatal("a module that does not compile must leave Built false")
		}
	})
}

// writeReasonFixture is a two-package module whose subject test fails exactly
// when the mutant the runner selects is active, so one fixture answers every
// mode through the one channel the product uses.
func writeReasonFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeExperimentFixtureFile(t, root, "go.mod", "module reasonfixture\n\ngo 1.25\n")
	writeExperimentFixtureFile(t, root, "subject/subject.go", `package subject

func Value() int { return 1 }
`)
	writeExperimentFixtureFile(t, root, "subject/subject_test.go", `package subject

import (
	"os"
	"testing"
)

func TestKilledByTheMutant(t *testing.T) {
	if os.Getenv("DITTO_MUTANT") == "1" {
		t.Fatal("the mutant was active, so this test failed")
	}
}
`)

	return root
}

func writeBrokenReasonFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeExperimentFixtureFile(t, root, "go.mod", "module brokenfixture\n\ngo 1.25\n")
	writeExperimentFixtureFile(t, root, "broken/broken.go", "package broken\n\nfunc Broken() { this is not Go }\n")
	writeExperimentFixtureFile(t, root, "broken/broken_test.go", "package broken\n\nimport \"testing\"\n\nfunc TestBroken(t *testing.T) {}\n")

	return root
}

func writeExperimentFixtureFile(t *testing.T, root, name, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(name))

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("create fixture directory for %s: %v", name, err)
	}

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture file %s: %v", name, err)
	}
}

// ordinaryJSONRun is the control: the command ditto actually configures by
// default, with the mutant active.
func ordinaryJSONRun(t *testing.T, root string) string {
	t.Helper()

	output, _ := experimentCommand(t, root, "go", "test", "-count=1", "-json", "./...")

	return output
}

// moduleScopeRun is the runner under test, answering the same mutant.
func moduleScopeRun(t *testing.T, root string) string {
	t.Helper()

	runner := gobuildrunner.NewModuleScope()
	sandbox := fakerepository.NewTemporaryAt(root)

	if baseline := runner.Test(sandbox); baseline.IsOk() {
		t.Fatalf("the module baseline was red, so nothing below is a mutant's verdict: %s", result.Output(baseline))
	}

	runner.Select(1)

	outcome := runner.Test(sandbox)
	if !outcome.IsOk() {
		t.Fatal("the module path reported the selected mutant as survived, so there is no kill to read a reason from")
	}

	return result.Output(outcome)
}

// test2json is the ceiling of the candidate fix: the technique measured, not yet
// the product.
func test2json(t *testing.T, root, binaryOutput string) string {
	t.Helper()

	command := exec.Command("go", "tool", "test2json", "-t", "-p", "reasonfixture/subject") //nolint:gosec,noctx // the fixed toolchain tool, measured only
	command.Dir = root
	command.Stdin = strings.NewReader(binaryOutput)

	converted, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go tool test2json failed: %v\n%s", err, converted)
	}

	return string(converted)
}

func experimentCommand(t *testing.T, root, name string, args ...string) (string, error) {
	t.Helper()

	command := exec.Command(name, args...) //nolint:gosec,noctx // fixture-controlled
	command.Dir = root
	command.Env = experimentEnvironmentWithMutant(os.Environ(), 1)

	output, err := command.CombinedOutput()

	return string(output), err
}

func experimentEnvironmentWithMutant(environment []string, mutant int) []string {
	inherited := []string{
		"GIT_DIR=", "GIT_INDEX_FILE=", "GIT_WORK_TREE=",
		"GIT_OBJECT_DIRECTORY=", "GIT_COMMON_DIR=", "DITTO_MUTANT=",
	}

	kept := make([]string, 0, len(environment)+1)

	for _, entry := range environment {
		if !hasExperimentPrefix(entry, inherited) {
			kept = append(kept, entry)
		}
	}

	return append(kept, "DITTO_MUTANT="+strconv.Itoa(mutant))
}

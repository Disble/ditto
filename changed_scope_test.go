package ditto_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Disble/ditto"
)

// These exercise the answers PlanChanged and RunChanged give when something is
// wrong, which is most of what they are: every branch below was a surviving
// mutant on ditto's own gate, because nothing had ever made those errors happen.
//
// They use a real repository rather than a double. The whole subject is what git
// says, and a fake that agrees with my reading of git proves my reading rather
// than the behaviour.

func repository(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	run(t, dir, "init")
	run(t, dir, "config", "user.email", "fixture@example.com")
	run(t, dir, "config", "user.name", "fixture")
	write(t, dir, "kept.go", "package fixture\n\nfunc Kept() int { return 1 }\n")
	run(t, dir, "add", "-A")
	run(t, dir, "commit", "-m", "base")
	run(t, dir, "tag", "base")

	return dir
}

func run(t *testing.T, dir string, args ...string) {
	t.Helper()

	command := exec.CommandContext(t.Context(), "git", args...)
	command.Dir = dir
	// Removed rather than blanked: git rejects an empty GIT_DIR outright, and
	// an inherited one would point the fixture at the real checkout.
	command.Env = withoutGitEnvironment(os.Environ())

	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

func TestPlanChangedReadsACommittedChange(t *testing.T) {
	dir := repository(t)

	write(t, dir, "added.go", "package fixture\n\nfunc Added(a, b int) bool { return a > b }\n")
	run(t, dir, "add", "-A")
	run(t, dir, "commit", "-m", "add")

	plan, err := ditto.PlanChanged(dir, "base", nil)
	if err != nil {
		t.Fatalf("planning: %v", err)
	}

	if !plan.Mutable() {
		t.Fatal("a committed Go change was not worth mutating")
	}

	if len(plan.Files) != 1 || plan.Files[0] != "added.go" {
		t.Fatalf("files = %v, want only added.go", plan.Files)
	}

	if !plan.Derived {
		t.Fatalf("the scope fell open to whole files: %s", plan.Reason)
	}
}

// A commit that changes no Go source is not a failure. It is a scope of nothing,
// and saying so is what lets a gate skip honestly rather than report a green it
// did not earn.
func TestPlanChangedIsEmptyWhenNoGoSourceMoved(t *testing.T) {
	dir := repository(t)

	write(t, dir, "readme.md", "# fixture\n")
	run(t, dir, "add", "-A")
	run(t, dir, "commit", "-m", "docs")

	plan, err := ditto.PlanChanged(dir, "base", nil)
	if err != nil {
		t.Fatalf("planning: %v", err)
	}

	if plan.Mutable() {
		t.Fatalf("a docs-only commit was reported as mutable: %v", plan.Files)
	}
}

func TestPlanChangedExcludesByPrefix(t *testing.T) {
	dir := repository(t)

	if err := os.Mkdir(filepath.Join(dir, "tools"), 0o750); err != nil {
		t.Fatalf("creating tools: %v", err)
	}

	write(t, dir, "tools/tool.go", "package tools\n\nfunc Tool(a, b int) bool { return a > b }\n")
	run(t, dir, "add", "-A")
	run(t, dir, "commit", "-m", "tool")

	plan, err := ditto.PlanChanged(dir, "base", []string{"tools/"})
	if err != nil {
		t.Fatalf("planning: %v", err)
	}

	if plan.Mutable() {
		t.Fatalf("an excluded prefix was still planned: %v", plan.Files)
	}
}

// A base that does not exist is an error rather than an empty scope. The two are
// the same exit code and opposite meanings: one is a change with nothing in it,
// the other is a question git could not answer.
func TestPlanChangedRefusesAnUnknownBase(t *testing.T) {
	_, err := ditto.PlanChanged(repository(t), "no-such-ref", nil)
	if err == nil {
		t.Fatal("an unknown base was accepted")
	}

	if !strings.Contains(err.Error(), "no-such-ref") {
		t.Fatalf("the error does not name the base: %v", err)
	}
}

func TestPlanChangedRefusesSomewhereThatIsNotARepository(t *testing.T) {
	if _, err := ditto.PlanChanged(t.TempDir(), "base", nil); err == nil {
		t.Fatal("a directory outside any repository was accepted")
	}
}

// RunChanged refuses a dirty checkout, and that refusal is the whole safety of
// reusing the index-backed sandbox: a range scope names bytes of HEAD, and those
// are the same bytes only while nothing is modified or staged.
func TestRunChangedRefusesADirtyCheckout(t *testing.T) {
	dir := repository(t)

	write(t, dir, "added.go", "package fixture\n\nfunc Added(a, b int) bool { return a > b }\n")
	run(t, dir, "add", "-A")
	run(t, dir, "commit", "-m", "add")
	write(t, dir, "kept.go", "package fixture\n\nfunc Kept() int { return 2 }\n")

	err := ditto.RunChanged(dir, "base", nil)
	if err == nil {
		t.Fatal("a dirty checkout was accepted")
	}

	if !strings.Contains(err.Error(), "kept.go") {
		t.Fatalf("the refusal does not name what is dirty: %v", err)
	}
}

// Nothing to mutate is nothing to do, and it is not an error. A gate that
// treated it as one would fail every docs-only commit.
func TestRunChangedDoesNothingWhenNothingChanged(t *testing.T) {
	dir := repository(t)

	write(t, dir, "readme.md", "# fixture\n")
	run(t, dir, "add", "-A")
	run(t, dir, "commit", "-m", "docs")

	if err := ditto.RunChanged(dir, "base", nil); err != nil {
		t.Fatalf("a docs-only commit was reported as a failure: %v", err)
	}
}

func TestRunChangedRefusesAnUnknownBase(t *testing.T) {
	if err := ditto.RunChanged(repository(t), "no-such-ref", nil); err == nil {
		t.Fatal("an unknown base was accepted")
	}
}

func TestRunChangedRefusesSomewhereThatIsNotARepository(t *testing.T) {
	if err := ditto.RunChanged(t.TempDir(), "base", nil); err == nil {
		t.Fatal("a directory outside any repository was accepted")
	}
}

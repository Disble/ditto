package ditto_test

import (
	"strings"
	"testing"

	"github.com/Disble/ditto"
	"github.com/Disble/ditto/internal/dittotesting"
)

// These exercise the answers PlanChanged and RunChanged give when something is
// wrong, which is most of what they are: every branch below was a surviving
// mutant on ditto's own gate, because nothing had ever made those errors happen.
//
// They use a real repository rather than a double. The whole subject is what git
// says, and a fake that agrees with my reading of git proves my reading rather
// than the behaviour.

func TestPlanChangedReadsACommittedChange(t *testing.T) {
	dir := dittotesting.GitRepository(t)

	dittotesting.WriteFile(t, dir, "added.go", "package fixture\n\nfunc Added(a, b int) bool { return a > b }\n")
	dittotesting.Git(t, dir, "add", "-A")
	dittotesting.Git(t, dir, "commit", "-m", "add")

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
	dir := dittotesting.GitRepository(t)

	dittotesting.WriteFile(t, dir, "readme.md", "# fixture\n")
	dittotesting.Git(t, dir, "add", "-A")
	dittotesting.Git(t, dir, "commit", "-m", "docs")

	plan, err := ditto.PlanChanged(dir, "base", nil)
	if err != nil {
		t.Fatalf("planning: %v", err)
	}

	if plan.Mutable() {
		t.Fatalf("a docs-only commit was reported as mutable: %v", plan.Files)
	}
}

func TestPlanChangedExcludesByPrefix(t *testing.T) {
	dir := dittotesting.GitRepository(t)

	dittotesting.WriteFile(t, dir, "tools/tool.go", "package tools\n\nfunc Tool(a, b int) bool { return a > b }\n")
	dittotesting.Git(t, dir, "add", "-A")
	dittotesting.Git(t, dir, "commit", "-m", "tool")

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
	_, err := ditto.PlanChanged(dittotesting.GitRepository(t), "no-such-ref", nil)
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

// RunChanged refuses a STAGED change, and that refusal is the whole safety of
// reusing the index-backed sandbox: the sandbox is written from the index while
// a range scope names bytes of HEAD, so the two agree while the index does.
//
// A worktree-only modification is a different thing and is deliberately allowed;
// it never reaches the sandbox, so it cannot move a verdict. That case is pinned
// in internal/staged rather than here, because allowing it here would run a real
// release.
func TestRunChangedRefusesAStagedChange(t *testing.T) {
	dir := dittotesting.GitRepositoryWithAChange(t)

	dittotesting.WriteFile(t, dir, "kept.go", "package fixture\n\nfunc Kept() int { return 2 }\n")
	dittotesting.Git(t, dir, "add", "kept.go")

	err := ditto.RunChanged(dir, "base", nil)
	if err == nil {
		t.Fatal("a staged change was accepted")
	}

	if !strings.Contains(err.Error(), "kept.go") {
		t.Fatalf("the refusal does not name what is staged: %v", err)
	}
}

// Nothing to mutate is nothing to do, and it is not an error. A gate that
// treated it as one would fail every docs-only commit.
func TestRunChangedDoesNothingWhenNothingChanged(t *testing.T) {
	dir := dittotesting.GitRepository(t)

	dittotesting.WriteFile(t, dir, "readme.md", "# fixture\n")
	dittotesting.Git(t, dir, "add", "-A")
	dittotesting.Git(t, dir, "commit", "-m", "docs")

	if err := ditto.RunChanged(dir, "base", nil); err != nil {
		t.Fatalf("a docs-only commit was reported as a failure: %v", err)
	}
}

func TestRunChangedRefusesAnUnknownBase(t *testing.T) {
	if err := ditto.RunChanged(dittotesting.GitRepository(t), "no-such-ref", nil); err == nil {
		t.Fatal("an unknown base was accepted")
	}
}

func TestRunChangedRefusesSomewhereThatIsNotARepository(t *testing.T) {
	if err := ditto.RunChanged(t.TempDir(), "base", nil); err == nil {
		t.Fatal("a directory outside any repository was accepted")
	}
}

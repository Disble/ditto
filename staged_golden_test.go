package ditto_test

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// thoroughHelper pins both boundaries, so a mutant that moves one is caught.
const thoroughHelper = `package fixture

func Cases() map[int]string {
	return map[int]string{100: "A", 90: "A", 89: "B", 70: "B", 69: "C", 0: "C"}
}
`

// weakHelper checks one value, so the same mutants survive. It is only ever
// written into the worktree and never staged: that is the whole case.
const weakHelper = `package fixture

func Cases() map[int]string {
	return map[int]string{100: "A"}
}
`

// TestStagedMeasuresTheIndexAndNotTheWorktree is the guard for the thing that
// makes `ditto staged` worth having inside ditto instead of in every consumer.
//
// Mutants come from the staged content, but the test command runs against a
// whole tree, and a tracked file left dirty in the worktree decides what the
// suite proves. Measured before this existed: pointed at the worktree, seven of
// eight verdicts moved — a score of 0.13 against 1.00 for the identical mutants
// of an identical file. A check over staged paths cannot see it, because the
// file that moved them was never staged.
//
// The control is the second half and is not decoration: the same repository read
// through the worktree MUST report survivors. If it does not, the fixture no
// longer contains the case and the first half is agreeing with nothing.
func TestStagedMeasuresTheIndexAndNotTheWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("runs a full release against a git fixture")
	}

	binary := buildCommand(t)
	repository := stagedFixture(t)
	helper := filepath.Join(repository, "helper.go")

	dirty := runOutput(t, repository, binary, "staged", "--cwd", repository,
		"--test-command", "go test -count=1 ./...", "--threshold", "0")

	// One variable, changed and changed back: the same staged release with the
	// unstaged file put back the way the index has it. If the worktree reached
	// the verdict, these two disagree.
	writeFile(t, helper, thoroughHelper)

	clean := runOutput(t, repository, binary, "staged", "--cwd", repository,
		"--test-command", "go test -count=1 ./...", "--threshold", "0")

	writeFile(t, helper, weakHelper)

	if survivorsIn(t, dirty) != survivorsIn(t, clean) {
		t.Fatalf("the staged verdict moved with an unstaged file: %d survivor(s) against %d\n--- dirty ---\n%s\n--- clean ---\n%s",
			survivorsIn(t, dirty), survivorsIn(t, clean), dirty, clean)
	}

	// The control, and it has to move. The same repository read through the
	// worktree must report survivors: the weakened helper is what makes the
	// boundary mutants live, and if it cannot do that here then the agreement
	// above is agreement with nothing.
	worktree := runOutput(t, repository, binary, "run", "--root", repository,
		"--test-command", "go test -count=1 ./...", "--threshold", "0")

	if survivorsIn(t, worktree) == 0 {
		t.Fatalf("the control did not move: the worktree reported no survivors either, "+
			"so this fixture cannot tell the two readings apart\n%s", worktree)
	}
}

// TestStagedRefusesAFileStagedAndEditedAtOnce keeps the other half of the
// contract: the bytes that were measured are the bytes that were scored.
func TestStagedRefusesAFileStagedAndEditedAtOnce(t *testing.T) {
	if testing.Short() {
		t.Skip("runs git")
	}

	binary := buildCommand(t)
	repository := stagedFixture(t)

	// alpha.go is already staged; editing it to something else is the case.
	writeFile(t, filepath.Join(repository, "alpha.go"), gradeSource("score >= 72"))

	run := command(t, repository, binary, "staged", "--cwd", repository, "--dry")

	output, err := run.CombinedOutput()
	if err == nil {
		t.Fatalf("a file staged and edited at once was accepted:\n%s", output)
	}

	if !strings.Contains(string(output), "staged and also edited in the worktree") {
		t.Fatalf("the refusal does not name the case:\n%s", output)
	}
}

func gradeSource(second string) string {
	return `package fixture

func Grade(score int) string {
	if score >= 90 {
		return "A"
	}
	if ` + second + ` {
		return "B"
	}
	return "C"
}
`
}

// stagedFixture is a repository with alpha.go staged and clean, and helper.go
// tracked, weakened in the worktree, and never staged.
func stagedFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	writeFile(t, filepath.Join(root, "go.mod"), "module stagedfixture\n\ngo 1.27\n")
	writeFile(t, filepath.Join(root, "alpha.go"), gradeSource("score >= 70"))
	writeFile(t, filepath.Join(root, "helper.go"), thoroughHelper)
	writeFile(t, filepath.Join(root, "alpha_test.go"), `package fixture

import "testing"

func TestGrade(t *testing.T) {
	for score, want := range Cases() {
		if got := Grade(score); got != want {
			t.Errorf("Grade(%d) = %q, want %q", score, got, want)
		}
	}
}
`)

	git(t, root, "init")
	git(t, root, "config", "user.email", "fixture@example.invalid")
	git(t, root, "config", "user.name", "Staged Fixture")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "fixture")

	// The staged change: one line of alpha.go, staged and left clean.
	writeFile(t, filepath.Join(root, "alpha.go"), gradeSource("score > 69"))
	git(t, root, "add", "alpha.go")

	// The case: a tracked file the scope does not name, weakened in the
	// worktree and never staged.
	writeFile(t, filepath.Join(root, "helper.go"), weakHelper)

	return root
}

func git(t *testing.T, root string, args ...string) {
	t.Helper()

	output, err := command(t, root, "git", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func buildCommand(t *testing.T) string {
	t.Helper()

	binary := filepath.Join(t.TempDir(), "ditto"+binarySuffix())

	output, err := command(t, moduleRoot(t), "go", "build", "-o", binary, "./cmd/ditto").CombinedOutput()
	if err != nil {
		t.Fatalf("building the command: %v\n%s", err, output)
	}

	return binary
}

func runOutput(t *testing.T, dir, binary string, args ...string) string {
	t.Helper()

	output, err := command(t, dir, binary, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", binary, args, err, output)
	}

	return strings.ReplaceAll(string(output), "\r\n", "\n")
}

var survivedPattern = regexp.MustCompile(`Survived:\s+(\d+)`)

// survivorsIn reads the report rather than the exit code, because the number is
// the point and an exit code cannot carry it. An absent line is a failure: a run
// that printed no summary and a run that found nothing are different answers.
func survivorsIn(t *testing.T, output string) int {
	t.Helper()

	matches := survivedPattern.FindStringSubmatch(output)
	if matches == nil {
		t.Fatalf("the run printed no summary:\n%s", output)
	}

	survived, err := strconv.Atoi(matches[1])
	if err != nil {
		t.Fatalf("the summary said %q survived: %v", matches[1], err)
	}

	return survived
}

package staged

import (
	"errors"
	"strings"
	"testing"
)

// scriptedGit answers whichever command it recognises by its argument shape, so
// a test can say what git would have said without a repository on disk.
type scriptedGit struct {
	answers map[string]string
	seen    []string
}

func (g *scriptedGit) Output(_, _ string, args ...string) ([]byte, error) {
	joined := strings.Join(args, " ")
	g.seen = append(g.seen, joined)

	for shape, answer := range g.answers {
		if strings.Contains(joined, shape) {
			return []byte(answer), nil
		}
	}

	return nil, errors.New("scriptedGit: nothing scripted for: " + joined)
}

func scripted(answers map[string]string) (*Repository, *scriptedGit) {
	git := &scriptedGit{answers: answers}

	return &Repository{root: ".", runner: git}, git
}

// TestChangedFilesReadsARangeRatherThanTheIndex is backlog entry 21's answer.
//
// The staged scope reads `--cached`, which is empty on a CI checkout: nothing is
// staged after a push, so a gate pointed at it would skip and report a green
// that measured nothing. A range says the same thing about a change that has
// already been committed.
func TestChangedFilesReadsARangeRatherThanTheIndex(t *testing.T) {
	t.Parallel()

	repository, git := scripted(map[string]string{
		"diff --name-only": "internal/thing/thing.go\x00internal/thing/thing_test.go\x00readme.md\x00",
	})

	files, err := repository.ChangedFiles("v0.7.0", []string{"testdata/"})
	if err != nil {
		t.Fatalf("listing the changed files: %v", err)
	}

	if len(files) != 1 || files[0] != "internal/thing/thing.go" {
		t.Fatalf("changed files = %v, want only the non-test Go source", files)
	}

	// The three-dot form asks what the branch added rather than what the base
	// also did, which is what makes the scope the CHANGE and not the drift
	// between two lines of work.
	if !strings.Contains(git.seen[0], "v0.7.0...HEAD") {
		t.Fatalf("git was asked %q, want a three-dot range against the base", git.seen[0])
	}
}

// TestChangedScopeOfDerivesByteRanges holds the property the whole feature rests
// on: a scope that is not derived is a scope that mutates whole files, and on a
// repository-sized checkout that is the bill this exists to avoid.
func TestChangedScopeOfDerivesByteRanges(t *testing.T) {
	t.Parallel()

	content := "package thing\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n"
	diff := "@@ -4 +4 @@\n-\treturn a - b\n+\treturn a + b\n"

	repository, _ := scripted(map[string]string{
		"diff --no-ext-diff": diff,
		"show":               content,
	})

	scope, err := repository.ChangedScopeOf("v0.7.0", []string{"thing.go"})
	if err != nil {
		t.Fatalf("reading the changed scope: %v", err)
	}

	if !scope.Derived {
		t.Fatalf("scope fell open to whole files: %s", scope.Reason)
	}

	if len(scope.Ranges["thing.go"]) != 1 {
		t.Fatalf("ranges = %v, want one span covering the changed line", scope.Ranges)
	}
}

// TestRequireNothingStagedGuardsTheRightTree is what makes reusing the
// index-backed sandbox honest, and the precise condition is the point.
//
// The sandbox is `git checkout-index --all`, so it holds the INDEX; a range
// scope names bytes of HEAD. Those agree while the index agrees with HEAD, which
// is narrower than a clean worktree — and the difference is not academic. Written
// as a clean-worktree check, this refused ditto's own CI in fifty-one seconds,
// because the Devbox install step modifies a tracked lockfile that has nothing
// to do with any change.
func TestRequireNothingStagedGuardsTheRightTree(t *testing.T) {
	t.Parallel()

	t.Run("refuses a staged change, because the sandbox would carry it", func(t *testing.T) {
		t.Parallel()

		repository, _ := scripted(map[string]string{"diff --cached --name-only": "internal/thing/thing.go\n"})

		err := repository.RequireNothingStaged()
		if err == nil {
			t.Fatal("a staged change was accepted")
		}

		if !strings.Contains(err.Error(), "thing.go") {
			t.Fatalf("the refusal does not name what is staged: %v", err)
		}
	})

	t.Run("allows a worktree modification, because the sandbox never sees it", func(t *testing.T) {
		t.Parallel()

		repository, _ := scripted(map[string]string{"diff --cached --name-only": ""})

		if err := repository.RequireNothingStaged(); err != nil {
			t.Fatalf("a worktree-only modification was refused: %v", err)
		}
	})
}

// TestChangedScopeFailsOpenRatherThanGuessing covers both fail-open branches.
//
// Mutating too much is a cost; mutating the wrong bytes is a wrong answer. So a
// diff that yields no usable range widens to whole files and says why, rather
// than producing a scope nobody can trust.
func TestChangedScopeFailsOpenRatherThanGuessing(t *testing.T) {
	t.Parallel()

	content := "package thing\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n"

	t.Run("when the diff carries no hunk at all", func(t *testing.T) {
		t.Parallel()

		repository, _ := scripted(map[string]string{
			"diff --no-ext-diff": "diff --git a/thing.go b/thing.go\nsimilarity index 100%\n",
			"show":               content,
		})

		scope, err := repository.ChangedScopeOf("v0.7.0", []string{"thing.go"})
		if err != nil {
			t.Fatalf("reading the changed scope: %v", err)
		}

		if scope.Derived {
			t.Fatal("a diff with no hunk produced a derived scope")
		}

		if scope.Ranges["thing.go"] != nil {
			t.Fatalf("a failed-open scope carries ranges: %v", scope.Ranges)
		}

		// The reason is asserted, not just its absence. There are two ways to
		// fail open here and they mean different things -- a diff nobody could
		// parse, and a range that missed the file -- and a reader chasing a
		// widened scope needs to know which one happened.
		if !strings.Contains(scope.Reason, "no usable range") {
			t.Fatalf("the reason names the wrong failure: %q", scope.Reason)
		}
	})

	t.Run("when the hunk names lines the committed content does not have", func(t *testing.T) {
		t.Parallel()

		repository, _ := scripted(map[string]string{
			"diff --no-ext-diff": "@@ -900 +900 @@\n-\tgone\n+\talso gone\n",
			"show":               content,
		})

		scope, err := repository.ChangedScopeOf("v0.7.0", []string{"thing.go"})
		if err != nil {
			t.Fatalf("reading the changed scope: %v", err)
		}

		if scope.Derived {
			t.Fatal("a range past the end of the file produced a derived scope")
		}

		if !strings.Contains(scope.Reason, "committed bytes") {
			t.Fatalf("the reason does not say what went wrong: %q", scope.Reason)
		}
	})
}

// TestChangedFilesSurvivesAnEmptyRange is the docs-only commit: no Go source
// moved, which is a scope of nothing rather than a failure.
func TestChangedFilesSurvivesAnEmptyRange(t *testing.T) {
	t.Parallel()

	repository, _ := scripted(map[string]string{"diff --name-only": ""})

	files, err := repository.ChangedFiles("v0.7.0", nil)
	if err != nil {
		t.Fatalf("listing the changed files: %v", err)
	}

	if len(files) != 0 {
		t.Fatalf("files = %v, want none", files)
	}
}

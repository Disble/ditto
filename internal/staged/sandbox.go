package staged

import (
	"fmt"
	"os"
)

// Sandbox is a checkout of the index, and the thing a release must be pointed at.
//
// The measurement that put this here: with a release pointed at the worktree
// instead, and one tracked file left dirty and unstaged, seven of eight verdicts
// moved — 0.13 against 1.00 for the identical eight mutants of an identical
// file. RejectPartial does not cover it, because the file that moved them was
// never staged and a check over staged paths never looks at it.
//
// `git checkout-index` also happens to be the cheap way to do it: it writes
// exactly the staged content and nothing else, with no second walk of the tree.
type Sandbox struct {
	Root    string
	cleanup func()
}

// Close removes the sandbox. It is safe to call more than once.
func (s *Sandbox) Close() {
	if s.cleanup == nil {
		return
	}

	s.cleanup()
	s.cleanup = nil
}

// Materialize writes the index into a temporary directory.
func (r *Repository) Materialize() (*Sandbox, error) {
	directory, err := os.MkdirTemp("", "ditto-staged-")
	if err != nil {
		return nil, fmt.Errorf("creating the staged sandbox: %w", err)
	}

	cleanup := func() { _ = os.RemoveAll(directory) }

	// The trailing separator is required: git treats --prefix as a literal
	// string prepended to each path, not as a directory to write into.
	if _, err := r.git("checkout-index", "--all", "--prefix="+directory+string(os.PathSeparator)); err != nil {
		cleanup()

		return nil, fmt.Errorf("materialising the index: %w", err)
	}

	return &Sandbox{Root: directory, cleanup: cleanup}, nil
}

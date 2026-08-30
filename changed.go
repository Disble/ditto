package ditto

import (
	"fmt"

	"github.com/Disble/ditto/internal/staged"
)

// PlanChanged answers what a committed change justifies, and changes nothing.
//
// It is PlanStaged's question asked of a range instead of the index, and it
// exists because a gate cannot ask the index anything. On a CI checkout nothing
// is staged: a run pointed at the staged scope skips, reports success, and
// measures nothing — which is the one result this repository refuses to call
// green. The change is still there. It is just already committed.
//
// The scope is `base...HEAD`, the diff against their merge base, so a base that
// has moved on since the change was written does not drag somebody else's
// commits into the bill.
func PlanChanged(directory, baseRef string, excludePrefixes []string) (StagedPlan, error) {
	repository, err := staged.New(staged.OSRunner{}, directory)
	if err != nil {
		return StagedPlan{}, fmt.Errorf("reading the repository: %w", err)
	}

	files, err := repository.ChangedFiles(baseRef, excludePrefixes)
	if err != nil {
		return StagedPlan{}, fmt.Errorf("reading the changed files: %w", err)
	}

	plan := StagedPlan{Root: repository.Root(), Files: files, Ranges: map[string][]Range{}}
	if len(files) == 0 {
		return plan, nil
	}

	scope, err := repository.ChangedScopeOf(baseRef, files)
	if err != nil {
		return StagedPlan{}, fmt.Errorf("reading the changed scope: %w", err)
	}

	plan.Ranges = rangesFrom(scope.Ranges)
	plan.Derived = scope.Derived
	plan.Reason = scope.Reason

	return plan, nil
}

// RunChanged mutates exactly what a committed change justifies.
//
// It refuses a checkout whose index has moved away from HEAD, and that refusal
// is the whole safety of the thing. The sandbox is written from the INDEX and a
// range scope names bytes of HEAD; those agree exactly while the index agrees
// with HEAD. Scoping against one tree and mutating another is the defect already
// measured on a fixture built for it — seven of eight verdicts moved — and it is
// silent, which is why this stops rather than warns.
//
// A worktree modification is not that. It is never written into the sandbox, so
// it cannot move a verdict, and refusing it only stops runs that would have been
// correct.
//
// Everything below the scope is the staged path unchanged: the same sandbox, the
// same `.ditto.json` for what git does not carry, the same notice when the diff
// could not be turned into ranges.
func RunChanged(directory, baseRef string, excludePrefixes []string, options ...Option) error {
	plan, err := PlanChanged(directory, baseRef, excludePrefixes)
	if err != nil {
		return err
	}

	if !plan.Mutable() {
		return nil
	}

	repository, err := staged.New(staged.OSRunner{}, directory)
	if err != nil {
		return fmt.Errorf("reading the repository: %w", err)
	}

	if err := repository.RequireNothingStaged(); err != nil {
		return fmt.Errorf("checking the checkout: %w", err)
	}

	return runInSandbox(directory, plan, options)
}

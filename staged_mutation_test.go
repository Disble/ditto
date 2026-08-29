//go:build mutation

package ditto_test

import (
	"testing"

	"github.com/Disble/ditto"
)

// TestStagedMutation is the gate for a change rather than for the repository.
//
// TestMutation asks the repository-sized question -- 727 mutants -- and then
// runs out of its thirty minutes at 424 of them, measured three times. Gating
// removes 54% of the compilations and does not close it; cutting the suite the
// mutant is judged by, 46%, moved the gate by 0.5% because `-failfast` already
// stops a killed mutant at its first failing test. Both levers are spent, and
// the bill is the wrong size rather than badly paid.
//
// Ditto's own answer to that is the staged scope: mutate what the change
// touched. WithChangedRanges is recorded at 4 laboratory runs where the whole
// fixture charges 48, and dharness has run its gate this way from the start.
//
// It skips when nothing is staged, because a scope of nothing is not a failure
// -- it is a commit that changed no Go source, and there is nothing to judge.
func TestStagedMutation(t *testing.T) {
	plan, err := ditto.PlanStaged(".", []string{"testdata/"})
	if err != nil {
		t.Fatalf("reading the staged change: %v", err)
	}

	if !plan.Mutable() {
		t.Skip("nothing staged is worth mutating")
	}

	t.Logf("staged scope: %d file(s), %d with byte ranges", len(plan.Files), len(plan.Ranges))

	if notice := plan.ScopeNotice(); notice != "" {
		t.Log(notice)
	}

	// RunStaged rather than Release: Release reads the WORKTREE, and a staged
	// gate has to read the INDEX. Measured on a fixture built for it -- against
	// the worktree, with one tracked file left dirty and unstaged, seven of
	// eight verdicts moved. Scoping correctly and then measuring the wrong bytes
	// would be the same defect wearing the fix's clothes.
	if err := ditto.RunStaged(".", []string{"testdata/"},
		ditto.ForceColors(),
		ditto.WithTestCommand(makeCommand(t)+" test.failfast MAKEFLAGS="),
		ditto.WithMinimumThreshold(0.5),
		ditto.Parallel(),
		ditto.Gated(),
	); err != nil {
		t.Fatal(err)
	}
}

//go:build mutation

package ditto_test

import (
	"os"
	"testing"

	"github.com/Disble/ditto"
)

// baseRefVariable names the ref the gate measures against.
//
// There is no default that is right everywhere: on a CI checkout the useful base
// is the last release, on a branch it is the trunk, and guessing wrong is either
// a bill nobody asked for or a scope of nothing. So it is named, and the gate
// says plainly when it was not.
const baseRefVariable = "DITTO_GATE_BASE"

// TestChangedMutation is the gate that finishes.
//
// TestMutation asks the repository-sized question — 736 mutants — and dies at
// its thirty minutes having reached about 424 of them, measured four times now.
// Both levers are spent: gating removes 54% of the compilations and does not
// close it, and cutting the suite the mutant is judged by, by 46%, moved the
// gate by 0.5% because `-failfast` already stops a killed mutant at its first
// failing test. The bill is the wrong SIZE rather than badly paid, and backlog
// entry 21 wrote down the answer without building it: ditto's own answer to a
// repository-sized bill is to mutate what the change touched.
//
// TestStagedMutation cannot be that gate. It reads the index, and on a CI
// checkout nothing is staged — so it skips, reports success, and measures
// nothing. That is the shape of failure this repository refuses, and it is why
// the range scope had to exist before the gate could move.
func TestChangedMutation(t *testing.T) {
	base := os.Getenv(baseRefVariable)
	if base == "" {
		t.Skipf("set %s to the ref this change is measured against, for example a release tag", baseRefVariable)
	}

	plan, err := ditto.PlanChanged(".", base, []string{"testdata/"})
	if err != nil {
		t.Fatalf("reading the change since %s: %v", base, err)
	}

	// Said whether or not there is anything to do, because a gate that reports
	// success has to say what it measured. "Nothing changed" and "the scope was
	// never read" produce the same exit code and are not the same result.
	t.Logf("scope since %s: %d file(s), %d with byte ranges", base, len(plan.Files), len(plan.Ranges))

	if !plan.Mutable() {
		t.Skipf("nothing changed since %s is worth mutating", base)
	}

	for _, file := range plan.Files {
		t.Logf("  %s: %d range(s)", file, len(plan.Ranges[file]))
	}

	if notice := plan.ScopeNotice(); notice != "" {
		t.Log(notice)
	}

	if err := ditto.RunChanged(".", base, []string{"testdata/"},
		ditto.ForceColors(),
		ditto.WithTestCommand(makeCommand(t)+" test.failfast MAKEFLAGS="),
		ditto.WithMinimumThreshold(0.5),
		ditto.Parallel(),
		ditto.Gated(),
	); err != nil {
		t.Fatal(err)
	}
}

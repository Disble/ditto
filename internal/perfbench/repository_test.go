//go:build livetree

// This counter reads the repository it is compiled inside, and that is what
// makes it the only test here that cannot run with the others.
//
// Measured: ditto's sandbox carries every file but .git (fsrepository.go), and
// each mutant's test command is `./...`, so an untagged version of this test
// runs INSIDE every mutant's sandbox and reads the MUTATED tree. Any mutation
// that changes the number of mutable sites then moves the count, the assertion
// fails, make exits non-zero, and ditto reads that as a killed mutant.
//
// Measured end to end on `internal/schemata/instrument.go:147:3 -> Range Break`,
// a mutant that survives on its own: with this file untagged the suite reported
// `REGRESSION mutantsPerReleaseOnThisRepository: 661, baseline 660 (+1)` and 7
// failures; without it, `DONE 368 tests, 10 skipped` and the mutant survived.
// 66 of the repository's 660 mutants are Range Break, so up to a tenth of the
// run was answerable this way.
//
// The tag is `livetree` and not `mutation`: `make test.mutation` runs the root
// package alone, so a mutation-tagged counter would never run at all. `make
// test.counters` runs this one, and the gate runs that.
//
// See docs/experiments/a-counter-that-answers-for-itself.md.

package perfbench_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/dittotesting/fakereporter"
	"github.com/Disble/ditto/internal/fsrepository"
	"github.com/Disble/ditto/internal/ignoredrepository"
)

// repositoryRoot is ditto's own tree. The fixture counters below it measure
// selection on six synthetic files; this one measures the repository the gate
// actually runs against, which is the number that grew unwatched.
const repositoryRoot = "../.."

// gateExclusions are the patterns ditto_mutation_test.go passes. The counter is
// meaningless unless it is scoped exactly like the run it speaks for.
func gateExclusions(t *testing.T) []*regexp.Regexp {
	t.Helper()

	return []*regexp.Regexp{regexp.MustCompile(`(^release\.go$|testdata\/.*)`)}
}

// TestCounterMutantsPerReleaseOnThisRepository records what one full run of the
// gate has to pay for.
//
// perfbench/doc.go names this number in prose -- "how many mutants a scope
// produces" -- and nothing measured it. Every recorded counter is against the
// synthetic fixture, which does not change when the repository does, so the
// ratchet stayed green while the real cost grew 43% between 2026-08-15 and
// 2026-08-28 and took the gate past its timeout.
//
// It costs no test command and no sandbox: countingLaboratory answers every
// mutant without running anything, so this is selection only.
func TestCounterMutantsPerReleaseOnThisRepository(t *testing.T) {
	t.Parallel()

	root, err := filepath.Abs(repositoryRoot)
	if err != nil {
		t.Fatalf("resolving the repository root: %v", err)
	}

	before := treeFingerprint(t, root)

	laboratory := &countingLaboratory{}
	repository := ignoredrepository.New(gateExclusions(t), fsrepository.New(root))

	ditto.New(repository, laboratory, fakereporter.New()).Release(defaultViruses()...)

	// H3: counting is not running. A counter that pays the cost it measures is
	// not a counter, and one that writes into the tree is the defect AGENTS.md
	// forbids.
	if after := treeFingerprint(t, root); after != before {
		t.Fatalf("the count wrote into the repository: %d entries before, %d after", before, after)
	}

	assertCounter(t, "mutantsPerReleaseOnThisRepository", laboratory.calls)
}

// treeFingerprint counts the entries under root, so a run that writes into the
// repository it is measuring cannot pass quietly.
func treeFingerprint(t *testing.T, root string) int {
	t.Helper()

	var entries int

	err := filepath.WalkDir(root, func(_ string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		entries++

		return nil
	})
	if err != nil {
		t.Fatalf("walking the repository: %v", err)
	}

	return entries
}

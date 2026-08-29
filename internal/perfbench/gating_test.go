//go:build livetree

package perfbench_test

import (
	"path/filepath"
	"testing"

	"github.com/Disble/ditto/internal/fsrepository"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/ignoredrepository"
	"github.com/Disble/ditto/internal/schemata"
)

// TestGatingOnThisRepository answers what turning Gated() on would cost, in
// exact integers, without running one test command.
//
// The gated path compiles once per file; the ordinary path compiles once per
// mutant. Whether that is worth turning on is a question about COMPILATIONS, and
// a compilation is an integer. Wall clock is reported here and never gated --
// see doc.go -- and the counters this reads are the ones the gated path already
// keeps and nothing outside its own package has ever read.
//
// docs/experiments/turning-gating-on.md
func TestGatingOnThisRepository(t *testing.T) {
	t.Parallel()

	root, err := filepath.Abs(repositoryRoot)
	if err != nil {
		t.Fatalf("resolving the repository root: %v", err)
	}

	repository := ignoredrepository.New(gateExclusions(t), fsrepository.New(root))

	var mutants, gated, filesWithGating, files int

	for _, source := range repository.ListGoSourceFiles() {
		incubated := source.Incubate(defaultViruses()...)
		if len(incubated) == 0 {
			continue
		}

		files++

		mutated := make([]*gomutatedfile.GoMutatedFile, 0, len(incubated))
		for _, one := range incubated {
			mutated = append(mutated, one.Mutate())
		}

		raw := make([][]byte, 0, len(mutated))
		for _, one := range mutated {
			raw = append(raw, one.Mutated())
		}

		planned := schemata.Plan(mutated[0].Source(), raw)

		inThisFile := 0

		for _, id := range planned.Selector {
			if id != 0 {
				inThisFile++
			}
		}

		mutants += len(incubated)
		gated += inThisFile

		if inThisFile > 0 {
			filesWithGating++
		}
	}

	// Ungated: one compilation per mutant. Gated: one per file that has any
	// gated mutant, plus one for each mutant that still falls back.
	ungated := mutants
	withGating := filesWithGating + (mutants - gated)

	t.Logf("source files with mutants : %d", files)
	t.Logf("mutants                   : %d", mutants)
	t.Logf("gated                     : %d (%.1f%%)", gated, 100*float64(gated)/float64(mutants))
	t.Logf("compilations ungated      : %d", ungated)
	t.Logf("compilations with gating  : %d", withGating)
	t.Logf("removed                   : %d (%.1f%%)", ungated-withGating,
		100*float64(ungated-withGating)/float64(ungated))
}

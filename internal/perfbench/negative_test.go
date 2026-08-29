//go:build livetree

package perfbench_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Disble/ditto/internal/fsrepository"
	"github.com/Disble/ditto/internal/ignoredrepository"
)

// TestHowManyMutantsGoNegative counts the mutants Integer Decrement produces by
// turning a literal 0 into -1.
//
// In Go a negative literal is a unary expression, not a BasicLit, so 0 is the
// only value this virus can take below zero -- and `index -1 must not be
// negative` is the single largest cause of non-compiling mutants measured on a
// real repository: 42 of 94. docs/experiments/false-kills.md.
func TestHowManyMutantsGoNegative(t *testing.T) {
	t.Parallel()

	root, err := filepath.Abs(repositoryRoot)
	if err != nil {
		t.Fatalf("resolving the repository root: %v", err)
	}

	repository := ignoredrepository.New(gateExclusions(t), fsrepository.New(root))

	var decrements, negatives, total int

	for _, source := range repository.ListGoSourceFiles() {
		for _, one := range source.Incubate(defaultViruses()...) {
			mutant := one.Mutate()
			total++

			if !strings.Contains(mutant.Label(), "Integer Decrement") {
				continue
			}

			decrements++

			if strings.Contains(mutant.Change(), "-1") {
				negatives++
			}
		}
	}

	t.Logf("mutants total          : %d", total)
	t.Logf("Integer Decrement      : %d", decrements)
	t.Logf("of those, produce -1   : %d (%.1f%% of all mutants)",
		negatives, 100*float64(negatives)/float64(total))
}

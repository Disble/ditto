package ditto

import (
	"testing"

	"github.com/Disble/ditto/internal/dittotesting/fakereporter"
)

// TestScoredIsUnknownRatherThanZero is the distinction the whole empty-scope fix
// rests on, and the one its mutants found unguarded.
//
// A reporter that cannot answer must read as UNKNOWN, not as zero. Zero means
// "the scope produced nothing to judge", and returning it for "I could not tell"
// would turn every genuinely failing run behind such a reporter into a silent
// success. The sentinel is -1 because a count cannot be negative, so no real
// answer collides with it.
func TestScoredIsUnknownRatherThanZero(t *testing.T) {
	t.Parallel()

	unreadable := &release{reporter: fakereporter.Countless{}}
	if got := unreadable.scored(); got != -1 {
		t.Fatalf("a reporter that cannot count scored %d, want -1", got)
	}

	for _, total := range []int{0, 1, 47} {
		readable := &release{reporter: fakereporter.Counting{Scored: total}}
		if got := readable.scored(); got != total {
			t.Fatalf("scored() = %d, want %d", got, total)
		}
	}
}

// TestAnUnreadableCountIsNotAnEmptyScope is the behaviour that distinction
// protects: a failing run whose count cannot be read is still a failing run.
func TestAnUnreadableCountIsNotAnEmptyScope(t *testing.T) {
	t.Parallel()

	rel := &release{
		opts:     Options{MinimumThreshold: 0.5},
		reporter: fakereporter.Countless{},
	}

	if rel.scored() == 0 {
		t.Fatal("an unreadable count read as an empty scope, which would report a failing run as nothing to do")
	}
}

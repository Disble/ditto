//go:build mutation

package goldenproject_test

import (
	"os"
	"testing"

	"github.com/Disble/ditto"
)

// TestMutation is the release the golden test records. The threshold is zero so
// the run reports its score instead of failing on it: what is being pinned here
// is which mutants live and which die, not whether the fixture is well tested.
func TestMutation(t *testing.T) {
	options := []ditto.Option{
		ditto.WithRepositoryRoot("."),
		ditto.WithTestCommand("go test -count=1 ./calc"),
		ditto.WithMinimumThreshold(0.0),
	}

	// The same run, from one compilation instead of one per mutant. The golden
	// it is compared against is the same file either way: if the gated path said
	// anything different, that is the whole point of having a golden.
	if os.Getenv("DITTO_GOLDEN_GATED") == "1" {
		options = append(options, ditto.Gated())
	}

	ditto.Release(t, options...)
}

//go:build mutation

package goldenproject_test

import (
	"testing"

	"github.com/Disble/ditto"
)

// TestMutation is the release the golden test records. The threshold is zero so
// the run reports its score instead of failing on it: what is being pinned here
// is which mutants live and which die, not whether the fixture is well tested.
func TestMutation(t *testing.T) {
	ditto.Release(
		t,
		ditto.WithRepositoryRoot("."),
		ditto.WithTestCommand("go test -count=1 ./calc"),
		ditto.WithMinimumThreshold(0.0),
	)
}

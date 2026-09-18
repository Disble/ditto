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
	// The default Go module scope, spelled out. Gated() only replaces the
	// command shapes whose complete execution plan has been measured, and a
	// package-local scope such as `./calc` is deliberately not one of them:
	// optimizing it would answer a smaller question than the caller asked,
	// which is how the package-only path came to miss a cross-package kill.
	// This fixture therefore names the scope Gated() is allowed to replace, so
	// the run below still proves that gating engages at all.
	//
	// DITTO_GOLDEN_PACKAGE_ONLY asks for the other half of the same contract: a
	// command whose scope module scope may not replace has to fall back, and the
	// report has to say `none` rather than quietly optimize something else.
	testCommand := "go test -count=1 ./..."
	if os.Getenv("DITTO_GOLDEN_PACKAGE_ONLY") == "1" {
		testCommand = "go test -count=1 ./calc"
	}

	options := []ditto.Option{
		ditto.WithRepositoryRoot("."),
		ditto.WithTestCommand(testCommand),
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

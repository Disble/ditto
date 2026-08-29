//go:build mutation

package ditto_test

import (
	"testing"

	"github.com/Disble/ditto"
)

func TestMutation(t *testing.T) {
	ditto.Release(
		t,
		ditto.ForceColors(),
		ditto.WithRepositoryRoot("."),
		ditto.WithTestCommand("make test.failfast MAKEFLAGS="),
		ditto.WithMinimumThreshold(0.5),
		ditto.Parallel(),
		// Measured before turning on, twice. Gating removes 394 of this
		// repository's 729 compilations -- 54% -- because 60.1% of its mutants
		// can be expressed as a runtime branch. And it changes no verdict: over
		// internal/schemata/instrument.go, 78 mutants with 28 SURVIVORS in them,
		// the two paths returned the same mutants and the same verdict for every
		// one. The survivors are the point; an earlier comparison over a scope
		// where everything died proved nothing and was thrown away.
		// docs/experiments/turning-gating-on.md.
		ditto.Gated(),
		ditto.IgnoreSourceFiles("(^release\\.go$|testdata\\/.*)"),
	)
}

//go:build mutation

package ditto_test

import (
	"os/exec"
	"testing"

	"github.com/Disble/ditto"
)

func TestMutation(t *testing.T) {
	ditto.Release(
		t,
		ditto.ForceColors(),
		ditto.WithRepositoryRoot("."),
		ditto.WithTestCommand(makeCommand(t)+" test.failfast MAKEFLAGS="),
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
		// Measured before turning on, twice. Gating removes 394 of this
		// repository's 729 compilations -- 54% -- because 60.1% of its mutants
		// can be expressed as a runtime branch. And over
		// internal/schemata/instrument.go, 78 mutants with 28 SURVIVORS in
		// them, both paths returned the same mutants and the same verdict for
		// every one. docs/experiments/turning-gating-on.md.
		ditto.Gated(),
		ditto.IgnoreSourceFiles("(^release\\.go$|testdata\\/.*)"),
	)
}

// makeCommand is the name GNU make answers to on this machine.
//
// This test hardcoded `make`, which does not exist on Windows -- the same reason
// .githooks/pre-commit probes three names, written down there since the hook was
// unrunnable for it. So the gate could never run here at all: the command failed
// instantly with no output, ditto read a non-zero exit as a red baseline, and
// refused in under three seconds. A test command that cannot start is not a red
// baseline, and finding that out took two full runs that measured nothing.
func makeCommand(t *testing.T) string {
	t.Helper()

	for _, candidate := range []string{"make", "mingw32-make", "gmake"} {
		if _, err := exec.LookPath(candidate); err == nil {
			return candidate
		}
	}

	t.Fatal("no GNU make on PATH; looked for make, mingw32-make, gmake")

	return ""
}

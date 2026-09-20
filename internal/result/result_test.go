package result_test

import (
	"testing"

	"github.com/Disble/ditto/internal/result"
)

// TestOutputAnswersForBothEndings pins the widened contract, and it is a test
// rather than a sentence because the narrower one was load-bearing: the package
// scope a release reads is announced by a GREEN suite, and a green suite is an
// Err. Answering empty there is what made the scope unreadable.
func TestOutputAnswersForBothEndings(t *testing.T) {
	t.Parallel()

	// Ok is a command that failed — a killed mutant — and its output is what a
	// refusal prints.
	if got := result.Output(result.Ok("boom")); got != "boom" {
		t.Fatalf("Output(Ok) = %q, want the command's own words", got)
	}

	// Err is a command that succeeded, and its stream is the only place the
	// package scope of a passing suite exists.
	if got := result.Output(result.Err[string]("{\"Action\":\"skip\"}")); got != "{\"Action\":\"skip\"}" {
		t.Fatalf("Output(Err) = %q, want the stream a passing suite produced", got)
	}
}

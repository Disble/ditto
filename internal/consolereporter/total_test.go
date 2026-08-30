package consolereporter_test

import (
	"testing"

	"github.com/Disble/ditto/internal/consolereporter"
	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/dittotesting/fakelogger"
	"github.com/Disble/ditto/internal/dittotesting/fakescorecalculator"
	"github.com/Disble/ditto/internal/dittotesting/stubdiffer"
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/gatedreporter"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/result"
	"github.com/Disble/ditto/internal/verbosereporter"
	"github.com/stretchr/testify/assert"
)

// A scope with nothing mutable in it and a suite that failed are different
// answers, and the score cannot tell them apart: the calculator reports -1 for
// an empty run, which is below every threshold. The count is what separates
// them.
func TestTotalSeparatesAnEmptyRunFromAFailingOne(t *testing.T) {
	reporter := consolereporter.New(fakelogger.New(), stubdiffer.New(""), fakescorecalculator.Always(0), 0)

	assert.Equal(t, 0, reporter.Total(), "a reporter that has scored nothing counts nothing")

	reporter.AddDiagnostic(ditto.NewDiagnostic(
		future.Resolved(result.Ok("mutant killed")),
		gomutatedfile.New("dummy", "dummy.go", nil, nil),
	))
	reporter.Summarize()

	assert.Equal(t, 1, reporter.Total())
}

// TestTheDecoratorsForwardTheCount is backlog entry 12's class, caught once
// already: a decorator that drops a capability is refused by nothing. Without
// this the count is unreadable exactly when the stack is deepest.
func TestTheDecoratorsForwardTheCount(t *testing.T) {
	logger := fakelogger.New()
	console := consolereporter.New(logger, stubdiffer.New(""), fakescorecalculator.Always(0), 0)

	console.AddDiagnostic(ditto.NewDiagnostic(
		future.Resolved(result.Ok("mutant killed")),
		gomutatedfile.New("dummy", "dummy.go", nil, nil),
	))
	console.Summarize()

	// Both decorators, and the gated one over the verbose one, which is the
	// stack the CI gate actually builds and the one the count was unreadable in.
	for name, wrapped := range map[string]ditto.Reporter{
		"verbose":            verbosereporter.New(logger, console),
		"gated":              gatedreporter.New(logger, noGates{}, console),
		"gated over verbose": gatedreporter.New(logger, noGates{}, verbosereporter.New(logger, console)),
	} {
		t.Run(name, func(t *testing.T) {
			counted, ok := wrapped.(interface{ Total() int })
			if !ok {
				t.Fatalf("the %s reporter does not forward the count", name)
			}

			assert.Equal(t, 1, counted.Total())
		})
	}
}

// noGates is a gated laboratory that gated nothing, which is all this needs: the
// counters are not the subject here, the forwarding is.
type noGates struct{}

func (noGates) Gated() int    { return 0 }
func (noGates) FellBack() int { return 0 }

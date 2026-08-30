package consolereporter_test

import (
	"testing"

	"github.com/Disble/ditto/internal/consolereporter"
	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/dittotesting/fakelogger"
	"github.com/Disble/ditto/internal/dittotesting/fakescorecalculator"
	"github.com/Disble/ditto/internal/dittotesting/stubdiffer"
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/result"
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

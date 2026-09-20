package consolereporter_test

import (
	"strings"
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

// A mutant the command cannot execute is not a smaller denominator to read a
// score against. It is the part of the scope this run could not measure, so it
// leaves the score on both sides, it is named with its package, and the run does
// not pass whatever the ratio over the rest says.
//
// Measured on the report that asked for this: 43 mutants, 23 killed, 20
// survivors, and 17 of those survivors in a package the command never builds —
// printed as a score of 0.53 against a bar of 0.80, over a run whose ceiling was
// 0.605. docs/reports/ditto-mutation-scope.md.
func TestUnmeasuredMutantsLeaveTheScoreAndAreNamed(t *testing.T) {
	logger := fakelogger.New()

	// The score is pinned to one thing only: 1.00, above the bar. A run that
	// measured everything it scored passes; this one does not, and that is the
	// assertion.
	reporter := consolereporter.New(logger, stubdiffer.New(""), fakescorecalculator.Always(1.0), 0.8)

	for range 4 {
		reporter.AddDiagnostic(ditto.NewDiagnostic(
			future.Resolved(result.Ok("mutant killed")),
			gomutatedfile.New("Comparison", "internal/observability/syncdiag/reader.go", nil, nil),
		))
	}

	for range 17 {
		reporter.AddDiagnostic(ditto.NewUnmeasuredDiagnostic(
			gomutatedfile.New("Comparison Invert", "internal/desktop/app_runtime_services.go", nil, nil),
		))
	}

	assert.False(t, reporter.Summarize().IsOk(),
		"a run that could not measure part of its scope passed on the strength of the rest")

	assert.Equal(t, 4, reporter.Total(), "the unmeasured mutants are still counted in the denominator")
	assert.Equal(t, 17, reporter.Unmeasured())

	report := strings.Join(logger.LoggedLines(), "\n")

	assert.Contains(t, report, "17 of the 21 mutants in this scope are never compiled by this test command")
	assert.Contains(t, report, "internal/desktop (17)")
	assert.Contains(t, report, "Name every package the scope mutates in --test-command, or narrow the scope.")

	// A package that WAS measured is not named as unmeasured, and an unmeasured
	// mutant is not rendered as a survivor: its diff would be a diff of a program
	// that never ran.
	assert.NotContains(t, report, "internal/observability/syncdiag")
	assert.NotContains(t, report, "Mutant survived")
	assert.NotContains(t, report, "internal/desktop/app_runtime_services.go")
}

// And it says nothing when there is nothing to say, for the reason the
// non-viable line says nothing: a line printed on every run is a line people
// stop reading.
func TestAMeasuredScopeSaysNothingAboutBeingUnmeasured(t *testing.T) {
	logger := fakelogger.New()
	reporter := consolereporter.New(logger, stubdiffer.New(""), fakescorecalculator.Always(1.0), 0.8)

	reporter.AddDiagnostic(ditto.NewDiagnostic(
		future.Resolved(result.Ok("mutant killed")),
		gomutatedfile.New("Comparison", "calc/calc.go", nil, nil),
	))

	assert.True(t, reporter.Summarize().IsOk())
	assert.Zero(t, reporter.Unmeasured())

	report := strings.Join(logger.LoggedLines(), "\n")

	assert.NotContains(t, report, "Unmeasured")
	assert.NotContains(t, report, "never compiled by this test command")
}

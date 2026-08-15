package gatedreporter_test

import (
	"testing"

	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/dittotesting/fakelogger"
	"github.com/Disble/ditto/internal/dittotesting/fakereporter"
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/gatedreporter"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/result"
	"github.com/stretchr/testify/assert"
)

type fakeGates struct{ gated, fellBack int }

func (g fakeGates) Gated() int    { return g.gated }
func (g fakeGates) FellBack() int { return g.fellBack }

func TestGatedReporter(t *testing.T) {
	t.Parallel()

	t.Run("says how much of the run came from one compilation", func(t *testing.T) {
		t.Parallel()

		logger := fakelogger.New()
		reporter := gatedreporter.New(logger, fakeGates{gated: 5, fellBack: 2}, fakereporter.New())

		reporter.Summarize()

		assert.Equal(t, []string{
			"┃ Gated: 5 of 7 mutants ran from one compilation; 2 kept their own.",
		}, logger.LoggedLines())
	})

	// The whole point of entry 11: a gated run that gated nothing looked exactly
	// like one that gated everything, and only a panic dropped into TestAll by
	// hand told the difference. Zero has to be as loud as any other number.
	t.Run("says so when the gated path engaged for nothing", func(t *testing.T) {
		t.Parallel()

		logger := fakelogger.New()
		reporter := gatedreporter.New(logger, fakeGates{gated: 0, fellBack: 7}, fakereporter.New())

		reporter.Summarize()

		assert.Equal(t, []string{
			"┃ Gated: none of 7 mutants ran from one compilation; 7 kept their own.",
		}, logger.LoggedLines())
	})

	t.Run("says nothing when no mutant reached a laboratory at all", func(t *testing.T) {
		t.Parallel()

		logger := fakelogger.New()
		reporter := gatedreporter.New(logger, fakeGates{gated: 0, fellBack: 0}, fakereporter.New())

		reporter.Summarize()

		assert.Empty(t, logger.LoggedLines())
	})

	t.Run("hands every diagnostic and the verdict through unchanged", func(t *testing.T) {
		t.Parallel()

		delegate := fakereporter.New()
		reporter := gatedreporter.New(fakelogger.New(), fakeGates{gated: 1, fellBack: 0}, delegate)

		reporter.AddDiagnostic(ditto.NewDiagnostic(
			future.Resolved(result.Err[string]("mutant survived")),
			gomutatedfile.New("dummy", "dummy.go", nil, nil),
		))

		assert.Equal(t, result.Err[any](""), reporter.Summarize())
		assert.Equal(t, 1, delegate.GetSummary().Survived)
	})
}

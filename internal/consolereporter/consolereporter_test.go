package consolereporter_test

import (
	"testing"

	"github.com/Disble/ditto/internal/color"
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

func TestConsoleReporterSummary(t *testing.T) {
	t.Run("reports summary when there are no diagnostics", func(t *testing.T) {
		logger := fakelogger.New()
		reporter := consolereporter.New(logger, stubdiffer.New("+dummy diff"), fakescorecalculator.Always(0), 0)
		reporter.Summarize()

		assert.Equal(t, []string{
			"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓",
			"┃ • Total:        0                    ┃",
			"┃ • Killed:       0                    ┃",
			"┃ • Survived:     0                    ┃",
			"┠┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┨",
			"┃ ✓ Score:     0.00 (minimum: 0.00)    ┃",
			"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛",
		}, logger.LoggedLines())
	})

	t.Run("reports summary when there is one positive diagnostic", func(t *testing.T) {
		logger := fakelogger.New()
		reporter := consolereporter.New(logger, stubdiffer.New("+dummy diff"), fakescorecalculator.Always(0), 0)
		reporter.AddDiagnostic(ditto.NewDiagnostic(
			future.Resolved(result.Ok("mutant killed")),
			gomutatedfile.New("dummy", "dummy.go", nil, nil)),
		)
		reporter.Summarize()

		assert.Equal(t, []string{
			"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓",
			"┃ • Total:        1                    ┃",
			"┃ • Killed:       1                    ┃",
			"┃ • Survived:     0                    ┃",
			"┠┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┨",
			"┃ ✓ Score:     0.00 (minimum: 0.00)    ┃",
			"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛",
		}, logger.LoggedLines())
	})

	t.Run("reports summary when there is one negative diagnostic", func(t *testing.T) {
		logger := fakelogger.New()
		reporter := consolereporter.New(logger, stubdiffer.New("+dummy diff"), fakescorecalculator.Always(0), 0)
		reporter.AddDiagnostic(ditto.NewDiagnostic(
			future.Resolved(result.Err[string]("mutant survived")),
			gomutatedfile.New("dummy", "dummy.go", nil, nil)),
		)
		reporter.Summarize()

		assert.Equal(t, []string{
			"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╍┅",
			"┃ 🧬 Mutant survived: dummy.go → dummy",
			"┠┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄",
			"┃ +dummy diff",
			"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╍┅",
			"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓",
			"┃ • Total:        1                    ┃",
			"┃ • Killed:       0                    ┃",
			"┃ • Survived:     1                    ┃",
			"┠┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┨",
			"┃ ✓ Score:     0.00 (minimum: 0.00)    ┃",
			"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛",
		}, logger.LoggedLines())
	})

	t.Run("reports summary when there are multiple mixed diagnostics", func(t *testing.T) {
		logger := fakelogger.New()
		reporter := consolereporter.New(logger, stubdiffer.New("+dummy diff"), fakescorecalculator.Always(0), 0)
		reporter.AddDiagnostic(ditto.NewDiagnostic(
			future.Resolved(result.Err[string]("mutant survived")),
			gomutatedfile.New("dummy", "dummy.go", nil, nil)),
		)
		reporter.AddDiagnostic(ditto.NewDiagnostic(
			future.Resolved(result.Ok("mutant killed")),
			gomutatedfile.New("dummy", "dummy.go", nil, nil)),
		)
		reporter.AddDiagnostic(ditto.NewDiagnostic(
			future.Resolved(result.Ok("mutant killed")),
			gomutatedfile.New("dummy", "dummy.go", nil, nil)),
		)
		reporter.AddDiagnostic(ditto.NewDiagnostic(
			future.Resolved(result.Ok("mutant killed")),
			gomutatedfile.New("dummy", "dummy.go", nil, nil)),
		)
		reporter.AddDiagnostic(ditto.NewDiagnostic(
			future.Resolved(result.Err[string]("mutant survived")),
			gomutatedfile.New("dummy", "dummy.go", nil, nil)),
		)
		reporter.AddDiagnostic(ditto.NewDiagnostic(
			future.Resolved(result.Err[string]("mutant survived")),
			gomutatedfile.New("dummy", "dummy.go", nil, nil)),
		)
		reporter.AddDiagnostic(ditto.NewDiagnostic(
			future.Resolved(result.Ok("mutant killed")),
			gomutatedfile.New("dummy", "dummy.go", nil, nil)),
		)
		reporter.AddDiagnostic(ditto.NewDiagnostic(
			future.Resolved(result.Ok("mutant killed")),
			gomutatedfile.New("dummy", "dummy.go", nil, nil)),
		)
		reporter.Summarize()

		assert.Equal(t, []string{
			"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╍┅",
			"┃ 🧬 Mutant survived: dummy.go → dummy",
			"┠┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄",
			"┃ +dummy diff",
			"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╍┅",
			"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╍┅",
			"┃ 🧬 Mutant survived: dummy.go → dummy",
			"┠┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄",
			"┃ +dummy diff",
			"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╍┅",
			"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╍┅",
			"┃ 🧬 Mutant survived: dummy.go → dummy",
			"┠┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄",
			"┃ +dummy diff",
			"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╍┅",
			"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓",
			"┃ • Total:        8                    ┃",
			"┃ • Killed:       5                    ┃",
			"┃ • Survived:     3                    ┃",
			"┠┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┨",
			"┃ ✓ Score:     0.00 (minimum: 0.00)    ┃",
			"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛",
		}, logger.LoggedLines())
	})

	t.Run("reports the score calculated by the given score calculator", func(t *testing.T) {
		{
			logger := fakelogger.New()
			reporter := consolereporter.New(logger, stubdiffer.New("+dummy diff"), fakescorecalculator.Always(0.32), 0)
			reporter.Summarize()

			assert.Equal(t, []string{
				"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓",
				"┃ • Total:        0                    ┃",
				"┃ • Killed:       0                    ┃",
				"┃ • Survived:     0                    ┃",
				"┠┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┨",
				"┃ ✓ Score:     0.32 (minimum: 0.00)    ┃",
				"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛",
			}, logger.LoggedLines())
		}

		{
			logger := fakelogger.New()
			reporter := consolereporter.New(logger, stubdiffer.New("+dummy diff"), fakescorecalculator.Always(0.99), 0)
			reporter.Summarize()

			assert.Equal(t, []string{
				"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓",
				"┃ • Total:        0                    ┃",
				"┃ • Killed:       0                    ┃",
				"┃ • Survived:     0                    ┃",
				"┠┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┨",
				"┃ ✓ Score:     0.99 (minimum: 0.00)    ┃",
				"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛",
			}, logger.LoggedLines())
		}
	})

	t.Run("prints a colorful summary", func(t *testing.T) {
		reset := color.Force()
		defer reset()

		t.Run("successful", func(t *testing.T) {
			logger := fakelogger.New()
			reporter := consolereporter.New(logger, stubdiffer.New("+dummy diff"), fakescorecalculator.Always(0.99), 0)
			reporter.Summarize()

			assert.Equal(t, []string{
				"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓",
				"┃ • \033[1mTotal\033[22m:        0                    ┃",
				"┃ • \033[1mKilled\033[22m:       0                    ┃",
				"┃ • \033[1mSurvived\033[22m:     0                    ┃",
				"┠┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┨",
				"┃ \033[1;32m✓ Score:     0.99 (minimum: 0.00)\033[22;0m    ┃",
				"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛",
			}, logger.LoggedLines())
		})

		t.Run("failure", func(t *testing.T) {
			logger := fakelogger.New()
			reporter := consolereporter.New(logger, stubdiffer.New("+dummy diff"), fakescorecalculator.Always(0.99), 1.0)
			reporter.Summarize()

			assert.Equal(t, []string{
				"┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓",
				"┃ • \033[1mTotal\033[22m:        0                    ┃",
				"┃ • \033[1mKilled\033[22m:       0                    ┃",
				"┃ • \033[1mSurvived\033[22m:     0                    ┃",
				"┠┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┨",
				"┃ \033[1;31m⨯ Score:     0.99 (minimum: 1.00)\033[22;0m    ┃",
				"┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛",
			}, logger.LoggedLines())
		})
	})
}

func TestConsoleReporter(t *testing.T) {
	t.Run("fails when below the score is below the configured", func(t *testing.T) {
		{
			reporter := consolereporter.New(
				fakelogger.New(),
				stubdiffer.New("+dummy diff"),
				fakescorecalculator.Always(0.99),
				1.0,
			)

			assert.Equal(t, result.Err[any](""), reporter.Summarize())
		}

		{
			reporter := consolereporter.New(
				fakelogger.New(),
				stubdiffer.New("+dummy diff"),
				fakescorecalculator.Always(1.0),
				1.0,
			)

			assert.Equal(t, result.Ok[any](nil), reporter.Summarize())
		}

		{
			reporter := consolereporter.New(
				fakelogger.New(),
				stubdiffer.New("+dummy diff"),
				fakescorecalculator.Always(0.005),
				0.01,
			)

			assert.Equal(t, result.Err[any](""), reporter.Summarize())
		}

		{
			reporter := consolereporter.New(
				fakelogger.New(),
				stubdiffer.New("+dummy diff"),
				fakescorecalculator.Always(0.5),
				0.5,
			)

			assert.Equal(t, result.Ok[any](nil), reporter.Summarize())
		}
	})
}

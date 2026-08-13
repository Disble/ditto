package verbosereporter_test

import (
	"testing"

	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/dittotesting/fakelogger"
	"github.com/Disble/ditto/internal/dittotesting/fakereporter"
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/result"
	"github.com/Disble/ditto/internal/verbosereporter"
	"github.com/stretchr/testify/assert"
)

func TestVerboseReporter(t *testing.T) {
	t.Run("logs when adding a diagnostic", func(t *testing.T) {
		logger := fakelogger.New()

		verbosereporter.New(
			logger,
			fakereporter.New(),
		).AddDiagnostic(ditto.NewDiagnostic(
			future.Resolved(result.Ok("dummy")),
			gomutatedfile.New("dummy", "dummy.go", nil, nil),
		))

		assert.Equal(t, []string{
			"registering diagnostic…",
		}, logger.LoggedLines())
	})

	t.Run("logs when summarizing", func(t *testing.T) {
		logger := fakelogger.New()

		verbosereporter.New(
			logger,
			fakereporter.New(),
		).Summarize()

		assert.Equal(t, []string{
			"summarizing report…",
		}, logger.LoggedLines())
	})
}

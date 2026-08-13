package verbosetemporarydir_test

import (
	"testing"

	"github.com/Disble/ditto/internal/dittotesting/fakelogger"
	"github.com/Disble/ditto/internal/dittotesting/faketempdirectory"
	"github.com/Disble/ditto/internal/verbosetemporarydir"
	"github.com/stretchr/testify/assert"
)

func TestVerboseTemporaryDir(t *testing.T) {
	t.Run("logs when preparing a new temporary directory", func(t *testing.T) {
		logger := fakelogger.New()

		verbosetemporarydir.New(
			logger,
			faketempdirectory.NewFakeTemporaryDirectory("test"),
		).New()

		assert.Equal(t, []string{
			"setting up new temporary directory at 'test-1'",
		}, logger.LoggedLines())
	})
}

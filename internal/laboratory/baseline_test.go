package laboratory_test

import (
	"testing"

	"github.com/Disble/ditto/internal/dittotesting"
	"github.com/Disble/ditto/internal/dittotesting/fakelogger"
	"github.com/Disble/ditto/internal/dittotesting/fakerepository"
	"github.com/Disble/ditto/internal/dittotesting/faketempdirectory"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/laboratory"
	"github.com/Disble/ditto/internal/result"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBaselineIsAnnounced covers backlog entry 24.
//
// The laboratory already runs the suite once on unmutated code, and already
// times nothing. That one run is the per-mutant price of the test command, and
// it was measured and thrown away — so a caller staring at silence had no way
// to know whether the wait ahead was seconds or an afternoon.
//
// The duration is not asserted, because it is a real clock. The sentence is,
// because the number alone does not say that every mutant pays it again.
func TestBaselineIsAnnounced(t *testing.T) {
	source := dittotesting.Source(`
	|package source
	|
	|var number = 0
	|`)

	mutant := gomutatedfile.New("Integer Increment", "source.go", source, dittotesting.Source(`
	|package source
	|
	|var number = 1
	|`))

	newLaboratory := func(logger *fakelogger.FakeLogger) *laboratory.Laboratory {
		return laboratory.New(
			logger,
			&observingRunner{answer: result.Ok("mutant died")},
			faketempdirectory.NewFakeTemporaryDirectory("tmpdir"),
		)
	}

	t.Run("says what the suite costs, and that every mutant pays it again", func(t *testing.T) {
		logger := fakelogger.New()
		repository := fakerepository.New(fakerepository.FS{"source.go": source}, fakerepository.NewTemporary())

		newLaboratory(logger).Test(repository, mutant).Await()

		require.Len(t, logger.LoggedLines(), 1)
		assert.Contains(t, logger.LoggedLines()[0], "baseline")
		assert.Contains(t, logger.LoggedLines()[0], "every mutant runs it again")
	})

	t.Run("says it once per release rather than once per mutant", func(t *testing.T) {
		logger := fakelogger.New()
		repository := fakerepository.New(fakerepository.FS{"source.go": source}, fakerepository.NewTemporary())
		lab := newLaboratory(logger)

		lab.Test(repository, mutant).Await()
		lab.Test(repository, mutant).Await()
		lab.Test(repository, mutant).Await()

		assert.Len(t, logger.LoggedLines(), 1)
	})
}

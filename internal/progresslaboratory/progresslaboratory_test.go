package progresslaboratory_test

import (
	"testing"

	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/dittotesting"
	"github.com/Disble/ditto/internal/dittotesting/fakelaboratory"
	"github.com/Disble/ditto/internal/dittotesting/fakelogger"
	"github.com/Disble/ditto/internal/dittotesting/fakerepository"
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/progresslaboratory"
	"github.com/Disble/ditto/internal/result"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mutant(label string) *gomutatedfile.GoMutatedFile {
	source := dittotesting.Source(`
	|package source
	|
	|var number = 0
	|`)

	return gomutatedfile.New(label, "source.go", source, dittotesting.Source(`
	|package source
	|
	|var number = 1
	|`))
}

// recordingLaboratory notes when it was asked, so the announcement can be shown
// to happen BEFORE the work rather than after it. A line that only appears once
// a mutant has finished cannot say which mutant a stall is inside.
type recordingLaboratory struct {
	logger *fakelogger.FakeLogger
	seen   [][]string
}

func (l *recordingLaboratory) Test(
	_ ditto.Repository,
	_ *gomutatedfile.GoMutatedFile,
) future.Future[result.Result[string]] {
	l.seen = append(l.seen, append([]string{}, l.logger.LoggedLines()...))

	return future.Resolved(result.Ok("mutant died"))
}

func TestAnnouncesBeforeItRuns(t *testing.T) {
	logger := fakelogger.New()
	inner := &recordingLaboratory{logger: logger}
	subject := progresslaboratory.New(logger, inner)

	subject.Test(fakerepository.New(fakerepository.FS{}), mutant("Integer Increment")).Await()

	require.Len(t, inner.seen, 1)
	assert.Len(t, inner.seen[0], 1, "the mutant was announced only after it had run")
	assert.Contains(t, inner.seen[0][0], "Integer Increment")
}

// TestBatchesExactlyWhenItsDelegateDoes is the invariant the goldens depend on.
//
// A decorator that always claimed to batch would make every laboratory beneath
// it look batchable, and `ditto run` would stop saying what ditto.Release says
// because the two stacks would take different branches.
func TestBatchesExactlyWhenItsDelegateDoes(t *testing.T) {
	t.Run("does not batch over a laboratory that cannot", func(t *testing.T) {
		subject := progresslaboratory.New(fakelogger.New(), fakelaboratory.NewAlways(result.Ok("died")))

		_, batches := subject.(ditto.BatchLaboratory)

		assert.False(t, batches)
	})

	t.Run("batches over one that can", func(t *testing.T) {
		subject := progresslaboratory.New(fakelogger.New(), &batchingLaboratory{})

		_, batches := subject.(ditto.BatchLaboratory)

		assert.True(t, batches)
	})

	t.Run("names every mutant of a batch, in order", func(t *testing.T) {
		logger := fakelogger.New()
		subject := progresslaboratory.New(logger, &batchingLaboratory{})

		batched, ok := subject.(ditto.BatchLaboratory)
		require.True(t, ok)

		batched.TestAll(
			fakerepository.New(fakerepository.FS{}),
			[]*gomutatedfile.GoMutatedFile{mutant("Integer Increment"), mutant("Comparison")},
		)

		require.Len(t, logger.LoggedLines(), 2)
		assert.Contains(t, logger.LoggedLines()[0], "Integer Increment")
		assert.Contains(t, logger.LoggedLines()[1], "Comparison")
	})
}

type batchingLaboratory struct{}

func (l *batchingLaboratory) Test(
	_ ditto.Repository,
	_ *gomutatedfile.GoMutatedFile,
) future.Future[result.Result[string]] {
	return future.Resolved(result.Ok("mutant died"))
}

func (l *batchingLaboratory) TestAll(
	_ ditto.Repository,
	files []*gomutatedfile.GoMutatedFile,
) []future.Future[result.Result[string]] {
	results := make([]future.Future[result.Result[string]], 0, len(files))
	for range files {
		results = append(results, future.Resolved(result.Ok("mutant died")))
	}

	return results
}

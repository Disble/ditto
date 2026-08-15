package verboselaboratory_test

import (
	"testing"

	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/dittotesting/fakelaboratory"
	"github.com/Disble/ditto/internal/dittotesting/fakelogger"
	"github.com/Disble/ditto/internal/dittotesting/fakerepository"
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/result"
	"github.com/Disble/ditto/internal/verboselaboratory"
	"github.com/stretchr/testify/assert"
)

func TestVerboseLaboratory(t *testing.T) {
	t.Run("logs when running tests", func(t *testing.T) {
		logger := fakelogger.New()

		dummyRepository := fakerepository.New(fakerepository.FS{})
		dummyMutatedFile := gomutatedfile.New("dummy-infection", "some-path.go", nil, nil)
		verboselaboratory.New(
			logger,
			fakelaboratory.NewAlways(result.Ok("dummy result")),
		).Test(dummyRepository, dummyMutatedFile)

		assert.Equal(t, []string{
			"running laboratory tests for 'some-path.go'",
			"laboratory result for 'some-path.go': Ok[string](dummy result)",
		}, logger.LoggedLines())
	})

	// Measured before this existed, on the golden fixture: gated 4 of 7 mutants
	// without `-v` and none of 7 with it. `Release` reads testing.Verbose and
	// wraps the stack in this decorator, so a decorator that does not forward
	// makes every laboratory beneath it look like one that cannot batch — and
	// `go test -v` is what CI runs. See docs/experiments/forwarding-the-batch.md.
	t.Run("forwards a whole batch to a laboratory that can take one", func(t *testing.T) {
		delegate := &batchingLaboratory{}

		results := verboselaboratory.New(fakelogger.New(), delegate).TestAll(
			fakerepository.New(fakerepository.FS{}),
			[]*gomutatedfile.GoMutatedFile{
				gomutatedfile.New("a", "a.go", nil, nil),
				gomutatedfile.New("b", "b.go", nil, nil),
			},
		)

		assert.Equal(t, 1, delegate.batches, "the batch has to reach the laboratory below, once")
		assert.Equal(t, 0, delegate.singles, "no mutant may be sent one at a time when the batch was taken")
		assert.Len(t, results, 2)
	})

	// A laboratory that cannot batch still has to work, unchanged, which is the
	// constraint every step of the gated path is under.
	t.Run("keeps one mutant at a time when the delegate cannot batch", func(t *testing.T) {
		logger := fakelogger.New()

		results := verboselaboratory.New(logger, fakelaboratory.NewAlways(result.Ok("dummy result"))).TestAll(
			fakerepository.New(fakerepository.FS{}),
			[]*gomutatedfile.GoMutatedFile{
				gomutatedfile.New("a", "a.go", nil, nil),
				gomutatedfile.New("b", "b.go", nil, nil),
			},
		)

		assert.Len(t, results, 2)
		assert.Equal(t, []string{
			"running laboratory tests for 'a.go'",
			"laboratory result for 'a.go': Ok[string](dummy result)",
			"running laboratory tests for 'b.go'",
			"laboratory result for 'b.go': Ok[string](dummy result)",
		}, logger.LoggedLines())
	})
}

// batchingLaboratory counts how it was asked, which is the whole question: a
// batch forwarded once, or one mutant at a time.
type batchingLaboratory struct {
	batches, singles int
}

func (l *batchingLaboratory) Test(
	_ ditto.Repository,
	_ *gomutatedfile.GoMutatedFile,
) future.Future[result.Result[string]] {
	l.singles++

	return future.Resolved(result.Ok("dummy result"))
}

func (l *batchingLaboratory) TestAll(
	_ ditto.Repository,
	files []*gomutatedfile.GoMutatedFile,
) []future.Future[result.Result[string]] {
	l.batches++

	results := make([]future.Future[result.Result[string]], 0, len(files))
	for range files {
		results = append(results, future.Resolved(result.Ok("dummy result")))
	}

	return results
}

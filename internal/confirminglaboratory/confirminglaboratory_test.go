package confirminglaboratory_test

import (
	"testing"

	"github.com/Disble/ditto/internal/confirminglaboratory"
	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/dittotesting"
	"github.com/Disble/ditto/internal/dittotesting/fakelogger"
	"github.com/Disble/ditto/internal/dittotesting/fakerepository"
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/result"
	"github.com/Disble/ditto/internal/verdict"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertionKill is what `go test -json` streams when a test ran and failed. It
// is the only reason a re-run can tell anything about: a flake can manufacture
// a kill, and only this kind.
const assertionKill = `{"Action":"run","Test":"TestSomething"}
{"Action":"fail","Test":"TestSomething"}`

// buildFailure is a mutant that never became a program. It already leaves the
// score on both sides, so re-running it would buy nothing and cost a suite.
const buildFailure = `{"Action":"build-fail","Package":"example"}`

// scriptedLaboratory answers a different result each call, so the second run of
// the same mutant can disagree with the first — which is exactly what a flake
// does.
type scriptedLaboratory struct {
	answers []result.Result[string]
	calls   int
}

func (l *scriptedLaboratory) Test(
	_ ditto.Repository,
	_ *gomutatedfile.GoMutatedFile,
) future.Future[result.Result[string]] {
	answer := l.answers[min(l.calls, len(l.answers)-1)]
	l.calls++

	return future.Resolved(answer)
}

func mutant() *gomutatedfile.GoMutatedFile {
	source := dittotesting.Source(`
	|package source
	|
	|var number = 0
	|`)

	return gomutatedfile.New("Integer Increment", "source.go", source, dittotesting.Source(`
	|package source
	|
	|var number = 1
	|`))
}

// TestConfirmation covers backlog entry 27.
//
// verifyBaseline is a sync.Once: it catches a suite that is red before anything
// is scored, and cannot catch one that goes red at mutant 37. With no retry
// anywhere, a spurious failure during a mutant run classified as an assertion
// and became a false kill indistinguishable from a real one.
func TestConfirmation(t *testing.T) {
	repository := fakerepository.New(fakerepository.FS{})

	t.Run("a mutant that dies twice is dead", func(t *testing.T) {
		inner := &scriptedLaboratory{answers: []result.Result[string]{
			result.Ok(assertionKill),
			result.Ok(assertionKill),
		}}

		got := confirminglaboratory.New(fakelogger.New(), inner).Test(repository, mutant()).Await()

		assert.True(t, got.IsOk())
		assert.Equal(t, 2, inner.calls)
	})

	t.Run("a mutant that dies once and lives on the re-run is a survivor", func(t *testing.T) {
		// A flake can only manufacture a kill: nothing about a spurious failure
		// makes a mutant the tests DID catch look like it escaped. So a mutant
		// that survives either run survived.
		inner := &scriptedLaboratory{answers: []result.Result[string]{
			result.Ok(assertionKill),
			result.Err[string]("the suite passed"),
		}}

		got := confirminglaboratory.New(fakelogger.New(), inner).Test(repository, mutant()).Await()

		assert.False(t, got.IsOk())
	})

	t.Run("says so, because a verdict that changed on a re-run is worth reading", func(t *testing.T) {
		logger := fakelogger.New()
		inner := &scriptedLaboratory{answers: []result.Result[string]{
			result.Ok(assertionKill),
			result.Err[string]("the suite passed"),
		}}

		confirminglaboratory.New(logger, inner).Test(repository, mutant()).Await()

		require.Len(t, logger.LoggedLines(), 1)
		assert.Contains(t, logger.LoggedLines()[0], "Integer Increment")
	})

	t.Run("a survivor is never re-run, because a flake cannot manufacture one", func(t *testing.T) {
		inner := &scriptedLaboratory{answers: []result.Result[string]{result.Err[string]("the suite passed")}}

		confirminglaboratory.New(fakelogger.New(), inner).Test(repository, mutant()).Await()

		assert.Equal(t, 1, inner.calls)
	})

	t.Run("only an assertion kill is re-run", func(t *testing.T) {
		for name, output := range map[string]string{
			"build failure": buildFailure,
			"deadline":      "ditto: the test command passed its 10m0s deadline and was stopped; " + verdict.DeadlineMarker,
			"unknown":       "a test command that emits no json stream",
		} {
			t.Run(name, func(t *testing.T) {
				inner := &scriptedLaboratory{answers: []result.Result[string]{result.Ok(output)}}

				confirminglaboratory.New(fakelogger.New(), inner).Test(repository, mutant()).Await()

				assert.Equal(t, 1, inner.calls)
			})
		}
	})
}

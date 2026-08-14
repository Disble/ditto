package testingtlaboratory

import (
	"testing"

	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/result"
)

type TestingT interface {
	Helper()
	Run(name string, testFn func(*testing.T)) bool
}

type TestingTLaboratory struct {
	t        TestingT
	delegate ditto.Laboratory
	parallel bool
	// goParallel is a seam, not a configuration point. Observing that a
	// subtest was marked parallel otherwise means reading testing.T's
	// unexported state, which is not observable from outside: T.Parallel
	// returns early when its parent has no barrier, and supplying one makes it
	// block forever on a nil signal channel. Both are private details that
	// have changed between Go releases and will change again.
	goParallel func(*testing.T)
}

func New(t TestingT, delegate ditto.Laboratory, parallel bool) *TestingTLaboratory {
	t.Helper()

	return &TestingTLaboratory{
		t:          t,
		delegate:   delegate,
		parallel:   parallel,
		goParallel: (*testing.T).Parallel,
	}
}

func (l *TestingTLaboratory) Test(
	repository ditto.Repository,
	file *gomutatedfile.GoMutatedFile,
) future.Future[result.Result[string]] {
	l.t.Helper()

	fut := future.Deferred[result.Result[string]]()

	l.t.Run(file.Label(), func(t *testing.T) { //nolint:thelper
		if l.parallel {
			l.goParallel(t)
		}

		fut.Resolve(l.delegate.Test(repository, file).Await())
	})

	return fut
}

// TestAll forwards a whole file's mutants when the laboratory below can take
// them, and keeps a subtest per mutant either way.
//
// Without this the batch stops here: Release asks the outermost laboratory, and
// a decorator that does not forward makes every laboratory beneath it look like
// one that cannot batch. That failure is silent — the run still works, one
// compilation per mutant, exactly as if nothing had been wired at all.
func (l *TestingTLaboratory) TestAll(
	repository ditto.Repository,
	files []*gomutatedfile.GoMutatedFile,
) []future.Future[result.Result[string]] {
	l.t.Helper()

	batched, ok := l.delegate.(ditto.BatchLaboratory)
	if !ok {
		results := make([]future.Future[result.Result[string]], 0, len(files))
		for _, file := range files {
			results = append(results, l.Test(repository, file))
		}

		return results
	}

	// The batch runs first, because that is what one compilation for several
	// mutants means. The subtests then report what it found, in order.
	inner := batched.TestAll(repository, files)
	results := make([]future.Future[result.Result[string]], len(files))

	for i, file := range files {
		fut := future.Deferred[result.Result[string]]()
		results[i] = fut

		l.t.Run(file.Label(), func(t *testing.T) { //nolint:thelper
			if l.parallel {
				l.goParallel(t)
			}

			fut.Resolve(inner[i].Await())
		})
	}

	return results
}

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

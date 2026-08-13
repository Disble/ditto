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
}

func New(t TestingT, delegate ditto.Laboratory, parallel bool) *TestingTLaboratory {
	t.Helper()

	return &TestingTLaboratory{
		t:        t,
		delegate: delegate,
		parallel: parallel,
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
			t.Parallel()
		}

		fut.Resolve(l.delegate.Test(repository, file).Await())
	})

	return fut
}

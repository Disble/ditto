package verboselaboratory

import (
	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/result"
)

type VerboseLaboratory struct {
	logger   ditto.Logger
	delegate ditto.Laboratory
}

func New(logger ditto.Logger, delegate ditto.Laboratory) *VerboseLaboratory {
	return &VerboseLaboratory{
		logger:   logger,
		delegate: delegate,
	}
}

func (l *VerboseLaboratory) Test(
	repository ditto.Repository,
	file *gomutatedfile.GoMutatedFile,
) future.Future[result.Result[string]] {
	l.logger.Logf("running laboratory tests for '%s'", file)
	fut := l.delegate.Test(repository, file)
	l.logger.Logf("laboratory result for '%s': %+v", file, fut.Await())

	return fut
}

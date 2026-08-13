package laboratory

import (
	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/result"
)

type TestRunner interface {
	Test(repository ditto.TemporaryRepository) result.Result[string]
}

type TemporaryDirectory interface {
	New() string
}

type Laboratory struct {
	testRunner         TestRunner
	temporaryDirectory TemporaryDirectory
}

func New(testRunner TestRunner, temporaryDirectory TemporaryDirectory) *Laboratory {
	return &Laboratory{
		testRunner:         testRunner,
		temporaryDirectory: temporaryDirectory,
	}
}

func (l *Laboratory) Test(
	repository ditto.Repository,
	file *gomutatedfile.GoMutatedFile,
) future.Future[result.Result[string]] {
	tempRepository := repository.LinkAllToTemporaryRepository(l.temporaryDirectory.New())
	defer tempRepository.Remove()

	file.WriteTo(tempRepository)

	return future.Resolved(l.testRunner.Test(tempRepository))
}

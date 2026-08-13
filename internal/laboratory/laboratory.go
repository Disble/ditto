package laboratory

import (
	"sync"

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

// Laboratory runs each mutant in a sandbox, and keeps the sandboxes.
//
// Building one is not cheap: it walks the whole repository and creates a
// symlink per file, measured at roughly 0.45ms per file. Nothing about a
// sandbox depends on which mutant will run in it, so rebuilding it for every
// mutant paid that walk once per mutant rather than once per run.
//
// Sandboxes are pooled rather than shared, so a caller running mutants
// concurrently draws as many as its peak concurrency and a sequential one —
// which is what ditto is built for — only ever builds one. The number alive at
// any instant is therefore the same as when each mutant built its own, which
// matters because that number is also what an interrupted run leaves behind.
type Laboratory struct {
	testRunner         TestRunner
	temporaryDirectory TemporaryDirectory

	mutex sync.Mutex
	idle  []ditto.TemporaryRepository
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
	sandbox := l.acquire(repository)
	defer l.hand(sandbox, file)

	file.WriteTo(sandbox)

	return future.Resolved(l.testRunner.Test(sandbox))
}

func (l *Laboratory) acquire(repository ditto.Repository) ditto.TemporaryRepository {
	l.mutex.Lock()

	if last := len(l.idle) - 1; last >= 0 {
		sandbox := l.idle[last]
		l.idle = l.idle[:last]
		l.mutex.Unlock()

		return sandbox
	}

	l.mutex.Unlock()

	return repository.LinkAllToTemporaryRepository(l.temporaryDirectory.New())
}

// hand returns a sandbox to the pool with the mutation undone.
//
// Restoring first is the whole safety of reuse: a sandbox released while still
// carrying its mutation would make the next mutant run against two mutations
// at once, and the result would read as an ordinary survivor.
func (l *Laboratory) hand(sandbox ditto.TemporaryRepository, file *gomutatedfile.GoMutatedFile) {
	file.RestoreIn(sandbox)

	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.idle = append(l.idle, sandbox)
}

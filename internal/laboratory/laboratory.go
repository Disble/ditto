package laboratory

import (
	"sync"

	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/result"
	"github.com/Disble/ditto/internal/verdict"
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

	baseline sync.Once
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

	l.verifyBaseline(sandbox)

	file.WriteTo(sandbox)

	return future.Resolved(l.testRunner.Test(sandbox))
}

// verifyBaseline runs the suite once, on unmutated code, before any mutant is
// scored.
//
// Ditto recognises a killed mutant by the test command exiting non-zero, and a
// command that fails before it compiles anything exits non-zero too. Every
// mutant is then scored killed and the report says 1.00 without naming a cause —
// the highest number the tool can print, for a run that tested nothing.
//
// Measured on ditto's own gate: **431 of 431 mutants killed in 5.46 seconds**,
// twelve milliseconds each, because `make` needed a git directory that a
// sandbox does not have. docs/experiments/false-perfect-score.md.
//
// The gated path has refused this since it learned to read its own unselected
// run, which it pays for anyway. This path has no such run, so it buys one: once
// per release, not once per mutant, and `perf/baseline.json` ratchets that
// number in both directions. One run is what the answer costs.
//
// The sandbox arrives clean — the mutation is written after this returns — so
// what runs here is the repository's own suite.
func (l *Laboratory) verifyBaseline(sandbox ditto.TemporaryRepository) {
	l.baseline.Do(func() {
		// Ok means the command failed. For a mutant that is a kill; with nothing
		// mutated it is a suite that was already red.
		res := l.testRunner.Test(sandbox)
		if res.IsOk() {
			// The command's own output travels with the refusal. Without it the
			// reader is told a suite is red and left to guess which of a hundred
			// reasons it is, in a sandbox they cannot see.
			panic(ditto.NewRefusalError("ditto: the test command fails on unmutated code, so every mutant would be scored " +
				"killed; refusing to score against a red baseline\n\n" + verdict.Text(result.Output(res))))
		}
	})
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

package laboratory

import (
	"sync"
	"time"

	"github.com/Disble/ditto/internal/color"
	"github.com/Disble/ditto/internal/commandscope"
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
//
// It is also the only place that can learn what the configured test command
// executes, without paying for it: the baseline run it already makes once per
// release prints the `go test -json` stream that names every package the command
// executes, and the sandbox it ran in is the tree to resolve the rest against.
// See Executes and internal/commandscope.
type Laboratory struct {
	logger             ditto.Logger
	testRunner         TestRunner
	temporaryDirectory TemporaryDirectory

	mutex sync.Mutex
	idle  []ditto.TemporaryRepository

	baseline sync.Once
	scope    *commandscope.Scope
}

// scopeRunner is the process seam under the one `go list` a scope costs.
//
// A package-level variable with a Set for tests rather than a constructor
// parameter, because it is not something a caller configures: it is how ditto
// asks the toolchain which packages the command compiles, and the only thing
// that ever varies is a test's answer to it.
var scopeRunner commandscope.Runner = commandscope.OSRunner{} //nolint:gochecknoglobals // one seam, with the restore below

// SetScopeRunnerForTest replaces the process the scope resolution would start,
// and returns the restore.
func SetScopeRunnerForTest(runner commandscope.Runner) func() {
	previous := scopeRunner
	scopeRunner = runner

	return func() { scopeRunner = previous }
}

func New(logger ditto.Logger, testRunner TestRunner, temporaryDirectory TemporaryDirectory) *Laboratory {
	return &Laboratory{
		logger:             logger,
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

// Executes reports whether the configured test command can execute the package
// that owns a repository-relative source path.
//
// This is the one question the report cannot answer for itself. Mutants come
// from files the scope selected and are judged by one command, and when that
// command does not compile a mutant's package the mutant is not badly tested, it
// is unmeasured: nothing the command runs can kill it, so it survives, and a
// score counting it mixes 'your tests missed this' with 'your command cannot see
// this'. Measured on the report that asked for this: 43 mutants, 20 survivors,
// 17 of them in one package the command never builds,
// docs/reports/ditto-mutation-scope.md.
//
// It answers true when the question cannot be answered — a command that is not
// `go test -json` has no readable package scope, and so does one whose toolchain
// cannot be asked — because a wrong accusation fails a run that was correct,
// while a missing one only leaves the caller with what it had before.
//
// The first call is what pays for the baseline on a release that never asks
// otherwise. There is no second cost: the same sandbox and the same once.
//
// There is no nil check on the scope, and that is deliberate rather than an
// omission. The baseline always produces one — commandscope.New answers for an
// empty stream by declining to refuse anything — so a guard here would be
// unreachable, and a guard no test can distinguish from its absence is dead
// weight pretending to be defense. Measured: deleting it changed no test.
func (l *Laboratory) Executes(repository ditto.Repository, relativePath string) bool {
	sandbox := l.acquire(repository)
	defer l.returnToPool(sandbox)

	l.verifyBaseline(sandbox)

	return l.scope.Executes(relativePath)
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
		started := time.Now()
		res := l.testRunner.Test(sandbox)
		elapsed := time.Since(started)

		if res.IsOk() {
			// The command's own output travels with the refusal. Without it the
			// reader is told a suite is red and left to guess which of a hundred
			// reasons it is, in a sandbox they cannot see.
			panic(ditto.NewRefusalError("ditto: the test command fails on unmutated code, so every mutant would be scored " +
				"killed; refusing to score against a red baseline\n\n" + verdict.Text(result.Output(res))))
		}

		// What a GREEN suite said about itself, which is the part nothing else
		// reads. `go test -json` names every package the command executes, and
		// this is the one run that produces that stream for free: the baseline
		// was already paid for, once per release. A command that emits no stream
		// leaves the scope unknown, and unknown refuses nothing.
		l.scope = commandscope.New(result.Output(res), sandbox.Root(), scopeRunner)

		// This run was already paid for and the clock above already measured it;
		// throwing the number away was the waste. It is the per-mutant price of
		// the test command, and printed beside the mutant count the release
		// announces it is the whole bill — which is what nobody had when a run
		// advancing normally was reported as a hang.
		l.logger.Logf(
			"%s baseline: the suite took %s on unmutated code, and every mutant runs it again.",
			color.Yellow("┃"),
			elapsed.Round(time.Millisecond),
		)
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

	l.returnToPool(sandbox)
}

// returnToPool hands a clean sandbox back for the next mutant. It is hand
// without the restore, for a caller that never wrote anything into it.
func (l *Laboratory) returnToPool(sandbox ditto.TemporaryRepository) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.idle = append(l.idle, sandbox)
}

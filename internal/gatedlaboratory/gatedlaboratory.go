// Package gatedlaboratory runs a file's mutants from one compilation.
//
// It is the two halves put together. internal/schemata turns a file and its
// mutants into one instrumented file that selects a mutant at run time, and
// internal/gobuildrunner compiles a package's tests once and runs that binary.
// Each is useless alone: compiling once only helps when the compiled thing stops
// changing between mutants, and instrumenting only helps if something compiles
// it once.
//
// What it removes is measured: the fixed cost of starting `go test`, 750-950 ms
// per mutant whatever the suite does. See docs/experiments/test-invocation.md.
package gatedlaboratory

import (
	"os"
	"path"
	"strings"

	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/gobuildrunner"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/result"
	"github.com/Disble/ditto/internal/schemata"
)

type TemporaryDirectory interface {
	New() string
}

// Runner is the seam over building and running, so the laboratory can be tested
// without a toolchain.
type Runner interface {
	Select(mutant int)
	Built() bool
	Test(repository ditto.TemporaryRepository) result.Result[string]
}

// scopedRunner is a runner that can be told which package's mutation it is
// answering.
//
// It is optional, the way the temporary directory's RemoveAll and the
// laboratory's Total are: a runner that cannot be scoped keeps running every
// package, which is what this did before the closure was measured. A decorator
// or a runner that dropped it would cost time and never a verdict.
type scopedRunner interface {
	ScopeTo(directory string)
}

// directorySharer is a runner that puts its compiled test binaries somewhere
// chosen for it.
type directorySharer interface {
	SetCompilationDirectory(directory string)
}

// GatedLaboratory instruments a file once, compiles once, and selects a mutant
// per run. Anything it cannot gate goes to the laboratory it delegates to, which
// is the path ditto has always taken.
type GatedLaboratory struct {
	delegate           ditto.Laboratory
	temporaryDirectory TemporaryDirectory
	newRunner          func(packagePath string) Runner

	gated    int
	fellBack int

	// sandbox and compilationDirectory belong to the release rather than to a
	// batch, and they are two halves of one thing. A fresh sandbox per batch
	// makes every package out of date — Go's build IDs cover a package's
	// directory — so a shared compilation directory pays nothing without it.
	// Measured the other way round first: the directory alone was worth zero.
	sandbox              ditto.TemporaryRepository
	compilationDirectory string
	directoriesCreated   int
}

// New retains the package-scope runner for its existing callers. Production
// Gated assembly uses NewModuleScope because its contract is the complete
// default Go module scope, not the mutated file's package.
func New(delegate ditto.Laboratory, temporaryDirectory TemporaryDirectory) *GatedLaboratory {
	return &GatedLaboratory{
		delegate:           delegate,
		temporaryDirectory: temporaryDirectory,
		newRunner: func(packagePath string) Runner {
			return gobuildrunner.New(packagePath)
		},
	}
}

// NewModuleScope gates through the measured default Go module scope.
func NewModuleScope(delegate ditto.Laboratory, temporaryDirectory TemporaryDirectory) *GatedLaboratory {
	return &GatedLaboratory{
		delegate:           delegate,
		temporaryDirectory: temporaryDirectory,
		newRunner: func(string) Runner {
			return gobuildrunner.NewModuleScope()
		},
	}
}

// NewDisabled keeps the batched-laboratory shape and counters while routing all
// mutants to the ordinary laboratory. It is used for custom commands whose
// complete execution plan module scope cannot faithfully represent.
func NewDisabled(delegate ditto.Laboratory, temporaryDirectory TemporaryDirectory) *GatedLaboratory {
	return &GatedLaboratory{
		delegate:           delegate,
		temporaryDirectory: temporaryDirectory,
	}
}

// Gated and FellBack are exact counters: how many mutants ran from the shared
// compilation, and how many kept their own. They are what this is judged on,
// because wall clock on a working machine varies by more than half and these do
// not vary at all.
func (l *GatedLaboratory) Gated() int    { return l.gated }
func (l *GatedLaboratory) FellBack() int { return l.fellBack }

// CompilationDirectories counts the compilation directories this release made.
// It is 0 for a release that never gated and 1 for one that did, whatever its
// batch count, which is the integer that says the toolchain's up-to-date check
// is allowed to work across batches.
func (l *GatedLaboratory) CompilationDirectories() int { return l.directoriesCreated }

// Test keeps the one-mutant-at-a-time contract, and takes the old path. A single
// mutant cannot repay a compilation.
func (l *GatedLaboratory) Test(
	repository ditto.Repository,
	file *gomutatedfile.GoMutatedFile,
) future.Future[result.Result[string]] {
	l.fellBack++

	return l.delegate.Test(repository, file)
}

// TestAll runs every mutant of one file, gating the ones it can.
func (l *GatedLaboratory) TestAll(
	repository ditto.Repository,
	files []*gomutatedfile.GoMutatedFile,
) []future.Future[result.Result[string]] {
	if len(files) == 0 {
		return nil
	}

	if l.newRunner == nil {
		return l.all(repository, files)
	}

	planned := schemata.Plan(files[0].Source(), mutated(files))
	if gatedCount(planned.Selector) == 0 {
		return l.all(repository, files)
	}

	sandbox := l.sandboxFor(repository)

	sandbox.Overwrite(files[0].Path(), planned.Instrumented)

	runner := l.newRunner(packageOf(files[0].Path()))

	// Told which package the mutation is in, so a runner that can work out which
	// test binaries may observe it starts only those. A package test binary
	// compiles its own package plus the transitive closure of what it imports,
	// so one without this package in that closure holds no code that can refer
	// to anything the mutation changed. docs/experiments/dependency-closure.md.
	if scoped, ok := runner.(scopedRunner); ok {
		scoped.ScopeTo(packageOf(files[0].Path()))
	}

	// Given the release's one compilation directory, taken here on the first
	// batch. A fresh directory per batch is a full module-wide rebuild every
	// time, because it throws away the toolchain's up-to-date check.
	if sharer, ok := runner.(directorySharer); ok {
		l.shareCompilationDirectory(sharer, sandbox.Root())
	}

	// The first run is what compiles. A package that does not build has to be
	// survivable rather than fatal: under one shared compilation a single bad
	// site would otherwise take every other mutant in the run with it, so the
	// whole file goes back to its own path.
	first := runner.Test(sandbox)
	if !runner.Built() {
		files[0].RestoreIn(sandbox)

		return l.all(repository, files)
	}

	// That run selected no mutant, so every gate took its original arm: it is
	// the file's own suite, and this path is the only one that ever measures it.
	//
	// Ok here means the command failed. For a selected run that is a killed
	// mutant; with nothing selected it means the SCHEMA broke this file, because
	// unselected arms are the original — so every mutant of it would be scored
	// killed and the report would say 1.00 without naming a cause. Measured at
	// 4 of 4 against 1 of 4 on the same mutants, docs/experiments/changed-scope.md.
	//
	// The whole file goes back to its own path, exactly as one that does not
	// build does. It used to refuse the entire run instead, and that was too
	// wide: turning gating on for this repository died on cmd/ditto/main.go in
	// 6.44 seconds while every other file was fine, and a schema that cannot
	// carry one file is a reason to stop schematising that file, not to stop
	// measuring the repository. Dextool blacklists such a schema and continues.
	//
	// Falling back cannot hide a genuinely red repository: the ordinary path
	// this returns to runs verifyBaseline once per release and refuses there.
	// One guard covers that, and the wider one only removed files it could have
	// measured.
	//
	// The run is already paid for. Reading it is what keeps it from being a lie.
	if first.IsOk() {
		files[0].RestoreIn(sandbox)

		return l.all(repository, files)
	}

	return l.selectEach(repository, files, planned.Selector, runner, sandbox)
}

func (l *GatedLaboratory) selectEach(
	repository ditto.Repository,
	files []*gomutatedfile.GoMutatedFile,
	selector []int,
	runner Runner,
	sandbox ditto.TemporaryRepository,
) []future.Future[result.Result[string]] {
	results := make([]future.Future[result.Result[string]], len(files))

	for i, file := range files {
		if selector[i] == 0 {
			l.fellBack++
			results[i] = l.delegate.Test(repository, file)

			continue
		}

		l.gated++

		runner.Select(selector[i])
		results[i] = future.Resolved(runner.Test(sandbox))
	}

	files[0].RestoreIn(sandbox)

	return results
}

// sandboxFor is the release's one sandbox, taken on the first batch that needs
// one and reused after that.
//
// Reuse is safe because every batch restores the file it overwrote before it
// returns, so the tree is pristine between batches and one batch cannot see
// another's mutation. What it buys is a stable path, and the path is the whole
// point: each batch used to link its own sandbox, the absolute package
// directories differed, Go's build IDs cover those directories, and nothing was
// ever up to date — measured as a shared compilation directory that paid
// nothing at all (docs/experiments/the-compile-is-per-file.md).
//
// It is created through the temporary directory rather than beside it, so the
// release's existing cleanup removes it with every other sandbox.
func (l *GatedLaboratory) sandboxFor(repository ditto.Repository) ditto.TemporaryRepository {
	if l.sandbox == nil {
		l.sandbox = repository.LinkAllToTemporaryRepository(l.temporaryDirectory.New())
	}

	return l.sandbox
}

// shareCompilationDirectory gives the runner the release's one compilation
// directory, taking it on the first batch that needs one.
//
// It is created inside the release's sandbox, so the existing cleanup removes it
// with the sandbox that holds it. A failure here is not fatal: the runner makes
// its own directory, which is what it did before, and the only cost is the
// rebuild this exists to avoid.
func (l *GatedLaboratory) shareCompilationDirectory(sharer directorySharer, sandboxRoot string) {
	if l.compilationDirectory == "" {
		directory, err := os.MkdirTemp(sandboxRoot, "ditto-module-compile-")
		if err != nil {
			return
		}

		l.compilationDirectory = directory
		l.directoriesCreated++
	}

	sharer.SetCompilationDirectory(l.compilationDirectory)
}

func (l *GatedLaboratory) all(
	repository ditto.Repository,
	files []*gomutatedfile.GoMutatedFile,
) []future.Future[result.Result[string]] {
	results := make([]future.Future[result.Result[string]], 0, len(files))

	for _, file := range files {
		l.fellBack++
		results = append(results, l.delegate.Test(repository, file))
	}

	return results
}

func mutated(files []*gomutatedfile.GoMutatedFile) [][]byte {
	all := make([][]byte, 0, len(files))
	for _, file := range files {
		all = append(all, file.Mutated())
	}

	return all
}

func gatedCount(selector []int) int {
	count := 0

	for _, id := range selector {
		if id != 0 {
			count++
		}
	}

	return count
}

// packageOf is the directory the file lives in, which is the package `go test -c`
// compiles. Paths are relative with forward slashes — that is a file's identity
// here, and the reason a pattern written with `/` matches on every platform.
func packageOf(relativePath string) string {
	directory := path.Dir(strings.ReplaceAll(relativePath, "\\", "/"))
	if directory == "." {
		return "."
	}

	return "./" + directory
}

// NewWithRunner is the seam the tests use, so the laboratory's decisions can be
// checked without a Go toolchain compiling anything.
func NewWithRunner(delegate ditto.Laboratory, temporaryDirectory TemporaryDirectory, runner Runner) *GatedLaboratory {
	return &GatedLaboratory{
		delegate:           delegate,
		temporaryDirectory: temporaryDirectory,
		newRunner:          func(string) Runner { return runner },
	}
}

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

// GatedLaboratory instruments a file once, compiles once, and selects a mutant
// per run. Anything it cannot gate goes to the laboratory it delegates to, which
// is the path ditto has always taken.
type GatedLaboratory struct {
	delegate           ditto.Laboratory
	temporaryDirectory TemporaryDirectory
	newRunner          func(packagePath string) Runner

	gated    int
	fellBack int
}

func New(delegate ditto.Laboratory, temporaryDirectory TemporaryDirectory) *GatedLaboratory {
	return &GatedLaboratory{
		delegate:           delegate,
		temporaryDirectory: temporaryDirectory,
		newRunner: func(packagePath string) Runner {
			return gobuildrunner.New(packagePath)
		},
	}
}

// Gated and FellBack are exact counters: how many mutants ran from the shared
// compilation, and how many kept their own. They are what this is judged on,
// because wall clock on a working machine varies by more than half and these do
// not vary at all.
func (l *GatedLaboratory) Gated() int    { return l.gated }
func (l *GatedLaboratory) FellBack() int { return l.fellBack }

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

	planned := schemata.Plan(files[0].Source(), mutated(files))
	if gatedCount(planned.Selector) == 0 {
		return l.all(repository, files)
	}

	sandbox := repository.LinkAllToTemporaryRepository(l.temporaryDirectory.New())
	sandbox.Overwrite(files[0].Path(), planned.Instrumented)

	runner := l.newRunner(packageOf(files[0].Path()))

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
	// mutant; with nothing selected it is a red baseline, and under a red
	// baseline every selected run fails too — so every mutant of the file is
	// scored killed and the report says 1.00 without naming a cause. Measured at
	// 4 of 4 against 1 of 4 on the same mutants, docs/experiments/changed-scope.md.
	//
	// The run is already paid for. Reading it is what keeps it from being a lie.
	if first.IsOk() {
		files[0].RestoreIn(sandbox)

		panic("ditto: " + files[0].Path() + " fails its own suite with no mutant selected, so every " +
			"mutant of it would be scored killed; refusing to score against a red baseline")
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

package ditto

import (
	"flag"
	"os"
	"testing"

	"github.com/Disble/ditto/internal/cmdtestrunner"
	"github.com/Disble/ditto/internal/color"
	"github.com/Disble/ditto/internal/consolereporter"
	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/fsrepository"
	"github.com/Disble/ditto/internal/fstemporarydir"
	"github.com/Disble/ditto/internal/gatedlaboratory"
	"github.com/Disble/ditto/internal/gatedreporter"
	"github.com/Disble/ditto/internal/gosourcefile"
	"github.com/Disble/ditto/internal/gotextdiff"
	"github.com/Disble/ditto/internal/ignoredrepository"
	"github.com/Disble/ditto/internal/iologger"
	"github.com/Disble/ditto/internal/laboratory"
	"github.com/Disble/ditto/internal/prettydiff"
	"github.com/Disble/ditto/internal/scopedrepository"
	"github.com/Disble/ditto/internal/scorecalculator"
	"github.com/Disble/ditto/internal/testingtlaboratory"
	"github.com/Disble/ditto/internal/verboselaboratory"
	"github.com/Disble/ditto/internal/verbosereporter"
	"github.com/Disble/ditto/internal/verboserepository"
	"github.com/Disble/ditto/internal/verbosetemporarydir"
	"github.com/Disble/ditto/internal/verbosetestrunner"
	"github.com/Disble/ditto/viruses"
	"github.com/Disble/ditto/viruses/arithmetic"
	"github.com/Disble/ditto/viruses/arithmeticassignment"
	"github.com/Disble/ditto/viruses/arithmeticassignmentinvert"
	"github.com/Disble/ditto/viruses/bitwise"
	"github.com/Disble/ditto/viruses/comparison"
	"github.com/Disble/ditto/viruses/comparisoninvert"
	"github.com/Disble/ditto/viruses/comparisonreplace"
	"github.com/Disble/ditto/viruses/floatdecrement"
	"github.com/Disble/ditto/viruses/floatincrement"
	"github.com/Disble/ditto/viruses/integerdecrement"
	"github.com/Disble/ditto/viruses/integerincrement"
	"github.com/Disble/ditto/viruses/loopbreak"
	"github.com/Disble/ditto/viruses/loopcondition"
	"github.com/Disble/ditto/viruses/rangebreak"
)

var dittoVerbose *bool //nolint:gochecknoglobals

func init() { //nolint:gochecknoinits // a flag must be registered before flag.Parse runs
	dittoVerbose = flag.Bool("ditto.v", false, "verbose: print additional output")
}

var defaultOptions = Options{ //nolint:gochecknoglobals
	Repository:                fsrepository.New("."),
	TestRunner:                cmdtestrunner.New("go", "test", "-count=1", "./..."),
	TemporaryDir:              fstemporarydir.New("ditto-"),
	MinimumThreshold:          1.0,
	Parallel:                  false,
	IgnoreSourceFilesPatterns: nil,
	Viruses: []viruses.Virus{
		arithmetic.New(),
		arithmeticassignment.New(),
		arithmeticassignmentinvert.New(),
		bitwise.New(),
		comparison.New(),
		comparisoninvert.New(),
		comparisonreplace.New(),
		floatdecrement.New(),
		floatincrement.New(),
		integerdecrement.New(),
		integerincrement.New(),
		loopbreak.New(),
		loopcondition.New(),
		rangebreak.New(),
	},
}

// Release releases the ditto! It infects all source files with viruses that
// mutate the source code DNA and perform tests to determine whether the mutants
// survive.
//
// This is the entry point to configure and run mutation tests. You may want to
// configure it with some options. Here is the available options and their
// defaults:
//
//   - WithRepositoryRoot: `.`
//   - WithTestCommand: `go test -count=1 ./...`
//   - WithMinimumThreshold: `1.0`
//   - Parallel: `false`
//   - IgnoreSourceFiles: `nil`
//   - WithViruses: all available (see viruses.Virus' implementations)
//
// The results are then presented in the console. If the mutation score is equal
// to or above the configured threshold (WithMinimumThreshold), the execution is
// considered successful. Failed otherwise. Regardless of the execution result,
// any surviving mutant (no tests failed after applying the source code
// mutation) will also be presented in the console for analysis.
func Release(t *testing.T, options ...Option) {
	t.Helper()

	opts := defaultOptions
	for _, option := range options {
		opts = option(opts)
	}

	var logger ditto.Logger = iologger.New(os.Stdout)

	var reporter ditto.Reporter = consolereporter.New(
		logger,
		prettydiff.New(gotextdiff.New()),
		scorecalculator.New(),
		opts.MinimumThreshold,
	)

	reclaimSandboxes(t, opts.TemporaryDir, logger)

	if opts.IgnoreSourceFilesPatterns != nil {
		opts.Repository = ignoredrepository.New(opts.IgnoreSourceFilesPatterns, opts.Repository)
	}

	if opts.ChangedRanges != nil {
		opts.Repository = scopedrepository.New(scopedRanges(opts.ChangedRanges), opts.Repository)
	}

	if verbose() {
		opts.Repository = verboserepository.New(logger, opts.Repository)
		opts.TemporaryDir = verbosetemporarydir.New(logger, opts.TemporaryDir)
		opts.TestRunner = verbosetestrunner.New(logger, opts.TestRunner)
		reporter = verbosereporter.New(logger, reporter)
	}

	lab, gates := assemble(opts, logger)

	// Wrapped here, after the verbose decorator and before the cleanup that
	// summarises, so the counts are the last thing a run says.
	if gates != nil {
		reporter = gatedreporter.New(logger, gates, reporter)
	}

	t.Cleanup(func() {
		t.Helper()

		res := reporter.Summarize()
		if !res.IsOk() {
			t.Fail()
		}
	})

	lab = testingtlaboratory.New(t, lab, opts.Parallel)

	logger.Logf("%s %s", color.Yellow("┃"), color.Green("Releasing Ditto…"))
	ditto.New(opts.Repository, lab, reporter).Release(
		opts.Viruses...,
	)
}

// reclaimSandboxes registers the cleanup that removes what a run left behind.
//
// Sandboxes outlive each mutant now, so removing them belongs to the run rather
// than to the loop that used to rebuild one per mutant. t.Cleanup covers a normal
// finish, a failed run and a t.Fatal. It cannot cover the process being killed —
// that is what the owning process id in the directory name is for, so a later run
// can tell an abandoned parent from one still in use. Registered before the
// decorators wrap the temporary directory, because they do not forward this.
func reclaimSandboxes(t *testing.T, temporaryDir laboratory.TemporaryDirectory, logger ditto.Logger) {
	t.Helper()

	sandboxes, ok := temporaryDir.(interface{ RemoveAll() error })
	if !ok {
		return
	}

	t.Cleanup(func() {
		err := sandboxes.RemoveAll()
		if err != nil {
			logger.Logf("%s %s", color.Yellow("┃"), err)
		}
	})
}

// assemble stacks the laboratories, innermost first. Gated goes below verbose so
// that what is logged is what actually ran, and both stay below the one that
// makes a subtest per mutant.
//
// It returns the gated laboratory as well as the stack, because its two counters
// are the only thing that can say whether the gated path engaged, and a decorator
// above it cannot be asked. The pointer is nil, concretely rather than as an
// interface holding a nil, when the run is not gated.
func assemble(opts Options, logger ditto.Logger) (ditto.Laboratory, *gatedlaboratory.GatedLaboratory) {
	var lab ditto.Laboratory = laboratory.New(opts.TestRunner, opts.TemporaryDir)

	var gates *gatedlaboratory.GatedLaboratory

	if opts.Gated {
		gates = gatedlaboratory.New(lab, opts.TemporaryDir)
		lab = gates
	}

	if verbose() {
		lab = verboselaboratory.New(logger, lab)
	}

	return lab, gates
}

func verbose() bool {
	return *dittoVerbose || testing.Verbose()
}

// scopedRanges converts the public range type into the one the source files
// carry. The two are kept apart so an internal package never becomes part of
// this module's published surface.
func scopedRanges(ranges map[string][]Range) map[string][]gosourcefile.Range {
	converted := make(map[string][]gosourcefile.Range, len(ranges))

	for path, spans := range ranges {
		fileRanges := make([]gosourcefile.Range, 0, len(spans))
		for _, span := range spans {
			fileRanges = append(fileRanges, gosourcefile.Range{Start: span.Start, End: span.End})
		}

		converted[path] = fileRanges
	}

	return converted
}

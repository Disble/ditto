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
	"github.com/Disble/ditto/internal/gotextdiff"
	"github.com/Disble/ditto/internal/ignoredrepository"
	"github.com/Disble/ditto/internal/iologger"
	"github.com/Disble/ditto/internal/laboratory"
	"github.com/Disble/ditto/internal/prettydiff"
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

func init() { //nolint:gochecknoinits
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

	// Sandboxes outlive each mutant now, so removing them belongs to the run
	// rather than to the loop that used to rebuild one per mutant. t.Cleanup
	// covers a normal finish, a failed run and a t.Fatal. It cannot cover the
	// process being killed — that is what the owning process id in the
	// directory name is for, so a later run can tell an abandoned parent from
	// one still in use. Registered before the decorators wrap it, because they
	// do not forward this.
	if sandboxes, ok := opts.TemporaryDir.(interface{ RemoveAll() error }); ok {
		t.Cleanup(func() {
			if err := sandboxes.RemoveAll(); err != nil {
				logger.Logf("%s %s", color.Yellow("┃"), err)
			}
		})
	}

	if opts.IgnoreSourceFilesPatterns != nil {
		opts.Repository = ignoredrepository.New(opts.IgnoreSourceFilesPatterns, opts.Repository)
	}

	if verbose() {
		opts.Repository = verboserepository.New(logger, opts.Repository)
		opts.TemporaryDir = verbosetemporarydir.New(logger, opts.TemporaryDir)
		opts.TestRunner = verbosetestrunner.New(logger, opts.TestRunner)
		reporter = verbosereporter.New(logger, reporter)
	}

	var lab ditto.Laboratory = laboratory.New(opts.TestRunner, opts.TemporaryDir)
	if verbose() {
		lab = verboselaboratory.New(logger, lab)
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

func verbose() bool {
	return *dittoVerbose || testing.Verbose()
}

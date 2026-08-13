package ditto

import (
	"regexp"
	"strings"

	"github.com/Disble/ditto/internal/cmdtestrunner"
	"github.com/Disble/ditto/internal/color"
	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/fsrepository"
	"github.com/Disble/ditto/internal/laboratory"
	"github.com/Disble/ditto/viruses"
)

type Option func(Options) Options

// Range is a half-open byte range within one file: Start is included, End is
// not. Offsets are counted from the first byte of that file.
type Range struct {
	Start int
	End   int
}

type Options struct {
	Repository                ditto.Repository
	TestRunner                laboratory.TestRunner
	TemporaryDir              laboratory.TemporaryDirectory
	MinimumThreshold          float32
	Parallel                  bool
	IgnoreSourceFilesPatterns []*regexp.Regexp
	Viruses                   []viruses.Virus
	ChangedRanges             map[string][]Range
}

// WithChangedRanges restricts the release to the given byte ranges of the given
// files, keyed by repository-relative path with forward slashes.
//
// This is what makes ditto cheap enough to run while you are still writing the
// code. Every mutant costs a full run of the test command, so mutating a line
// the change never touched buys nothing and is charged at the same rate as one
// that matters.
//
// A file with no entry is not mutated at all. A file with an empty range list
// is mutated whole.
//
// The ranges are kept per file on purpose, and callers should keep them that
// way too. A byte offset only means something against the file it was measured
// in, because each file is parsed on its own and every file's positions start
// from the same base. Ranges from several files merged into one set make every
// file answer to all of them: mutants appear in code no diff touched, and the
// number of them grows as the square of the number of files rather than in
// proportion to it.
func WithChangedRanges(ranges map[string][]Range) func(Options) Options {
	return func(options Options) Options {
		options.ChangedRanges = ranges

		return options
	}
}

// WithRepositoryRoot configures which directory is the repository root. This is
// usually required when your mutation test file lives some other place that is
// not root itself.
func WithRepositoryRoot(repositoryRoot string) func(Options) Options {
	return func(options Options) Options {
		options.Repository = fsrepository.New(repositoryRoot)

		return options
	}
}

// WithTestCommand configures the test command to run, as string. You may
// configure it as you wish, as a `makefile` phony target, for example. Or
// simply run the standard `go test` command with extra flags, such as `timeout`
// and `tags`.
func WithTestCommand(testCommand string) func(Options) Options {
	return func(options Options) Options {
		testCommandParts := strings.Split(testCommand, " ")
		options.TestRunner = cmdtestrunner.New(testCommandParts[0], testCommandParts[1:]...)

		return options
	}
}

// WithMinimumThreshold represents the minimum mutation test score to consider
// the execution successful. A float between `0.0` and `1.0`.
func WithMinimumThreshold(minimumThreshold float32) func(Options) Options {
	return func(options Options) Options {
		options.MinimumThreshold = minimumThreshold

		return options
	}
}

// Parallel indicates whether to run the tests on the mutants in parallel. Given
// Ditto is executed via Go's testing framework, the level of parallelism can be
// configured when running the mutation tests. For example, with
// WithTestCommand(`go test -v -tags=mutation -parallel 3`).
func Parallel() func(Options) Options {
	return func(options Options) Options {
		options.Parallel = true

		return options
	}
}

// IgnoreSourceFiles configures regular expressions representing source files
// to be filtered out and not suffer any mutations.
func IgnoreSourceFiles(patterns ...string) func(Options) Options {
	return func(options Options) Options {
		for _, pattern := range patterns {
			options.IgnoreSourceFilesPatterns = append(options.IgnoreSourceFilesPatterns, regexp.MustCompile(pattern))
		}

		return options
	}
}

// WithViruses configure the list of viruses to infect the source files with.
// You can also implement your own viruses (generic or even
// application-specific).
func WithViruses(virus viruses.Virus, rest ...viruses.Virus) func(Options) Options {
	return func(options Options) Options {
		options.Viruses = append([]viruses.Virus{virus}, rest...)

		return options
	}
}

// ForceColors forces the use of colors in the output. This is useful when
// running the mutation tests in a CI environment, for example.
func ForceColors() func(Options) Options {
	return func(options Options) Options {
		color.Force()

		return options
	}
}

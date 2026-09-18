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

// commandScope is the test-command shape whose complete execution plan Gated
// may replace. Unknown commands stay on the ordinary laboratory path.
type commandScope uint8

const (
	unsupportedScope commandScope = iota
	moduleScope
)

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
	Gated                     bool
	ConfirmKills              bool
	Verbose                   bool
	SandboxStrategy           string
	commandScope              commandScope
	// RepositoryRoot is kept beside Repository so a later option can rebuild it.
	RepositoryRoot string
}

// Verbose prints what a run is doing as it does it.
//
// Release reads `go test`'s own verbosity as well, because inside a test binary
// that is where a reader has already said what they want. Run cannot: outside a
// test binary `testing.Verbose` panics rather than answering, so a caller that
// wants the same output asks for it here.
func Verbose() func(Options) Options {
	return func(options Options) Options {
		options.Verbose = true

		return options
	}
}

// Gated runs eligible mutants from one complete-module compilation instead of
// one test-command start each.
//
// Ditto normally starts the test command once per mutant, and that start costs
// 750-950 ms whatever the suite does — the dominant cost of a run. Gating only
// optimizes the default Go module scope, `go test -count=1 ./...`, including
// the built-in -json form. Any custom or unsupported WithTestCommand keeps the
// ordinary laboratory path exactly as configured.
func Gated() func(Options) Options {
	return func(options Options) Options {
		options.Gated = true

		return options
	}
}

// ConfirmKills re-runs a mutant that died by assertion, once, and believes the
// second answer when it disagrees.
//
// It exists because verifyBaseline is a sync.Once: it refuses a suite that is
// already red before anything is scored, and cannot see one that goes red at
// mutant 37. With no retry anywhere, a spurious failure during a mutant run is
// a kill no test earned and nothing in the report tells it from a real one.
//
// Off by default because it doubles the cost of every assertion kill, and a
// repository whose suite does not flake buys nothing with it. Only assertion
// kills are re-run: a mutant that never built already leaves the score on both
// sides, and a deadline is a clock ditto fired itself.
func ConfirmKills() func(Options) Options {
	return func(options Options) Options {
		options.ConfirmKills = true

		return options
	}
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
		options.RepositoryRoot = repositoryRoot
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
		options.commandScope = scopeOf(testCommand)

		testCommandParts := strings.Split(testCommand, " ")
		options.TestRunner = cmdtestrunner.New(testCommandParts[0], testCommandParts[1:]...)

		return options
	}
}

// scopeOf recognizes only the command token sequences whose complete package
// scope has been measured. It is a closed set rather than a parser on purpose:
// anything it does not recognize keeps its ordinary execution, and a command
// that is almost the default is not the default. Extra spacing, an extra flag,
// a package-local scope, an alternate executable and a make target all fall
// outside it by construction rather than by remembering to check for them.
var moduleScopeCommands = map[string]bool{ //nolint:gochecknoglobals // one fixed set, read only
	"go test -count=1 ./...":       true,
	"go test -count=1 -json ./...": true,
	"go test -json -count=1 ./...": true,
}

func scopeOf(command string) commandScope {
	if moduleScopeCommands[command] {
		return moduleScope
	}

	return unsupportedScope
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

// WithSandboxStrategy chooses how each file reaches a sandbox: "link", "copy"
// or "hardlink".
//
// It exists to be measured rather than argued about. A symlink is a reference to
// a file and not a copy of one, and Go refuses to embed an irregular file, so a
// package with an embed directive cannot build in a linked sandbox.
// See docs/experiments/the-sandbox-is-a-reference.md.
//
// None of the three touches a symlink the repository already has: that is
// reproduced as the same link, carrying its raw target, so the sandbox holds the
// tree that is on disk rather than one flattened into regular files.
// See docs/experiments/a-symlink-in-the-tree.md.
func WithSandboxStrategy(strategy string) func(Options) Options {
	return func(options Options) Options {
		options.SandboxStrategy = strategy

		return options
	}
}

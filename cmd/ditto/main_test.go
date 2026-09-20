package main

import (
	"bytes"
	"flag"
	"fmt"
	"strings"
	"testing"

	"github.com/Disble/ditto"
	"github.com/Disble/ditto/internal/dittotesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVersion covers backlog entry 25: the only route to the answer was
// `go version -m $(command -v ditto)`, and the answer matters because a
// non-compiling mutant is scored differently by v0.6.0 and v0.7.0.
func TestVersion(t *testing.T) {
	t.Run("names the module version the binary was built from", func(t *testing.T) {
		assert.Equal(t, "ditto v0.7.0", versionLine("v0.7.0"))
	})

	t.Run("says so rather than inventing one when there is no released version", func(t *testing.T) {
		// `go build` inside a checkout records `(devel)`, and a test binary
		// records nothing at all. Neither is a release, and reporting either as
		// though it were is the failure this entry exists to prevent.
		assert.Equal(t, "ditto (built from source; no released version recorded)", versionLine("(devel)"))
		assert.Equal(t, "ditto (built from source; no released version recorded)", versionLine(""))
	})

	// -v as well as --version, because that is what people type. It is safe at
	// this level: subcommand flags are parsed by their own FlagSet, and Go's flag
	// package does no prefix matching, so `ditto run -v` was never a thing.
	for _, argument := range []string{"--version", "-version", "-v", "version"} {
		t.Run(argument+" reports the version and does not fail", func(t *testing.T) {
			out := &bytes.Buffer{}

			err := command([]string{argument}, out)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "ditto ")
			assert.NotContains(t, out.String(), "has no subcommand")
		})
	}
}

// TestStagedGated covers backlog entry 26: `-gated` was registered by
// `ditto run` and not by `ditto staged`, so every mutant of a staged run paid
// the fixed cost of starting the test command with no way to decline it.
func TestStagedGated(t *testing.T) {
	t.Run("staged can gate", func(t *testing.T) {
		options := ditto.Options{}
		for _, option := range stagedOptions("go test -count=1 -json ./...", 0.8, true, false, false, "") {
			options = option(options)
		}

		assert.True(t, options.Gated)
	})

	t.Run("and does not gate unless asked", func(t *testing.T) {
		options := ditto.Options{}
		for _, option := range stagedOptions("go test -count=1 -json ./...", 0.8, false, false, false, "") {
			options = option(options)
		}

		assert.False(t, options.Gated)
	})
}

// TestUnmeasuredScopeHasItsOwnExitCode covers the half of the report that is
// about a wrapper rather than about a reader: a scope ditto cannot measure and a
// score below the bar were both 1, so an automated gate could not tell them apart
// — and the responses differ, because no test answers an unmeasurable scope.
// docs/reports/ditto-mutation-scope.md.
func TestUnmeasuredScopeHasItsOwnExitCode(t *testing.T) {
	assert.Equal(t, exitUnmeasured, exitCodeOf(ditto.UnmeasuredScopeError{Unmeasured: 17, Scored: 26}))

	// Wrapped, because the error travels back through RunStaged and RunChanged
	// before main sees it and any of them may add context on the way.
	assert.Equal(t, exitUnmeasured, exitCodeOf(fmt.Errorf("running the change: %w",
		ditto.UnmeasuredScopeError{Unmeasured: 1, Scored: 2})))
}

// And every other failure stays one code: a refusal, a usage error and a score
// below the bar are told apart by their message, which a reader has.
func TestEveryOtherFailureKeepsThePlainExitCode(t *testing.T) {
	assert.Equal(t, exitFailure, exitCodeOf(ditto.ScoreBelowThresholdError{Minimum: 0.8}))
	assert.Equal(t, exitFailure, exitCodeOf(ditto.NoMutantsError{}))
	assert.Equal(t, exitFailure, exitCodeOf(errNoSubcommand))
}

// The legend is what makes the number usable, and a number nobody can look up is
// a number people guess at.
func TestUsageNamesTheExitCodes(t *testing.T) {
	out := &bytes.Buffer{}

	usage(out)

	assert.Contains(t, out.String(), "Exit codes")
	assert.Contains(t, out.String(), "unmeasured")
}

// TestTestCommandHelp covers backlog entry 23. The old description was accurate
// and told the reader nothing about what the default costs, which is the one
// thing they need at the moment of typing.
func TestTestCommandHelp(t *testing.T) {
	t.Run("says the command runs once per mutant", func(t *testing.T) {
		// Case-insensitive on purpose: the flag help has no bold, so the toll is
		// emphasised by capitals, and the assertion is about the fact being
		// there rather than about how it is shouted.
		assert.Contains(t, strings.ToLower(testCommandHelp), "once per mutant")
	})

	t.Run("names the lever rather than only the toll", func(t *testing.T) {
		// It named "the package that owns the change", which is right only while
		// the scope holds one package. A command that names one of five leaves
		// the other four unmeasurable, and the release now fails on that instead
		// of scoring it, so the lever is every package the scope mutates.
		assert.Contains(t, testCommandHelp, "name every package the scope mutates")
	})

	t.Run("says what a command that leaves part of the scope out costs", func(t *testing.T) {
		assert.Contains(t, testCommandHelp, "reported as unmeasured")
		assert.Contains(t, testCommandHelp, "exit 3")
	})

	t.Run("says what dropping -json costs, not only what it does", func(t *testing.T) {
		assert.Contains(t, testCommandHelp, "counted as killed")
	})
}

// TestTestCommandHelpRendersItsValueName is here because the defect it covers is
// invisible to every other test: flag.PrintDefaults takes the FIRST backquoted
// word in a usage string as the flag's value name, so a stray pair of backticks
// silently renames the flag's argument. The line shipped for two releases
// rendering as `-test-command -json`, which reads like a second flag, and no
// assertion on the constant could ever have seen it.
func TestTestCommandHelpRendersItsValueName(t *testing.T) {
	rendered := &bytes.Buffer{}

	flags := flag.NewFlagSet("ditto staged", flag.ContinueOnError)
	flags.SetOutput(rendered)
	flags.String("test-command", "go test -count=1 -json ./...", testCommandHelp)
	flags.PrintDefaults()

	assert.Contains(t, rendered.String(), "-test-command command")
	assert.NotContains(t, rendered.String(), "-test-command -json")
	assert.NotContains(t, rendered.String(), "-test-command ./...")
}

// TestGatedHelpCopy pins what --gated's -h line promises, because the behavior
// behind the flag changed in 0.11.0: Gated() now answers the complete module
// scope and only the exact default commands, so the previous wording -- "run a
// file's mutants from one compilation" -- described a package-local build that
// no longer exists, on all three subcommands at once.
func TestGatedHelpCopy(t *testing.T) {
	t.Run("says the scope is the whole module", func(t *testing.T) {
		assert.Contains(t, gatedHelp, "module")
	})

	t.Run("names the exact commands that are eligible", func(t *testing.T) {
		assert.Contains(t, gatedHelp, "go test -count=1 ./...")
		assert.Contains(t, gatedHelp, "-json")
	})

	t.Run("says a custom test command keeps its own path", func(t *testing.T) {
		assert.Contains(t, gatedHelp, "custom")
	})

	// The old line promised one compilation for "a file's" mutants, which was
	// the package-scope behavior. Nothing in the help may claim it again.
	t.Run("does not name the mutated file as the compilation unit", func(t *testing.T) {
		assert.NotContains(t, gatedHelp, "a file's mutants")
	})
}

// TestGatedHelpRendersWithoutAValueName is the same guard
// TestTestCommandHelpRendersItsValueName holds: the first backquoted word in a
// usage string becomes the flag's VALUE NAME, and --gated is a bool flag that
// must never render as though it takes one. A backtick anywhere in gatedHelp
// would silently turn `-gated` into `-gated something`.
func TestGatedHelpRendersWithoutAValueName(t *testing.T) {
	rendered := &bytes.Buffer{}

	flags := flag.NewFlagSet("ditto run", flag.ContinueOnError)
	flags.SetOutput(rendered)
	flags.Bool("gated", false, gatedHelp)
	flags.PrintDefaults()

	assert.Contains(t, rendered.String(), "-gated")
	assert.NotContains(t, rendered.String(), "-gated ", "a value name rendered after the bool flag")
}

// TestIncludePrefixReachesThePlan covers the flag the report asked for: the
// mirror of --exclude-prefix, so the honest pass it wants — one run per owning
// package — is one flag instead of one exclusion for each package the change
// happens to touch.
//
// Asserted through the two subcommands rather than through ditto.Prefixes,
// because the wiring between them is the part a unit test of the plan cannot
// see, and it is the part that would silently do nothing.
func TestIncludePrefixReachesThePlan(t *testing.T) {
	t.Run("changed", func(t *testing.T) {
		dir := dittotesting.GitRepository(t)

		dittotesting.WriteFile(t, dir, "web/web.go", "package web\n\nfunc Web(a, b int) bool { return a > b }\n")
		dittotesting.WriteFile(t, dir, "tools/tool.go", "package tools\n\nfunc Tool(a, b int) bool { return a > b }\n")
		dittotesting.Git(t, dir, "add", "-A")
		dittotesting.Git(t, dir, "commit", "-m", "two packages")

		out := &bytes.Buffer{}
		err := changedCommand([]string{"--since", "base", "--dry", "--cwd", dir, "--include-prefix", "web/"}, out)

		require.NoError(t, err)
		assert.Contains(t, out.String(), "web/web.go")
		assert.NotContains(t, out.String(), "tools/tool.go")
	})

	t.Run("staged", func(t *testing.T) {
		dir := dittotesting.GitRepository(t)

		dittotesting.WriteFile(t, dir, "web/web.go", "package web\n\nfunc Web(a, b int) bool { return a > b }\n")
		dittotesting.WriteFile(t, dir, "tools/tool.go", "package tools\n\nfunc Tool(a, b int) bool { return a > b }\n")
		dittotesting.Git(t, dir, "add", "-A")

		out := &bytes.Buffer{}
		err := stagedCommand([]string{"--cwd", dir, "--dry", "--include-prefix", "web/"}, out)

		require.NoError(t, err)
		assert.Contains(t, out.String(), "web/web.go")
		assert.NotContains(t, out.String(), "tools/tool.go")
	})
}

// TestChangedRefusesToGuessABase covers the one decision `changed` deliberately
// does not make for you. There is no default that is right on a CI checkout, in
// a working tree and on a branch at once, and a base guessed wrong is either a
// bill nobody asked for or a scope of nothing reported as success.
func TestChangedRefusesToGuessABase(t *testing.T) {
	err := changedCommand([]string{"--dry"}, &bytes.Buffer{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--since")
}

// TestChangedIsASubcommand keeps the dispatch honest: an unknown subcommand
// prints usage and fails, so a `changed` that was never wired would look exactly
// like a typo.
func TestChangedIsASubcommand(t *testing.T) {
	out := &bytes.Buffer{}

	err := command([]string{"--help"}, out)

	require.NoError(t, err)
	assert.Contains(t, out.String(), "ditto changed")
}

// TestChangedThresholdBounds pins the one arithmetic check in this command. A
// mutation score is between 0 and 1, and a threshold outside that is a request
// that can never be met or can never fail -- either way the gate stops meaning
// anything, silently.
func TestChangedThresholdBounds(t *testing.T) {
	for _, threshold := range []string{"-0.1", "1.1", "2"} {
		t.Run("refuses "+threshold, func(t *testing.T) {
			err := changedCommand([]string{"--since", "HEAD", "--threshold", threshold}, &bytes.Buffer{})

			require.Error(t, err)
			assert.Contains(t, err.Error(), "between 0 and 1")
		})
	}

	for _, threshold := range []string{"0", "0.5", "1"} {
		t.Run("accepts "+threshold, func(t *testing.T) {
			// It gets past the bounds check and fails on the repository instead,
			// which is what says the number itself was allowed through.
			err := changedCommand(
				[]string{"--since", "HEAD", "--threshold", threshold, "--dry", "--cwd", t.TempDir()},
				&bytes.Buffer{},
			)

			require.Error(t, err)
			assert.NotContains(t, err.Error(), "between 0 and 1")
		})
	}
}

// TestStagedThresholdBounds is the same check on the subcommand that already had
// it, so the two cannot drift apart unnoticed.
func TestStagedThresholdBounds(t *testing.T) {
	err := stagedCommand([]string{"--threshold", "1.5"}, &bytes.Buffer{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "between 0 and 1")
}

// TestChangedDispatchPassesItsFlagsOn covers the argument slice itself. Passing
// one too few drops the subcommand's first flag; one too many hands `changed`
// its own name as a positional and flag.Parse stops there. Both leave --since
// unset, and both look exactly like a user who forgot it.
func TestChangedDispatchPassesItsFlagsOn(t *testing.T) {
	err := command([]string{"changed", "--since", "HEAD", "--dry", "--cwd", t.TempDir()}, &bytes.Buffer{})

	require.Error(t, err)
	assert.NotContains(t, err.Error(), "--since", "the flags did not reach the subcommand")
}

// TestChangedDryReportsTheScope covers what --dry is for: answering "what would
// this cost" without paying for it. The per-file lines and the widened-scope
// notice are the whole output, and neither had ever been exercised.
func TestChangedDryReportsTheScope(t *testing.T) {
	dir := dittotesting.GitRepositoryWithAChange(t)
	out := &bytes.Buffer{}

	err := changedCommand([]string{"--since", "base", "--dry", "--cwd", dir}, out)

	require.NoError(t, err)
	assert.Contains(t, out.String(), "1 file(s) changed since base")
	assert.Contains(t, out.String(), "added.go:")
	// A derived scope says nothing about itself; only a widened one explains.
	assert.NotContains(t, out.String(), "scope:")
}

func TestChangedDrySaysWhenThereIsNothingToDo(t *testing.T) {
	dir := dittotesting.GitRepositoryWithAChange(t)
	out := &bytes.Buffer{}

	// Its own base: a range from a commit to itself is empty by construction.
	err := changedCommand([]string{"--since", "HEAD", "--dry", "--cwd", dir}, out)

	require.NoError(t, err)
	assert.Contains(t, out.String(), "nothing changed since HEAD is worth mutating")
}

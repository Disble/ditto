package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Disble/ditto"
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

	for _, argument := range []string{"--version", "-version", "version"} {
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
		for _, option := range stagedOptions("go test -count=1 -json ./...", 0.8, true, false, "") {
			options = option(options)
		}

		assert.True(t, options.Gated)
	})

	t.Run("and does not gate unless asked", func(t *testing.T) {
		options := ditto.Options{}
		for _, option := range stagedOptions("go test -count=1 -json ./...", 0.8, false, false, "") {
			options = option(options)
		}

		assert.False(t, options.Gated)
	})
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
		assert.Contains(t, testCommandHelp, "name the package")
	})

	t.Run("says what dropping -json costs, not only what it does", func(t *testing.T) {
		assert.Contains(t, testCommandHelp, "counted as killed")
	})
}

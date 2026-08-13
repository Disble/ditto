package ditto_test

import (
	"regexp"
	"testing"

	"github.com/Disble/ditto"
	"github.com/Disble/ditto/internal/cmdtestrunner"
	"github.com/Disble/ditto/internal/fsrepository"
	"github.com/Disble/ditto/viruses"
	"github.com/Disble/ditto/viruses/integerincrement"
	"github.com/Disble/ditto/viruses/loopbreak"
	"github.com/stretchr/testify/assert"
)

//nolint:exhaustruct
func TestOptions(t *testing.T) {
	t.Run("can configure repository root", func(t *testing.T) {
		options := ditto.WithRepositoryRoot(".")(ditto.Options{})
		assert.Equal(t, fsrepository.New("."), options.Repository)
	})

	t.Run("can configure test command to run", func(t *testing.T) {
		{
			options := ditto.WithTestCommand("yes")(ditto.Options{})
			assert.Equal(t, cmdtestrunner.New("yes", []string{}...), options.TestRunner)
		}
		{
			options := ditto.WithTestCommand("echo some value")(ditto.Options{})
			assert.Equal(t, cmdtestrunner.New("echo", "some", "value"), options.TestRunner)
		}
	})

	t.Run("can configure minimum threshold", func(t *testing.T) {
		{
			options := ditto.WithMinimumThreshold(0.25)(ditto.Options{})
			assert.Equal(t, float32(0.25), options.MinimumThreshold)
		}

		{
			options := ditto.WithMinimumThreshold(0.75)(ditto.Options{})
			assert.Equal(t, float32(0.75), options.MinimumThreshold)
		}
	})

	t.Run("can configure parallel", func(t *testing.T) {
		options := ditto.Parallel()(ditto.Options{})
		assert.Equal(t, true, options.Parallel)
	})

	t.Run("can configure source files to ignore", func(t *testing.T) {
		{
			options := ditto.IgnoreSourceFiles(".*")(ditto.Options{})
			assert.Equal(t, []*regexp.Regexp{regexp.MustCompile(".*")}, options.IgnoreSourceFilesPatterns)
		}

		{
			options := ditto.IgnoreSourceFiles(`.*\.go`)(ditto.Options{})
			assert.Equal(t, []*regexp.Regexp{regexp.MustCompile(`.*\.go`)}, options.IgnoreSourceFilesPatterns)
		}
	})

	t.Run("can configure viruses to infect source files", func(t *testing.T) {
		{
			options := ditto.WithViruses(loopbreak.New())(ditto.Options{})
			assert.Equal(t, []viruses.Virus{loopbreak.New()}, options.Viruses)
		}

		{
			options := ditto.WithViruses(loopbreak.New(), integerincrement.New())(ditto.Options{})
			assert.Equal(t, []viruses.Virus{loopbreak.New(), integerincrement.New()}, options.Viruses)
		}
	})
}

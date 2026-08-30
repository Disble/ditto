package ditto_test

import (
	"testing"

	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/dittotesting"
	"github.com/Disble/ditto/internal/dittotesting/fakelaboratory"
	"github.com/Disble/ditto/internal/dittotesting/fakelogger"
	"github.com/Disble/ditto/internal/dittotesting/fakereporter"
	"github.com/Disble/ditto/internal/dittotesting/fakerepository"
	"github.com/Disble/ditto/internal/result"
	"github.com/Disble/ditto/viruses/integerincrement"
	"github.com/stretchr/testify/assert"
)

// TestProgress covers the half of backlog entry 22 that belongs to the release
// rather than to a laboratory: which file is being mutated, and how many mutants
// it has.
//
// The count is the multiplier for the baseline duration the laboratory prints.
// Neither number was on screen when a ten-minute wait was reported as a hang,
// which is why nobody multiplied them.
func TestProgress(t *testing.T) {
	source := dittotesting.Source(`
	|package source
	|
	|var first = 0
	|var second = 2
	|`)

	t.Run("names the file and its mutant count before running any of them", func(t *testing.T) {
		logger := fakelogger.New()
		repository := fakerepository.New(fakerepository.FS{"source.go": source})

		ditto.New(logger, repository, fakelaboratory.NewAlways(result.Ok("mutant died")), fakereporter.New()).
			Release(integerincrement.New())

		assert.Equal(t, []string{"┃ source.go — 2 mutants"}, logger.LoggedLines())
	})

	t.Run("says nothing about a file that yields no mutants", func(t *testing.T) {
		logger := fakelogger.New()
		repository := fakerepository.New(fakerepository.FS{"quiet.go": dittotesting.Source(`
		|package source
		|
		|var text = "value"
		|`)})

		ditto.New(logger, repository, fakelaboratory.NewAlways(result.Ok("mutant died")), fakereporter.New()).
			Release(integerincrement.New())

		assert.Empty(t, logger.LoggedLines())
	})
}

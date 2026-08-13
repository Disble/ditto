package laboratory_test

import (
	"testing"

	"github.com/Disble/ditto/internal/dittotesting"
	"github.com/Disble/ditto/internal/dittotesting/fakerepository"
	"github.com/Disble/ditto/internal/dittotesting/faketempdirectory"
	"github.com/Disble/ditto/internal/dittotesting/faketestrunner"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/laboratory"
	"github.com/Disble/ditto/internal/result"
	"github.com/stretchr/testify/assert"
)

func TestLaboratory(t *testing.T) {
	source := dittotesting.Source(`
	|package source
	|
	|var number = 1
	|`)

	sourceIntegerdecrementMutation1 := dittotesting.Source(`
	|package source
	|
	|var number = 0
	|`)

	tempRepository := fakerepository.NewTemporary()
	repository := fakerepository.New(
		fakerepository.FS{
			"readme.md": []byte("read me"),
			"source.go": source,
		},
		tempRepository,
	)

	fut := laboratory.New(
		faketestrunner.New(
			faketestrunner.NewResult("tmpdir-1", result.Ok("mutants died")),
		),
		faketempdirectory.NewFakeTemporaryDirectory("tmpdir"),
	).Test(
		repository,
		gomutatedfile.New("dummy-infection", "source.go", source, sourceIntegerdecrementMutation1),
	)

	t.Run("copy all files to temporary repository replacing the mutated file", func(t *testing.T) {
		actual := tempRepository.ListFiles()
		expected := fakerepository.FS{
			"readme.md": []byte("read me"),
			"source.go": sourceIntegerdecrementMutation1,
		}
		assert.Equal(t, expected, actual)
	})

	t.Run("removes the temporary repository used", func(t *testing.T) {
		assert.True(t, tempRepository.Removed())
	})

	t.Run("reports the result of the test runner", func(t *testing.T) {
		assert.Equal(t, result.Ok("mutants died"), fut.Await())
	})
}

package laboratory_test

import (
	"testing"

	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/dittotesting"
	"github.com/Disble/ditto/internal/dittotesting/fakerepository"
	"github.com/Disble/ditto/internal/dittotesting/faketempdirectory"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/laboratory"
	"github.com/Disble/ditto/internal/result"
	"github.com/stretchr/testify/assert"
)

// observingRunner records what the sandbox held at the moment the test command
// ran.
//
// That instant is the only one where the mutation is supposed to be on disk.
// Reading the sandbox after Test returns cannot show it any more, because the
// sandbox is restored before it goes back to the pool — so the property is
// captured while it holds rather than inferred afterwards.
type observingRunner struct {
	seen   fakerepository.FS
	answer result.Result[string]
}

func (r *observingRunner) Test(repository ditto.TemporaryRepository) result.Result[string] {
	if sandbox, ok := repository.(*fakerepository.FakeTemporaryRepository); ok {
		r.seen = sandbox.ListFiles()
	}

	return r.answer
}

func TestLaboratory(t *testing.T) {
	source := dittotesting.Source(`
	|package source
	|
	|var number = 1
	|`)

	mutated := dittotesting.Source(`
	|package source
	|
	|var number = 0
	|`)

	sandbox := fakerepository.NewTemporary()
	spare := fakerepository.NewTemporary()
	repository := fakerepository.New(
		fakerepository.FS{
			"readme.md": []byte("read me"),
			"source.go": source,
		},
		sandbox,
		spare,
	)

	runner := &observingRunner{answer: result.Ok("mutants died")}
	subject := laboratory.New(runner, faketempdirectory.NewFakeTemporaryDirectory("tmpdir"))

	fut := subject.Test(
		repository,
		gomutatedfile.New("dummy-infection", "source.go", source, mutated),
	)

	t.Run("the test command runs against every file, with the mutated one in place", func(t *testing.T) {
		assert.Equal(t, fakerepository.FS{
			"readme.md": []byte("read me"),
			"source.go": mutated,
		}, runner.seen)
	})

	t.Run("the sandbox is handed back with the mutation undone", func(t *testing.T) {
		// A sandbox released still carrying its mutation would make the next
		// mutant run against two at once, and that result would read as an
		// ordinary survivor rather than as a bug.
		assert.Equal(t, fakerepository.FS{
			"readme.md": []byte("read me"),
			"source.go": source,
		}, sandbox.ListFiles())
	})

	t.Run("the sandbox is kept, not removed, because the run owns it now", func(t *testing.T) {
		// Removal moved to the end of the run: rebuilding a sandbox per mutant
		// is the cost this pool exists to stop paying.
		assert.False(t, sandbox.Removed())
	})

	t.Run("reports the result of the test runner", func(t *testing.T) {
		assert.Equal(t, result.Ok("mutants died"), fut.Await())
	})

	t.Run("a second mutant reuses the sandbox instead of building another", func(t *testing.T) {
		subject.Test(
			repository,
			gomutatedfile.New("another-infection", "source.go", source, mutated),
		)

		assert.Empty(t, spare.ListFiles(), "a second sandbox was built where the first should have been reused")
	})
}

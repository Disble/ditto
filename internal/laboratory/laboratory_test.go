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

	// calls counts invocations so the first one — the baseline, run on unmutated
	// code before any mutant is scored — can answer green. A runner that reports
	// the command failing every single time is describing a suite that is red
	// before anything is mutated, which the laboratory now refuses.
	calls int
}

func (r *observingRunner) Test(repository ditto.TemporaryRepository) result.Result[string] {
	r.calls++

	if r.calls == 1 {
		return result.Err[string]("the suite passes on unmutated code")
	}

	if sandbox, ok := repository.(*fakerepository.FakeTemporaryRepository); ok {
		r.seen = sandbox.ListFiles()
	}

	return r.answer
}

// alwaysFailingRunner is a test command that fails whatever it is pointed at —
// `make` dying before it compiles anything, a missing toolchain, a broken
// fixture.
type alwaysFailingRunner struct{ calls int }

func (r *alwaysFailingRunner) Test(ditto.TemporaryRepository) result.Result[string] {
	r.calls++

	return result.Ok[string]("make: *** [makefile:13: /.hooks.log] Error 128")
}

// A command that fails on unmutated code fails on every mutant too, and ditto
// recognises a killed mutant by exactly that. So every mutant is scored killed
// and the report says 1.00 without naming a cause.
//
// Measured on ditto's own gate, run 31860409386: **431 of 431 mutants killed, a
// perfect score, in 5.46 seconds** — twelve milliseconds each, by a `make`
// invocation that never compiled anything. The gated path has refused this since
// it could read its own unselected run; the ordinary path could not tell the two
// apart. docs/experiments/false-perfect-score.md.
func TestLaboratoryRefusesARedBaseline(t *testing.T) {
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

	runner := &alwaysFailingRunner{}
	subject := laboratory.New(runner, faketempdirectory.NewFakeTemporaryDirectory("tmpdir"))
	repository := fakerepository.New(fakerepository.FS{"source.go": source}, fakerepository.NewTemporary())

	// The command's own output travels with the refusal, so a reader is told why
	// the suite is red rather than only that it is. Without it the message names
	// a red baseline and leaves the reader to guess which of a hundred reasons
	// it is, in a sandbox they cannot look inside — measured at four rounds of
	// detective work over an embedded file that the output would have named on
	// the first.
	assert.PanicsWithError(t,
		"ditto: the test command fails on unmutated code, so every mutant would be scored "+
			"killed; refusing to score against a red baseline\n\n"+
			"make: *** [makefile:13: /.hooks.log] Error 128",
		func() {
			subject.Test(repository, gomutatedfile.New("dummy-infection", "source.go", source, mutated))
		})

	// The baseline is what refused, so the mutant's own run never happened. A
	// guard that let the run through and complained afterwards would still have
	// paid for every mutant in the release.
	assert.Equal(t, 1, runner.calls, "the mutant ran anyway")
}

// The baseline costs one run of the test command per release, not one per
// mutant. That is the whole difference between a guard and a tax, and
// perf/baseline.json ratchets the number either way.
func TestLaboratoryChecksTheBaselineOnce(t *testing.T) {
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

	runner := &observingRunner{answer: result.Ok("mutants died")}
	subject := laboratory.New(runner, faketempdirectory.NewFakeTemporaryDirectory("tmpdir"))
	repository := fakerepository.New(
		fakerepository.FS{"source.go": source},
		fakerepository.NewTemporary(), fakerepository.NewTemporary(), fakerepository.NewTemporary(),
	)

	for range 3 {
		subject.Test(repository, gomutatedfile.New("dummy-infection", "source.go", source, mutated))
	}

	assert.Equal(t, 4, runner.calls, "want one baseline and three mutants")
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

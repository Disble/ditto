package laboratory_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/dittotesting"
	"github.com/Disble/ditto/internal/dittotesting/fakelogger"
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
	subject := laboratory.New(fakelogger.New(), runner, faketempdirectory.NewFakeTemporaryDirectory("tmpdir"))
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
	subject := laboratory.New(fakelogger.New(), runner, faketempdirectory.NewFakeTemporaryDirectory("tmpdir"))
	repository := fakerepository.New(
		fakerepository.FS{"source.go": source},
		fakerepository.NewTemporary(), fakerepository.NewTemporary(), fakerepository.NewTemporary(),
	)

	for range 3 {
		subject.Test(repository, gomutatedfile.New("dummy-infection", "source.go", source, mutated))
	}

	assert.Equal(t, 4, runner.calls, "want one baseline and three mutants")
}

// streamingRunner is a command that passes and prints what `go test -json`
// prints. Its stream is the only place the package scope of a release exists.
type streamingRunner struct {
	stream string
	calls  int
}

func (r *streamingRunner) Test(ditto.TemporaryRepository) result.Result[string] {
	r.calls++

	return result.Err[string](r.stream)
}

// scriptedToolchain is the toolchain's answer to "what does that compile", and
// it counts how often it was asked.
type scriptedToolchain struct {
	output []byte
	err    error
	asked  int
}

func (s *scriptedToolchain) Output(_, _ string, _ ...string) ([]byte, error) {
	s.asked++

	return s.output, s.err
}

// A release judges mutants from files the scope selected with one configured
// command, and when that command does not compile a mutant's package the mutant
// is unmeasured rather than badly tested: nothing the command runs can kill it.
//
// The scope comes free. The baseline run the laboratory already makes once per
// release prints the stream that names the packages the command executes, and
// the sandbox it ran in is the tree to resolve the rest against — so the answer
// costs one `go list` and no extra suite run, which the counts below hold.
func TestLaboratoryKnowsWhichPackagesTheCommandExecutes(t *testing.T) {
	root := "tmpdir-1"

	runner := &streamingRunner{stream: `{"Action":"start","Package":"example.com/mod/internal/observability/syncdiag"}` + "\n"}

	toolchain := &scriptedToolchain{output: []byte(filepath.Join(root, "internal", "observability", "readcap") + "\n" +
		filepath.Join(root, "internal", "observability", "syncdiag") + "\n")}

	defer laboratory.SetScopeRunnerForTest(toolchain)()

	subject := laboratory.New(fakelogger.New(), runner, faketempdirectory.NewFakeTemporaryDirectory("tmpdir"))
	repository := fakerepository.New(fakerepository.FS{}, fakerepository.NewTemporary())

	assert.True(t, subject.Executes(repository, "internal/observability/syncdiag/reader.go"),
		"the package the command names is not reported as executed")

	// The closure is the load-bearing half: a package the command does not name
	// but does compile into a test binary IS killable by that binary's tests, so
	// a set comparison against the names alone would accuse a correct run.
	assert.True(t, subject.Executes(repository, "internal/observability/readcap/reader.go"),
		"a package compiled into the named package's tests is not reported as executed")

	// The report's own case, and the whole point: staged, mutated, never built.
	assert.False(t, subject.Executes(repository, "internal/desktop/app_runtime_services.go"),
		"a package the command does not compile is reported as executed")

	assert.Equal(t, 1, toolchain.asked, "want one toolchain query per scope")
	assert.Equal(t, 1, runner.calls, "want one baseline, not one per question asked of the scope")
}

// A command that is not `go test -json` has no readable scope — make, gotestsum,
// a wrapper — and ditto refuses nothing over it. Two costs are held here: the
// scope is unknown rather than guessed, and the toolchain is never asked, because
// there is nothing to ask.
func TestLaboratoryRefusesNothingForACommandWithNoReadableScope(t *testing.T) {
	runner := &streamingRunner{stream: "ok  \texample.com/mod/calc\t0.2s\n"}
	toolchain := &scriptedToolchain{output: []byte("anything")}

	defer laboratory.SetScopeRunnerForTest(toolchain)()

	subject := laboratory.New(fakelogger.New(), runner, faketempdirectory.NewFakeTemporaryDirectory("tmpdir"))
	repository := fakerepository.New(fakerepository.FS{}, fakerepository.NewTemporary())

	assert.True(t, subject.Executes(repository, "internal/desktop/app.go"),
		"an unreadable command refused a mutant")
	assert.Zero(t, toolchain.asked, "the toolchain was asked about a command ditto could not read")
}

// And the other half of that rule: a scope that could not be resolved is not
// evidence of a mismatch either.
func TestLaboratoryRefusesNothingWhenTheToolchainCannotAnswer(t *testing.T) {
	runner := &streamingRunner{stream: `{"Action":"start","Package":"example.com/mod/calc"}` + "\n"}
	toolchain := &scriptedToolchain{err: errors.New("no go binary")}

	defer laboratory.SetScopeRunnerForTest(toolchain)()

	subject := laboratory.New(fakelogger.New(), runner, faketempdirectory.NewFakeTemporaryDirectory("tmpdir"))
	repository := fakerepository.New(fakerepository.FS{}, fakerepository.NewTemporary())

	assert.True(t, subject.Executes(repository, "internal/desktop/app.go"),
		"a scope that could not be resolved refused a mutant")
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
	subject := laboratory.New(fakelogger.New(), runner, faketempdirectory.NewFakeTemporaryDirectory("tmpdir"))

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

package gatedlaboratory_test

import (
	"strings"
	"testing"

	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/gatedlaboratory"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/gosourcefile"
	"github.com/Disble/ditto/internal/result"
	"github.com/stretchr/testify/assert"
)

const source = "package calc\n\nfunc Over(a, b int) bool {\n\treturn a > b\n}\n"

func TestGatedLaboratory(t *testing.T) {
	t.Run("runs a file's gateable mutants from one compilation", func(t *testing.T) {
		runner := &fakeRunner{built: true}
		delegate := &countingLaboratory{}
		lab := gatedlaboratory.NewWithRunner(delegate, fakeTemporary{}, runner)

		results := lab.TestAll(fakeRepository{}, mutantsOf(
			strings.Replace(source, "a > b", "a >= b", 1),
			strings.Replace(source, "a > b", "a <= b", 1),
		))

		assert.Len(t, results, 2)
		assert.Equal(t, 2, lab.Gated())
		assert.Equal(t, 0, lab.FellBack())
		assert.Equal(t, 0, delegate.calls, "nothing should have taken the old path")
		assert.Equal(t, []int{1, 2}, runner.selected)
	})

	// Measured: a file whose instrumented form does not compile has to be
	// survivable, not fatal. Under one shared build a single bad site would
	// otherwise take every other mutant in the run with it.
	t.Run("gives the whole file back to the old path when the build fails", func(t *testing.T) {
		runner := &fakeRunner{built: false}
		delegate := &countingLaboratory{}
		lab := gatedlaboratory.NewWithRunner(delegate, fakeTemporary{}, runner)

		results := lab.TestAll(fakeRepository{}, mutantsOf(
			strings.Replace(source, "a > b", "a >= b", 1),
			strings.Replace(source, "a > b", "a <= b", 1),
		))

		assert.Len(t, results, 2)
		assert.Equal(t, 0, lab.Gated())
		assert.Equal(t, 2, lab.FellBack())
		assert.Equal(t, 2, delegate.calls)
	})

	t.Run("sends a mutant it cannot gate to the old path", func(t *testing.T) {
		const arithmetic = "package calc\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n"

		runner := &fakeRunner{built: true}
		delegate := &countingLaboratory{}
		lab := gatedlaboratory.NewWithRunner(delegate, fakeTemporary{}, runner)

		lab.TestAll(fakeRepository{}, []*gomutatedfile.GoMutatedFile{
			gomutatedfile.New("Arithmetic", "calc/calc.go",
				[]byte(arithmetic), []byte(strings.Replace(arithmetic, "a + b", "a - b", 1))),
		})

		assert.Equal(t, 0, lab.Gated())
		assert.Equal(t, 1, lab.FellBack())
	})

	t.Run("takes the old path for a single mutant asked for on its own", func(t *testing.T) {
		delegate := &countingLaboratory{}
		lab := gatedlaboratory.NewWithRunner(delegate, fakeTemporary{}, &fakeRunner{built: true})

		lab.Test(fakeRepository{}, mutantsOf(strings.Replace(source, "a > b", "a >= b", 1))[0])

		assert.Equal(t, 1, delegate.calls)
		assert.Equal(t, 0, lab.Gated())
	})

	// H3 of docs/experiments/changed-scope.md. GoBuildRunner.Runs increments once
	// per Test, so a run the laboratory makes before selecting anything is
	// counted as a mutant's. Every ratio published from that counter carries it.
	t.Run("counts a run before any mutant is selected", func(t *testing.T) {
		runner := &fakeRunner{built: true}
		lab := gatedlaboratory.NewWithRunner(&countingLaboratory{}, fakeTemporary{}, runner)

		lab.TestAll(fakeRepository{}, mutantsOf(
			strings.Replace(source, "a > b", "a >= b", 1),
			strings.Replace(source, "a > b", "a < b", 1),
		))

		assert.Equal(t, 2, lab.Gated(), "both mutants gate")
		assert.Equal(t, 2, runner.tests-1,
			"runs %d for %d gated mutants: the extra one is the unselected baseline",
			runner.tests, lab.Gated())
	})

	// A run with no mutant selected takes every gate's original arm, so it is the
	// file's own suite. If that fails, every selected run fails too and every
	// mutant of the file is scored killed — a perfect score produced by a broken
	// suite, with nothing in the output naming the cause. Measured at 4 of 4
	// killed against 1 of 4 on the same mutants; docs/experiments/changed-scope.md.
	t.Run("refuses when the suite fails with no mutant selected", func(t *testing.T) {
		runner := &fakeRunner{built: true, redBaseline: true}
		lab := gatedlaboratory.NewWithRunner(&countingLaboratory{}, fakeTemporary{}, runner)

		assert.PanicsWithError(t,
			"ditto: calc/calc.go fails its own suite with no mutant selected, so every "+
				"mutant of it would be scored killed; refusing to score against a red baseline",
			func() {
				lab.TestAll(fakeRepository{}, mutantsOf(
					strings.Replace(source, "a > b", "a >= b", 1),
				))
			})
	})

	t.Run("scores normally when the suite passes with no mutant selected", func(t *testing.T) {
		runner := &fakeRunner{built: true}
		lab := gatedlaboratory.NewWithRunner(&countingLaboratory{}, fakeTemporary{}, runner)

		results := lab.TestAll(fakeRepository{}, mutantsOf(
			strings.Replace(source, "a > b", "a >= b", 1),
		))

		assert.Len(t, results, 1)
		assert.Equal(t, 1, lab.Gated())
	})

	t.Run("answers nothing for no mutants", func(t *testing.T) {
		lab := gatedlaboratory.NewWithRunner(&countingLaboratory{}, fakeTemporary{}, &fakeRunner{built: true})

		assert.Nil(t, lab.TestAll(fakeRepository{}, nil))
	})
}

func mutantsOf(mutants ...string) []*gomutatedfile.GoMutatedFile {
	files := make([]*gomutatedfile.GoMutatedFile, 0, len(mutants))
	for _, mutant := range mutants {
		files = append(files, gomutatedfile.New("Comparison", "calc/calc.go", []byte(source), []byte(mutant)))
	}

	return files
}

type fakeRunner struct {
	built    bool
	selected []int
	tests    int

	// redBaseline makes the first run — the one with no mutant selected —
	// report the suite failing. Ok is how a failing command arrives here, which
	// is also how a killed mutant arrives, and telling those two apart is the
	// whole point of reading it.
	redBaseline bool
}

func (r *fakeRunner) Select(mutant int) { r.selected = append(r.selected, mutant) }
func (r *fakeRunner) Built() bool       { return r.built }
func (r *fakeRunner) Test(ditto.TemporaryRepository) result.Result[string] {
	r.tests++

	if r.redBaseline && r.tests == 1 {
		return result.Ok("FAIL calc [build failed]")
	}

	return result.Err[string]("")
}

type countingLaboratory struct{ calls int }

func (l *countingLaboratory) Test(
	ditto.Repository,
	*gomutatedfile.GoMutatedFile,
) future.Future[result.Result[string]] {
	l.calls++

	return future.Resolved(result.Err[string](""))
}

type fakeTemporary struct{}

func (fakeTemporary) New() string { return "temporary" }

// fakeRepository is local rather than shared: this test only needs a sandbox to
// be handed one, and it records what was written into it.
type fakeRepository struct{}

func (fakeRepository) ListGoSourceFiles() []*gosourcefile.GoSourceFile { return nil }

func (fakeRepository) LinkAllToTemporaryRepository(string) ditto.TemporaryRepository {
	return &fakeSandbox{}
}

type fakeSandbox struct{ written map[string]string }

func (s *fakeSandbox) Root() string { return "sandbox" }
func (s *fakeSandbox) Remove()      {}

func (s *fakeSandbox) Overwrite(filePath string, data []byte) {
	if s.written == nil {
		s.written = map[string]string{}
	}

	s.written[filePath] = string(data)
}

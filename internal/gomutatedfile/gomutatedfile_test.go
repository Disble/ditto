package gomutatedfile_test

import (
	"strings"
	"testing"

	"github.com/Disble/ditto/internal/dittotesting/fakediffer"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/stretchr/testify/assert"
)

func TestGoMutatedFile(t *testing.T) {
	t.Run("prints a diff", func(t *testing.T) {
		diff := gomutatedfile.New(
			"some-infection",
			"some-path.go",
			[]byte("original"),
			[]byte("mutated"),
		).Diff(fakediffer.New())

		assert.Equal(t, []string{
			"From: some-path.go (original)",
			"To: some-path.go (mutated with 'some-infection')",
			"",
			"- original",
			"+ mutated",
		}, strings.Split(diff, "\n"))
	})

	// A survivor a reader cannot jump to is a survivor they have to hunt for. The
	// address is read off the bytes the mutant already carries: nothing parses
	// twice and no virus is asked where it struck.
	t.Run("addresses a mutant at the byte its mutation starts", func(t *testing.T) {
		mutant := gomutatedfile.New(
			"Arithmetic",
			"calc/calc.go",
			[]byte("package calc\n\nfunc f() int { return 1 + 2 }\n"),
			[]byte("package calc\n\nfunc f() int { return 1 - 2 }\n"),
		)

		assert.Equal(t, "calc/calc.go:3:25", mutant.Address())
		assert.Equal(t, "calc/calc.go:3:25 → Arithmetic", mutant.Label())
		assert.Equal(t, "+ → -", mutant.Change())
	})

	t.Run("counts lines and columns from one", func(t *testing.T) {
		mutant := gomutatedfile.New(
			"Comparison",
			"a.go",
			[]byte("a>b"),
			[]byte("a<b"),
		)

		assert.Equal(t, "a.go:1:2", mutant.Address())
		assert.Equal(t, "> → <", mutant.Change())
	})

	// Widening `>` to `>=` replaces nothing and inserts one byte after it, so the
	// address is the inserted byte's own column rather than the operator's. The
	// difference is minimal by construction; it is not the expression.
	t.Run("addresses an inserted byte where it lands", func(t *testing.T) {
		mutant := gomutatedfile.New("Comparison", "a.go", []byte("a>b"), []byte("a>=b"))

		assert.Equal(t, "a.go:1:3", mutant.Address())
		assert.Equal(t, "inserts =", mutant.Change())
	})

	t.Run("names a deletion as a deletion", func(t *testing.T) {
		mutant := gomutatedfile.New("Comparison", "a.go", []byte("a>=b"), []byte("a>b"))

		assert.Equal(t, "deletes =", mutant.Change())
	})

	// An insertion replaces nothing, and a reader told "→ break" with no left
	// side would read it as a rendering fault rather than as the mutation.
	t.Run("names the empty side of an insertion", func(t *testing.T) {
		mutant := gomutatedfile.New(
			"Range Break",
			"loop.go",
			[]byte("package a\n\nfunc f(xs []int) {\n\tfor range xs {\n\t\tprintln(1)\n\t}\n}\n"),
			[]byte("package a\n\nfunc f(xs []int) {\n\tfor range xs {\n\t\tbreak\n\t\tprintln(1)\n\t}\n}\n"),
		)

		assert.Equal(t, "loop.go:5:3", mutant.Address())
		assert.Equal(t, "inserts break", mutant.Change())
	})

	// The fallback is not decoration: a laboratory or a test may build a mutant
	// with no bytes, and a report that panicked on one would be worse than a
	// report with no address.
	t.Run("falls back to the path when no address can be derived", func(t *testing.T) {
		identical := gomutatedfile.New("dummy", "dummy.go", []byte("same"), []byte("same"))

		assert.Equal(t, "dummy.go", identical.Address())
		assert.Equal(t, "dummy.go → dummy", identical.Label())
		assert.Equal(t, "", identical.Change())

		empty := gomutatedfile.New("dummy", "dummy.go", nil, nil)

		assert.Equal(t, "dummy.go", empty.Address())
		assert.Equal(t, "dummy.go → dummy", empty.Label())
		assert.Equal(t, "", empty.Change())
	})
}

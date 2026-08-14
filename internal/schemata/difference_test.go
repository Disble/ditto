package schemata_test

import (
	"testing"

	"github.com/Disble/ditto/internal/schemata"
	"github.com/stretchr/testify/assert"
)

// A gate has to know which bytes a virus replaced and what it replaced them
// with. Asking the viruses would mean teaching this package what each of the
// fourteen does, and a second copy of that knowledge is a copy that goes stale.
// The mutated file already carries the answer.
func TestDifference(t *testing.T) {
	t.Run("reports the smallest range that differs, not the token", func(t *testing.T) {
		difference, found := schemata.Difference(
			[]byte("func f(a, b int) bool { return a > b }"),
			[]byte("func f(a, b int) bool { return a >= b }"),
		)

		// `>` to `>=` shares its first byte, so the smallest difference is an
		// `=` inserted after it — not the operator. Widening this to the
		// expression that encloses it needs the syntax tree and belongs to the
		// step that builds the gate, not here.
		assert.True(t, found)
		assert.Equal(t, 34, difference.Start)
		assert.Equal(t, 34, difference.End)
		assert.Equal(t, "", difference.Original)
		assert.Equal(t, "=", difference.Mutated)
	})

	t.Run("reports a replacement shorter than what it replaced", func(t *testing.T) {
		difference, found := schemata.Difference([]byte("x := 100"), []byte("x := 99"))

		assert.True(t, found)
		assert.Equal(t, 5, difference.Start)
		assert.Equal(t, 8, difference.End)
		assert.Equal(t, "100", difference.Original)
		assert.Equal(t, "99", difference.Mutated)
	})

	t.Run("reports an insertion, which has an empty original", func(t *testing.T) {
		difference, found := schemata.Difference([]byte("for range x {\n}"), []byte("for range x {\nbreak\n}"))

		assert.True(t, found)
		assert.Equal(t, "", difference.Original)
		assert.Equal(t, "break\n", difference.Mutated)
	})

	// Two edits collapse into one span covering both, and that span is not an
	// expression. Refusing it is the gate builder's job, with the syntax tree in
	// hand; refusing it here would need a real diff and would duplicate the
	// check. What this promises is only that one replacement of the reported
	// range reproduces the mutated file — which is true of the covering span too.
	t.Run("covers two separate edits with the span between them", func(t *testing.T) {
		difference, found := schemata.Difference(
			[]byte("a > b && c > d"),
			[]byte("a >= b && c >= d"),
		)

		assert.True(t, found)
		assert.Equal(t, " b && c >", difference.Original)
		assert.Equal(t, "= b && c >=", difference.Mutated)
	})

	t.Run("refuses two files that are the same", func(t *testing.T) {
		_, found := schemata.Difference([]byte("a > b"), []byte("a > b"))

		assert.False(t, found)
	})

	t.Run("replacing the reported range always reproduces the mutated file", func(t *testing.T) {
		for _, pair := range [][2]string{
			{"a > b", "a >= b"},
			{"x := 100", "x := 99"},
			{"for range x {\n}", "for range x {\nbreak\n}"},
			{"a > b && c > d", "a >= b && c >= d"},
			{"return n", "return n + 1"},
		} {
			original, mutated := []byte(pair[0]), []byte(pair[1])

			difference, found := schemata.Difference(original, mutated)
			assert.True(t, found)

			rebuilt := string(original[:difference.Start]) + difference.Mutated + string(original[difference.End:])
			assert.Equal(t, pair[1], rebuilt)
		}
	})
}

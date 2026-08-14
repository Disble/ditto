package schemata_test

import (
	"testing"

	"github.com/Disble/ditto/internal/schemata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gateOf runs the whole path a caller would: read the difference off the two
// files, then widen it to the expression a gate can stand in for.
func gateOf(t *testing.T, original, mutated string) (schemata.Gate, bool) {
	t.Helper()

	difference, found := schemata.Difference([]byte(original), []byte(mutated))
	require.True(t, found)

	return schemata.Expand([]byte(original), difference)
}

func TestExpand(t *testing.T) {
	t.Run("widens an operator change to the comparison around it", func(t *testing.T) {
		gate, admitted := gateOf(t,
			"package p\n\nfunc f(a, b int) bool { return a > b }\n",
			"package p\n\nfunc f(a, b int) bool { return a >= b }\n",
		)

		assert.True(t, admitted)
		assert.Equal(t, schemata.Boolean, gate.Kind)
		assert.Equal(t, "a > b", gate.Original)
		assert.Equal(t, "a >= b", gate.Mutated)
	})

	// loopcondition replaces the whole condition rather than an operator inside
	// it, and it matches *ast.ForStmt — a statement — while mutating a bool
	// expression. The category belongs to what is mutated, not to the node the
	// virus matched on.
	t.Run("widens a replaced condition to the condition", func(t *testing.T) {
		gate, admitted := gateOf(t,
			"package p\n\nfunc f(n int) { for i := 0; i < n; i++ {\n} }\n",
			"package p\n\nfunc f(n int) { for i := 0; 0 != 0; i++ {\n} }\n",
		)

		assert.True(t, admitted)
		assert.Equal(t, schemata.Boolean, gate.Kind)
		assert.Equal(t, "i < n", gate.Original)
		assert.Equal(t, "0 != 0", gate.Mutated)
	})

	t.Run("widens a changed integer to the literal", func(t *testing.T) {
		gate, admitted := gateOf(t,
			"package p\n\nfunc f() int { return 100 }\n",
			"package p\n\nfunc f() int { return 99 }\n",
		)

		assert.True(t, admitted)
		assert.Equal(t, schemata.Integer, gate.Kind)
		assert.Equal(t, "100", gate.Original)
		assert.Equal(t, "99", gate.Mutated)
	})

	// Arithmetic yields a number, and Go has no conditional expression, so the
	// boolean gate cannot stand there. A generated function could, but nothing
	// has measured whether it reaches the same verdicts, so it is not admitted.
	t.Run("refuses arithmetic, which is an expression but not a bool", func(t *testing.T) {
		_, admitted := gateOf(t,
			"package p\n\nfunc f(a, b int) int { return a + b }\n",
			"package p\n\nfunc f(a, b int) int { return a - b }\n",
		)

		assert.False(t, admitted)
	})

	t.Run("refuses a statement, which no expression can replace", func(t *testing.T) {
		_, admitted := gateOf(t,
			"package p\n\nfunc f(xs []int) { for range xs {\nprintln(1)\n} }\n",
			"package p\n\nfunc f(xs []int) { for range xs {\nbreak\nprintln(1)\n} }\n",
		)

		assert.False(t, admitted)
	})

	// Two edits come back as the span covering both, which is not an expression.
	// This is the check Difference deliberately does not make.
	t.Run("refuses a span that covers two edits", func(t *testing.T) {
		_, admitted := gateOf(t,
			"package p\n\nfunc f(a, b, c, d int) bool { return a > b && c > d }\n",
			"package p\n\nfunc f(a, b, c, d int) bool { return a >= b && c >= d }\n",
		)

		assert.False(t, admitted)
	})

	t.Run("refuses an original that does not parse", func(t *testing.T) {
		broken := []byte("package p\n\nvar x = (\n")

		_, admitted := schemata.Expand(broken, schemata.Replacement{Start: 19, End: 20, Mutated: "1"})
		assert.False(t, admitted)
	})

	// A gate splices the mutated text back into the file. Under one shared
	// compilation, one piece of text that is not an expression fails the single
	// build and takes every other mutant in the run with it.
	t.Run("refuses a mutation that is not an expression on its own", func(t *testing.T) {
		source := []byte("package p\n\nfunc f(a, b int) bool { return a > b }\n")

		difference, found := schemata.Difference(source, []byte("package p\n\nfunc f(a, b int) bool { return a > ) }\n"))
		require.True(t, found)

		_, admitted := schemata.Expand(source, difference)
		assert.False(t, admitted)
	})
}

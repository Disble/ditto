package schemata_test

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/Disble/ditto/internal/schemata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const comparisonSource = "package p\n\nfunc f(a, b int) bool { return a > b }\n"

func TestInstrument(t *testing.T) {
	t.Run("substitutes a comparison with the selecting expression", func(t *testing.T) {
		source := []byte(comparisonSource)
		gate, admitted := gateOf(t, comparisonSource, "package p\n\nfunc f(a, b int) bool { return a >= b }\n")
		require.True(t, admitted)

		instrumented, applied := schemata.Instrument(source, []schemata.Gate{gate})

		assert.Equal(t, 1, gatedCount(applied))
		assert.Contains(t, string(instrumented),
			"((dittoMutant == 1 && a >= b) || (dittoMutant != 1 && a > b))")
	})

	t.Run("substitutes an integer with a call and generates its function", func(t *testing.T) {
		const source = "package p\n\nfunc f() int { return 100 }\n"

		gate, admitted := gateOf(t, source, "package p\n\nfunc f() int { return 99 }\n")
		require.True(t, admitted)

		instrumented, applied := schemata.Instrument([]byte(source), []schemata.Gate{gate})

		assert.Equal(t, 1, gatedCount(applied))
		assert.Contains(t, string(instrumented), "return dittoInteger1()")
		assert.Contains(t, string(instrumented), "func dittoInteger1() int {")
	})

	// Measured: `if hidden > 0` is one comparison and one literal, and the
	// literal is inside the comparison. Rewriting both spliced the outer one
	// over the inner one and produced a decapitated identifier — a file that
	// does not parse. The inner gate is dropped and keeps the per-mutant path.
	t.Run("drops a gate that sits inside another", func(t *testing.T) {
		const source = "package p\n\nfunc f(hidden int) bool { return hidden > 0 }\n"

		outer, admitted := gateOf(t, source, "package p\n\nfunc f(hidden int) bool { return hidden >= 0 }\n")
		require.True(t, admitted)

		inner, admitted := gateOf(t, source, "package p\n\nfunc f(hidden int) bool { return hidden > 1 }\n")
		require.True(t, admitted)

		instrumented, applied := schemata.Instrument([]byte(source), []schemata.Gate{outer, inner})

		assert.Equal(t, 1, gatedCount(applied))
		assert.Equal(t, []int{1, 0}, applied)
		assert.NotContains(t, string(instrumented), "dittoInteger")
	})

	t.Run("instruments every gate in a file at once", func(t *testing.T) {
		const source = "package p\n\nfunc f(a, b, c, d int) bool {\n\tif a > b {\n\t\treturn c > d\n\t}\n\n\treturn false\n}\n"

		first, admitted := gateOf(t, source, strings.Replace(source, "a > b", "a >= b", 1))
		require.True(t, admitted)

		second, admitted := gateOf(t, source, strings.Replace(source, "c > d", "c >= d", 1))
		require.True(t, admitted)

		instrumented, applied := schemata.Instrument([]byte(source), []schemata.Gate{first, second})

		assert.Equal(t, 2, gatedCount(applied))
		assert.Contains(t, string(instrumented), "dittoMutant == 1")
		assert.Contains(t, string(instrumented), "dittoMutant == 2")
	})

	// The gates arrive in whatever order the viruses produced them. Rewriting
	// has to be back to front, and getting that wrong shifts every offset after
	// the first edit.
	t.Run("does not depend on the order the gates arrive in", func(t *testing.T) {
		const source = "package p\n\nfunc f(a, b, c, d int) bool {\n\tif a > b {\n\t\treturn c > d\n\t}\n\n\treturn false\n}\n"

		first, _ := gateOf(t, source, strings.Replace(source, "a > b", "a >= b", 1))
		second, _ := gateOf(t, source, strings.Replace(source, "c > d", "c >= d", 1))

		forwards, _ := schemata.Instrument([]byte(source), []schemata.Gate{first, second})
		backwards, _ := schemata.Instrument([]byte(source), []schemata.Gate{second, first})

		assert.Equal(t, string(forwards), string(backwards))
	})

	t.Run("what it writes always parses", func(t *testing.T) {
		const source = "package p\n\nfunc f(a, b int) bool {\n\tif a > b {\n\t\treturn a > 10\n\t}\n\n\treturn b > 20\n}\n"

		gates := []schemata.Gate{}

		for _, pair := range [][2]string{{"a > b", "a >= b"}, {"a > 10", "a >= 10"}, {"b > 20", "b >= 20"}, {"10", "11"}} {
			gate, admitted := gateOf(t, source, strings.Replace(source, pair[0], pair[1], 1))
			require.True(t, admitted)

			gates = append(gates, gate)
		}

		instrumented, applied := schemata.Instrument([]byte(source), gates)
		assert.NotEmpty(t, applied)

		_, err := parser.ParseFile(token.NewFileSet(), "", instrumented, parser.AllErrors)
		assert.NoError(t, err, string(instrumented))
	})

	t.Run("leaves a file with no gates exactly as it was", func(t *testing.T) {
		instrumented, applied := schemata.Instrument([]byte(comparisonSource), nil)

		assert.Equal(t, 0, gatedCount(applied))
		assert.Equal(t, comparisonSource, string(instrumented))
	})
}

// gatedCount is how many of the gates handed in were actually instrumented; a
// zero selector means the mutant keeps its own file and its own compilation.
func gatedCount(selectors []int) int {
	count := 0

	for _, id := range selectors {
		if id != 0 {
			count++
		}
	}

	return count
}

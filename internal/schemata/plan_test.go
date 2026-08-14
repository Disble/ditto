package schemata_test

import (
	"strings"
	"testing"

	"github.com/Disble/ditto/internal/schemata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const planSource = "package p\n\nfunc f(a, b int) bool {\n\treturn a > b\n}\n"

func TestPlan(t *testing.T) {
	// comparison, comparisoninvert and comparisonreplace all infect the same
	// *ast.BinaryExpr, so one site carries several mutants. Every experiment
	// before this one used a single virus and never saw it.
	t.Run("gates several mutants of one site", func(t *testing.T) {
		planned := schemata.Plan([]byte(planSource), [][]byte{
			[]byte(strings.Replace(planSource, "a > b", "a >= b", 1)),
			[]byte(strings.Replace(planSource, "a > b", "a <= b", 1)),
			[]byte(strings.Replace(planSource, "a > b", "true", 1)),
		})

		assert.Equal(t, []int{1, 2, 3}, planned.Selector)
		assert.Contains(t, string(planned.Instrumented), "dittoMutant == 1 && a >= b")
		assert.Contains(t, string(planned.Instrumented), "dittoMutant == 2 && a <= b")
		assert.Contains(t, string(planned.Instrumented), "dittoMutant == 3 && true")
		assert.Contains(t, string(planned.Instrumented),
			"dittoMutant != 1 && dittoMutant != 2 && dittoMutant != 3 && a > b")
	})

	t.Run("reports zero for a mutant it cannot gate", func(t *testing.T) {
		const source = "package p\n\nfunc f(a, b int) int {\n\treturn a + b\n}\n"

		planned := schemata.Plan([]byte(source), [][]byte{
			[]byte(strings.Replace(source, "a + b", "a - b", 1)),
		})

		assert.Equal(t, []int{0}, planned.Selector)
		assert.Equal(t, source, string(planned.Instrumented))
	})

	t.Run("keeps one selector per mutant, in the order they arrived", func(t *testing.T) {
		const source = "package p\n\nfunc f(a, b int) bool {\n\tif a > b {\n\t\treturn a > 10\n\t}\n\n\treturn false\n}\n"

		planned := schemata.Plan([]byte(source), [][]byte{
			[]byte(strings.Replace(source, "a > 10", "a >= 10", 1)),
			[]byte(strings.Replace(source, "a + b", "a - b", 1)), // no such text: an identical file
			[]byte(strings.Replace(source, "a > b", "a >= b", 1)),
		})

		require.Len(t, planned.Selector, 3)
		assert.Equal(t, 0, planned.Selector[1], "an unchanged mutant cannot be gated")
		assert.NotEqual(t, 0, planned.Selector[0])
		assert.NotEqual(t, 0, planned.Selector[2])
		assert.NotEqual(t, planned.Selector[0], planned.Selector[2])
	})

	// GoInfectedFile.Mutate renders the mutant with format.Node while the
	// original is the file's own bytes. For a gofmt'd file those agree; for one
	// that is not, the difference carries formatting as well as the mutation and
	// there is no gate in it. Refusing is the safe answer, and it is the reason
	// the caller has to hand in matching renderings.
	t.Run("refuses a difference that is only formatting", func(t *testing.T) {
		planned := schemata.Plan(
			[]byte("package p\n\nfunc f(a, b int) bool {\n\treturn a  >  b\n}\n"),
			[][]byte{[]byte(planSource)},
		)

		assert.Equal(t, []int{0}, planned.Selector)
	})

	t.Run("gates nothing when there are no mutants", func(t *testing.T) {
		planned := schemata.Plan([]byte(planSource), nil)

		assert.Empty(t, planned.Selector)
		assert.Equal(t, planSource, string(planned.Instrumented))
	})
}

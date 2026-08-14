package schemata_test

import (
	"strings"
	"testing"

	"github.com/Disble/ditto/internal/schemata"
	"github.com/stretchr/testify/assert"
)

// A gate for an integer is a call, and a call is never a constant. Where Go
// requires one, the instrumented file does not compile — and under a single
// shared build that failure takes every other mutant in the run with it.
//
// Two of the three failures measured on real code are visible in the syntax
// tree. The third, `float64(ms)/1000`, is not: it needs to know what type the
// expression around it wants. Refusing what can be seen is worth doing anyway,
// because each one refused here is a compilation not wasted.
func TestConstantContexts(t *testing.T) {
	refused := []struct {
		name   string
		source string
		from   string
		to     string
	}{
		{
			name:   "a constant declaration",
			source: "package p\n\nconst width = 70\n",
			from:   "70", to: "71",
		},
		{
			name:   "a constant inside a grouped declaration",
			source: "package p\n\nconst (\n\tfirst  = 1\n\tsecond = 15\n)\n",
			from:   "15", to: "16",
		},
		{
			name:   "an array length",
			source: "package p\n\nvar buffer [8]byte\n",
			from:   "8", to: "9",
		},
	}

	for _, one := range refused {
		t.Run("refuses an integer in "+one.name, func(t *testing.T) {
			planned := schemata.Plan(
				[]byte(one.source),
				[][]byte{[]byte(strings.Replace(one.source, one.from, one.to, 1))},
			)

			assert.Equal(t, []int{0}, planned.Selector)
			assert.Equal(t, one.source, string(planned.Instrumented))
		})
	}

	t.Run("still gates an ordinary integer", func(t *testing.T) {
		const source = "package p\n\nfunc f() int {\n\treturn 70\n}\n"

		planned := schemata.Plan([]byte(source), [][]byte{
			[]byte(strings.Replace(source, "70", "71", 1)),
		})

		assert.Equal(t, []int{1}, planned.Selector)
	})

	// The comparison gate is an expression of the same type, not a call, but it
	// still reads a variable and so is still not a constant.
	t.Run("refuses a comparison in a constant declaration", func(t *testing.T) {
		const source = "package p\n\nconst ok = 1 > 0\n"

		planned := schemata.Plan([]byte(source), [][]byte{
			[]byte(strings.Replace(source, "1 > 0", "1 >= 0", 1)),
		})

		assert.Equal(t, []int{0}, planned.Selector)
	})
}

// This test lives inside the package rather than beside it, which testpackage
// objects to and which is the point: the question it answers is what the set of
// viruses ditto actually ships does, and `defaultOptions` is where that set
// lives. Restating the fourteen of them in an external test would be a second
// copy of a list that already exists, and a second copy is one that goes stale
// without anything failing — the same reason schemata reads a mutation off the
// bytes instead of keeping its own table of what each virus replaces.
package ditto //nolint:testpackage

import (
	"testing"

	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/gosourcefile"
)

// TestSurvivorsAreDistinguishable re-measures, through the shipped code, what
// docs/experiments/mutant-address.md measured through a probe.
//
// The probe walked the tree a second time so it could compare two mechanisms.
// This does not: it asks the default virus set for mutants exactly as a release
// does, and reads the line the report would print off each one. A result carried
// from a prototype into shipped code without being re-measured is a result about
// the prototype.
//
// It is also the guard for backlog entry 1. Over 135 mutants of four real files,
// 129 could not be told apart from another mutant by what ditto printed. Nothing
// refused that, which is why it survived long enough to be written down.
func TestSurvivorsAreDistinguishable(t *testing.T) {
	t.Parallel()

	fixtures := map[string]string{
		// Two viruses at one operator, plus a widening that inserts a byte and
		// replaces nothing.
		"comparisons": "package a\n\nfunc f(x, y int) bool {\n\treturn x > y\n}\n",

		// `x || y || z` parses as `(x || y) || z`, and comparisonreplace rewrites
		// the left operand of both — same byte, same replacement text, same
		// virus. It is the one shape measured where the address and the virus
		// name together are not enough, and it is why the change is printed
		// beside the address instead of being left inside the rendered diff.
		"nested disjunction": "package a\n\nfunc f(x, y, z bool) bool {\n\treturn x || y || z\n}\n",

		// An insertion that carries indentation with it, and a loop, so the
		// statement-shaped viruses enter the population.
		"loop with a body": "package a\n\nfunc f(xs []int) int {\n\ttotal := 0\n\tfor _, x := range xs {\n\t\ttotal += x\n\t}\n\n\treturn total\n}\n",
	}

	for name, source := range fixtures {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			path := name + ".go"
			mutants := mutantsOf(t, path, source)

			printed, addressAndVirus := tally(t, path, mutants)

			refuseCollisions(t, printed)

			// The change is not decoration, and this is what says so: on the
			// nested disjunction two mutants share an address and a virus name,
			// and only the change separates them. A fixture that stopped
			// producing that shape would let the change be dropped from the
			// printed line without anything going red — so the shape itself is
			// asserted, independently of what is printed from it.
			if name == "nested disjunction" && len(addressAndVirus) == len(mutants) {
				t.Fatalf("this fixture no longer puts two mutants at one address, so it cannot show "+
					"that the change text is load-bearing: %v", addressAndVirus)
			}
		})
	}
}

// tally counts the printed survivor lines and, separately, the labels alone —
// the second is what the nested disjunction needs to prove the change is
// load-bearing. It also refuses a mutant whose address never got past its path.
func tally(t *testing.T, path string, mutants []*gomutatedfile.GoMutatedFile) (map[string]int, map[string]int) {
	t.Helper()

	printed := map[string]int{}
	addressAndVirus := map[string]int{}

	for _, mutant := range mutants {
		line := survivorLine(mutant)

		printed[line]++
		addressAndVirus[mutant.Label()]++

		if mutant.Address() == path {
			t.Errorf("mutant %q carries no position, only its path", line)
		}
	}

	return printed, addressAndVirus
}

func refuseCollisions(t *testing.T, printed map[string]int) {
	t.Helper()

	for line, count := range printed {
		if count > 1 {
			t.Errorf("%d mutants print the identical line %q; a reader cannot tell them apart", count, line)
		}
	}
}

// survivorLine is what the console reporter prints for one survivor.
func survivorLine(mutant *gomutatedfile.GoMutatedFile) string {
	line := mutant.Label()
	if change := mutant.Change(); change != "" {
		line += " (" + change + ")"
	}

	return line
}

func mutantsOf(t *testing.T, path, source string) []*gomutatedfile.GoMutatedFile {
	t.Helper()

	infected := gosourcefile.New(path, []byte(source)).Incubate(defaultOptions.Viruses...)
	if len(infected) == 0 {
		t.Fatalf("fixture %s produced no mutants, so it asserts nothing", path)
	}

	mutants := make([]*gomutatedfile.GoMutatedFile, 0, len(infected))
	for _, one := range infected {
		mutants = append(mutants, one.Mutate())
	}

	return mutants
}

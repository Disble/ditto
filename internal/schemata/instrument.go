package schemata

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// selector is the variable every gate reads. Ids start at 1, so the zero value —
// which is what an unset or unparsable environment gives — selects nothing and
// the file behaves as it did before instrumenting.
const selector = "dittoMutant"

// Instrument rewrites every gate into one file that chooses a mutant at run
// time, and reports which gates it applied.
//
// A gate that is not applied is not lost: it keeps the path ditto has always
// taken, its own file and its own compilation. The count matters to the caller,
// because the ids the gates are given here are the ids the run has to select by.
func Instrument(source []byte, gates []Gate) ([]byte, []Gate) {
	applied := independent(gates)
	if len(applied) == 0 {
		return source, nil
	}

	// Numbered by where they sit, not by the order the viruses produced them.
	// An id is how a run asks for one mutant, so the same site has to answer to
	// the same number however the gates arrived.
	sort.Slice(applied, func(i, j int) bool { return applied[i].Start < applied[j].Start })

	// Back to front, so an edit never shifts the offsets of one still to come.
	// Applying them in the order the viruses produced them moved every offset
	// after the first edit and spliced one gate through the middle of another.
	ordered := make([]Gate, len(applied))
	copy(ordered, applied)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Start > ordered[j].Start })

	out := string(source)
	helpers := strings.Builder{}

	for _, gate := range ordered {
		id := identify(applied, gate)

		expression, helper := substitution(gate, id)
		helpers.WriteString(helper)

		out = out[:gate.Start] + expression + out[gate.End:]
	}

	return []byte(declare(out) + helpers.String()), applied
}

// substitution is the expression that replaces the site, and the function it
// needs, if any.
func substitution(gate Gate, id int) (string, string) {
	if gate.Kind == Integer {
		name := "dittoInteger" + strconv.Itoa(id)

		return name + "()", fmt.Sprintf(
			"\nfunc %s() int {\n\tif %s == %d {\n\t\treturn %s\n\t}\n\n\treturn %s\n}\n",
			name, selector, id, gate.Mutated, gate.Original,
		)
	}

	// Short-circuiting reaches exactly one side, so each operand is evaluated
	// once either way. A form that evaluated both would change any expression
	// whose operands have side effects, and `next() > limit()` would advance
	// twice.
	return fmt.Sprintf("((%s == %d && %s) || (%s != %d && %s))",
		selector, id, gate.Mutated, selector, id, gate.Original), ""
}

// independent drops any gate contained in another.
//
// The rewrite splices text using the original offsets, so an outer gate written
// after an inner one overwrites it. Measured on `hidden > 0`, where the literal
// sits inside the comparison: the result was a decapitated identifier and a file
// that did not parse. Which of the two survives is decided by position, so the
// outer one does, and the inner one keeps the per-mutant path.
func independent(gates []Gate) []Gate {
	kept := []Gate{}

	for _, gate := range gates {
		if !containedInAny(gate, gates) {
			kept = append(kept, gate)
		}
	}

	return kept
}

func containedInAny(gate Gate, gates []Gate) bool {
	for _, other := range gates {
		if other.Start == gate.Start && other.End == gate.End {
			continue
		}

		if other.Start <= gate.Start && gate.End <= other.End {
			return true
		}
	}

	return false
}

// identify numbers a gate by its position among the applied gates, so the ids do
// not depend on the order the viruses happened to produce them in.
func identify(applied []Gate, gate Gate) int {
	for i, candidate := range applied {
		if candidate.Start == gate.Start && candidate.End == gate.End {
			return i + 1
		}
	}

	return 0
}

// declare adds the selector: an import block right after the package clause and
// the variable at the end of the file. Go requires every import to appear before
// any other declaration, so the variable cannot sit between the new import block
// and the ones the file already had.
func declare(source string) string {
	end := strings.Index(source, "\n")

	for offset := 0; offset < len(source); {
		lineEnd := strings.Index(source[offset:], "\n")
		if lineEnd < 0 {
			lineEnd = len(source) - offset
		}

		if strings.HasPrefix(strings.TrimSpace(source[offset:offset+lineEnd]), "package ") {
			end = offset + lineEnd

			break
		}

		offset += lineEnd + 1
	}

	imports := "\n\nimport (\n\tdittoos \"os\"\n\tdittostrconv \"strconv\"\n)\n"
	variable := fmt.Sprintf("\nvar %s, _ = dittostrconv.Atoi(dittoos.Getenv(\"DITTO_MUTANT\"))\n", selector)

	return source[:end] + imports + source[end:] + variable
}

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

// site is one expression and every mutant of it.
//
// Several viruses infect the same node: comparison, comparisoninvert and
// comparisonreplace all rewrite the same *ast.BinaryExpr, so one expression
// carries three mutants and one gate has to choose between all of them.
type site struct {
	start, end int
	kind       Kind
	original   string
	mutants    []Gate
	ids        []int
}

// Instrument rewrites every gate into one file that chooses a mutant at run
// time, and returns the selector value for each gate it was given — zero for one
// it could not gate.
//
// A gate that gets zero is not lost: it keeps the path ditto has always taken,
// its own file and its own compilation. Every mutant that worked before still
// works, which is the constraint this whole change is under.
func Instrument(source []byte, gates []Gate) ([]byte, []int) {
	ids := make([]int, len(gates))

	sites := group(gates, ids)
	if len(sites) == 0 {
		return source, ids
	}

	// Back to front, so an edit never shifts the offsets of one still to come.
	// Applying them in the order the viruses produced them moved every offset
	// after the first edit and spliced one gate through the middle of another.
	rewritten := make([]site, len(sites))
	copy(rewritten, sites)
	sort.Slice(rewritten, func(i, j int) bool { return rewritten[i].start > rewritten[j].start })

	out := string(source)
	helpers := strings.Builder{}

	for _, one := range rewritten {
		expression, helper := substitution(one)
		helpers.WriteString(helper)

		out = out[:one.start] + expression + out[one.end:]
	}

	return []byte(declare(out) + helpers.String()), ids
}

// group collects the gates into sites, drops any site contained in another, and
// numbers the survivors. It fills ids in place, so a caller keeps one selector
// per gate it handed in.
//
// Sites are numbered by where they sit and mutants by the order they arrived in,
// so the same expression answers to the same numbers however the gates reached
// this function.
func group(gates []Gate, ids []int) []site {
	sites := []site{}

	for index, gate := range gates {
		if containedInAny(gate, gates) {
			continue
		}

		at := -1

		for i := range sites {
			if sites[i].start == gate.Start && sites[i].end == gate.End {
				at = i

				break
			}
		}

		if at < 0 {
			sites = append(sites, site{
				start: gate.Start, end: gate.End,
				kind: gate.Kind, original: gate.Original,
			})
			at = len(sites) - 1
		}

		sites[at].mutants = append(sites[at].mutants, gate)
		sites[at].ids = append(sites[at].ids, index)
	}

	sort.SliceStable(sites, func(i, j int) bool { return sites[i].start < sites[j].start })

	next := 1

	for i := range sites {
		for j, index := range sites[i].ids {
			ids[index] = next
			sites[i].ids[j] = next
			next++
		}
	}

	return sites
}

// substitution is the expression that replaces the site, and the function it
// needs, if any.
func substitution(one site) (string, string) {
	if one.kind == Integer {
		return integerCall(one)
	}

	// Short-circuiting reaches exactly one arm, so each operand is evaluated
	// once whichever mutant is selected. A form that evaluated more than one
	// would change any expression whose operands have side effects, and
	// `next() > limit()` would advance twice.
	arms := make([]string, 0, len(one.mutants)+1)
	unselected := make([]string, 0, len(one.mutants))

	for i, mutant := range one.mutants {
		arms = append(arms, fmt.Sprintf("(%s == %d && %s)", selector, one.ids[i], mutant.Mutated))
		unselected = append(unselected, fmt.Sprintf("%s != %d", selector, one.ids[i]))
	}

	arms = append(arms, "("+strings.Join(unselected, " && ")+" && "+one.original+")")

	return "(" + strings.Join(arms, " || ") + ")", ""
}

// integerCall is the only runtime selection Go offers for something that is not
// a bool: a call. A call is never a constant, so this cannot stand where Go
// requires one — measured at 3 sites in 91.
func integerCall(one site) (string, string) {
	name := "dittoInteger" + strconv.Itoa(one.ids[0])
	body := strings.Builder{}

	for i, mutant := range one.mutants {
		fmt.Fprintf(&body, "\tif %s == %d {\n\t\treturn %s\n\t}\n\n", selector, one.ids[i], mutant.Mutated)
	}

	return name + "()", fmt.Sprintf("\nfunc %s() int {\n%s\treturn %s\n}\n", name, body.String(), one.original)
}

// containedInAny reports whether a gate sits inside a different expression that
// is also gated.
//
// The rewrite splices text using the original offsets, so an outer gate written
// after an inner one overwrites it. Measured on `hidden > 0`, where the literal
// sits inside the comparison: the result was a decapitated identifier and a file
// that did not parse. The outer one survives, decided by position, and the inner
// one keeps the per-mutant path.
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

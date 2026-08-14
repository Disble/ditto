package schemata

// Planned is the gated form of one source file.
type Planned struct {
	// Instrumented is the file to compile once. It equals the original when
	// nothing could be gated.
	Instrumented []byte

	// Selector holds one value per mutant handed in, in the same order: the
	// number to put in the environment to select it, or zero for a mutant that
	// has to keep its own file and its own compilation.
	Selector []int
}

// Plan turns a file and its mutants into one instrumented file.
//
// This is where ditto's own output meets the gated path: GoInfectedFile.Mutate
// already produces, per mutant, the original bytes and the mutated ones, and
// every mutation this can gate is read off that pair. No virus is re-implemented
// here, and there is no table of operators to fall out of step with the fourteen
// that own them.
//
// The mutants must be rendered the same way the original is. Mutate prints the
// tree with format.Node while the original is the file's own bytes, so for a
// file that is not gofmt'd the difference between them carries formatting as
// well as the mutation. There is no gate in such a difference, and the answer is
// a zero selector rather than a wrong gate.
func Plan(original []byte, mutants [][]byte) Planned {
	gates := make([]Gate, 0, len(mutants))
	owner := make([]int, 0, len(mutants))

	for index, mutant := range mutants {
		difference, found := Difference(original, mutant)
		if !found {
			continue
		}

		gate, admitted := Expand(original, difference)
		if !admitted {
			continue
		}

		gates = append(gates, gate)
		owner = append(owner, index)
	}

	instrumented, ids := Instrument(original, gates)

	selector := make([]int, len(mutants))
	for i, id := range ids {
		selector[owner[i]] = id
	}

	return Planned{Instrumented: instrumented, Selector: selector}
}

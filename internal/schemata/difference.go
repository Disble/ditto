// Package schemata turns per-mutant source files into one instrumented file
// that selects a mutant at run time, so a release compiles once instead of once
// per mutant.
//
// The measured reason it exists: the fixed cost of starting `go test` is
// 750-950 ms per mutant regardless of how fast the suite is, and it is the
// dominant cost of a run. See docs/experiments/test-invocation.md.
package schemata

import "bytes"

// Replacement is the one stretch of bytes a mutation replaced, in the original's
// coordinates. An insertion has an empty Original; a deletion, an empty Mutated.
type Replacement struct {
	Start, End int
	Original   string
	Mutated    string
}

// Difference reports the single range in which two versions of a file differ.
//
// It reads the answer off the mutated file rather than asking the viruses what
// they did. Teaching this package what each of the fourteen viruses replaces
// would be a second copy of knowledge that already lives in one place, and a
// second copy is one that goes stale without anything failing.
//
// It reports false only when the two files are identical. Two edits in one file
// are reported as the single span that covers both, which is not something a
// gate can use — but refusing it here would need a real diff, and the check
// already has to exist further along: that span is not an expression, and the
// gate builder has the syntax tree to say so. One check, in the place that can
// make it.
//
// An earlier version of this carried a guard that re-applied the replacement and
// compared. It could not fail: the range is derived from the two files, so
// rebuilding from it reproduces the second one by construction. A check that
// cannot fail reads like a guarantee and is not one.
func Difference(original, mutated []byte) (Replacement, bool) {
	if bytes.Equal(original, mutated) {
		return Replacement{}, false
	}

	prefix := commonPrefix(original, mutated)
	suffix := commonSuffix(original[prefix:], mutated[prefix:])

	return Replacement{
		Start:    prefix,
		End:      len(original) - suffix,
		Original: string(original[prefix : len(original)-suffix]),
		Mutated:  string(mutated[prefix : len(mutated)-suffix]),
	}, true
}

func commonPrefix(a, b []byte) int {
	limit := min(len(a), len(b))

	for i := range limit {
		if a[i] != b[i] {
			return i
		}
	}

	return limit
}

func commonSuffix(a, b []byte) int {
	limit := min(len(a), len(b))

	for i := range limit {
		if a[len(a)-1-i] != b[len(b)-1-i] {
			return i
		}
	}

	return limit
}

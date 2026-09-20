package untested

// Reduce is imported by nothing and tested by nothing, so no command that names
// the covered package can compile it. It exists so the golden has a package
// whose mutants are unmeasurable rather than merely untested.
func Reduce(a, b int) int {
	return a - b
}

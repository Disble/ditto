package shared

// Sum has no tests of its own. It is compiled into the covered package's test
// binary, which is what makes its mutants killable by a command that never names
// this package.
func Sum(a, b int) int {
	return a + b
}

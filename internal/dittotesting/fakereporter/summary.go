package fakereporter

type Summary struct {
	Survived int
	Killed   int

	// NonViable is the mutants that never compiled, which are out of both the
	// numerator and the denominator. docs/metrics.md metric 1.
	NonViable int
}

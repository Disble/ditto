package fakereporter

type Summary struct {
	Survived int
	Killed   int

	// NonViable is the mutants that never compiled, which are out of both the
	// numerator and the denominator. docs/metrics.md metric 1.
	NonViable int

	// Unmeasured is the mutants the test command cannot execute, which leave the
	// score for the other reason a mutant cannot be judged and fail the run.
	// docs/reports/ditto-mutation-scope.md.
	Unmeasured int
}

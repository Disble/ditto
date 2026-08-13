package calc

// Covered is exercised by the suite from both sides, so its mutants die.
func Covered(a, b int) bool { return a > b }

// Partly is exercised from one side only, so some of its mutants live.
func Partly(a, b int) int {
	if a > b {
		return a - b
	}

	return b - a
}

// Uncovered is never called, so every mutant of it lives.
func Uncovered(a, b int) int { return a + b }

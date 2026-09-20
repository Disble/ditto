package covered

// Over is exercised from both sides, so its Comparison Invert mutant dies.
func Over(a, b int) bool {
	return a >= b
}

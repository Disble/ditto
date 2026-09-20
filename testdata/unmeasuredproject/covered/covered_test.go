package covered

import (
	"testing"

	"unmeasuredproject/shared"
)

func TestOver(t *testing.T) {
	if !Over(2, 1) || Over(1, 2) {
		t.Fatal("Over is wrong")
	}
}

// The fixture's third package reaches the score through this test and not
// through the command's package list: a mutant in shared is compiled into this
// package's test binary, so the command CAN kill it, and the run that follows
// has to judge it rather than report it as unmeasured.
func TestSum(t *testing.T) {
	if shared.Sum(2, 1) != 3 {
		t.Fatal("Sum is wrong")
	}
}

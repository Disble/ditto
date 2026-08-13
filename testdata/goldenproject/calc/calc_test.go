package calc

import "testing"

func TestCovered(t *testing.T) {
	if !Covered(2, 1) || Covered(1, 2) || Covered(1, 1) {
		t.Fatal("Covered is wrong")
	}
}

func TestPartly(t *testing.T) {
	if Partly(5, 3) != 2 {
		t.Fatal("Partly is wrong")
	}
}

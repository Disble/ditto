package scorecalculator

import (
	"github.com/Disble/ditto/internal/ditto"
)

func New() ditto.ScoreCalculator {
	return func(total, killed int) float32 {
		var score float32 = -1
		if total > 0 {
			score = float32(killed) / float32(total)
		}

		return score
	}
}

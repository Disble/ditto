package fakescorecalculator

import (
	"github.com/Disble/ditto/internal/ditto"
)

func Always(score float32) ditto.ScoreCalculator {
	return func(_total, _killed int) float32 { //nolint:revive
		return score
	}
}

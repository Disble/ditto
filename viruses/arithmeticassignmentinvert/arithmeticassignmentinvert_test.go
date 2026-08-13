package arithmeticassignmentinvert_test

import (
	"testing"

	"github.com/Disble/ditto/dittotesting"
	"github.com/Disble/ditto/viruses/arithmeticassignmentinvert"
)

func TestArithmeticAssignmentInvert(t *testing.T) {
	dittotesting.Run(t, dittotesting.NewScenarios(
		"Arithmetic Assignment Invert",
		arithmeticassignmentinvert.New(),
		dittotesting.Mutations{
			"no mutations": {"source.0.go", []string{}},
			"one mutation += to -=": {"source.1.go", []string{
				"source.1.mut.1.go",
			}},
			"one mutation -= to +=": {"source.2.go", []string{
				"source.2.mut.1.go",
			}},
			"one mutation *= to /=": {"source.3.go", []string{
				"source.3.mut.1.go",
			}},
			"one mutation /= to *=": {"source.4.go", []string{
				"source.4.mut.1.go",
			}},
			"one mutation %= to *=": {"source.5.go", []string{
				"source.5.mut.1.go",
			}},
			"many mutations": {"source.6.go", []string{
				"source.6.mut.1.go",
				"source.6.mut.2.go",
				"source.6.mut.3.go",
				"source.6.mut.4.go",
				"source.6.mut.5.go",
			}},
		},
	))
}

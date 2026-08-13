package floatincrement_test

import (
	"testing"

	"github.com/Disble/ditto/dittotesting"
	"github.com/Disble/ditto/viruses/floatincrement"
)

func TestFloatIncrement_32(t *testing.T) {
	dittotesting.Run(t, dittotesting.NewScenarios(
		"Float Increment",
		floatincrement.New(),
		dittotesting.Mutations{
			"no mutations": {"source32.0.go", []string{}},
			"one mutation": {"source32.1.go", []string{
				"source32.1.mut.1.go",
			}},
			"one mutation fine precision": {"source32.2.go", []string{
				"source32.2.mut.1.go",
			}},
			"two mutations": {"source32.3.go", []string{
				"source32.3.mut.1.go",
				"source32.3.mut.2.go",
			}},
			"many mutations": {"source32.4.go", []string{
				"source32.4.mut.1.go",
				"source32.4.mut.2.go",
				"source32.4.mut.3.go",
			}},
		},
	))
}

func TestFloatIncrement_64(t *testing.T) {
	dittotesting.Run(t, dittotesting.NewScenarios(
		"Float Increment",
		floatincrement.New(),
		dittotesting.Mutations{
			"no mutations": {"source64.0.go", []string{}},
			"one mutation": {"source64.1.go", []string{
				"source64.1.mut.1.go",
			}},
			"one mutation fine precision": {"source64.2.go", []string{
				"source64.2.mut.1.go",
			}},
			"two mutations": {"source64.3.go", []string{
				"source64.3.mut.1.go",
				"source64.3.mut.2.go",
			}},
			"many mutations": {"source64.4.go", []string{
				"source64.4.mut.1.go",
				"source64.4.mut.2.go",
				"source64.4.mut.3.go",
			}},
		},
	))
}

package arithmetic_test

import (
	"testing"

	"github.com/Disble/ditto/dittotesting"
	"github.com/Disble/ditto/viruses/arithmetic"
)

func TestArithmetic(t *testing.T) {
	dittotesting.Run(t, dittotesting.NewScenarios(
		"Arithmetic",
		arithmetic.New(),
		dittotesting.Mutations{
			"no mutations":                        {"source.0.go", []string{}},
			"no mutation of string concatenation": {"source.7.go", []string{}},
			"one mutation + to -": {"source.1.go", []string{
				"source.2.go",
			}},
			"one mutation - to +": {"source.2.go", []string{
				"source.1.go",
			}},
			"one mutation * to /": {"source.3.go", []string{
				"source.4.go",
			}},
			"one mutation / to *": {"source.4.go", []string{
				"source.3.go",
			}},
			"one mutation % to *": {"source.5.go", []string{
				"source.3.go",
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

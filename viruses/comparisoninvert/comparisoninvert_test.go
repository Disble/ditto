package comparisoninvert_test

import (
	"testing"

	"github.com/Disble/ditto/dittotesting"
	"github.com/Disble/ditto/viruses/comparisoninvert"
)

func TestComparisonInvert(t *testing.T) {
	dittotesting.Run(t, dittotesting.NewScenarios(
		"Comparison Invert",
		comparisoninvert.New(),
		dittotesting.Mutations{
			"no mutations": {"source.0.go", []string{}},
			"one mutation > to <=": {"source.1.go", []string{
				"source.2.go",
			}},
			"one mutation <= to >": {"source.2.go", []string{
				"source.1.go",
			}},
			"one mutation < to >=": {"source.3.go", []string{
				"source.4.go",
			}},
			"one mutation >= to <": {"source.4.go", []string{
				"source.3.go",
			}},
			"one mutation == to !=": {"source.5.go", []string{
				"source.6.go",
			}},
			"one mutation != to ==": {"source.6.go", []string{
				"source.5.go",
			}},
			"many mutations": {"source.7.go", []string{
				"source.7.mut.1.go",
				"source.7.mut.2.go",
				"source.7.mut.3.go",
				"source.7.mut.4.go",
				"source.7.mut.5.go",
				"source.7.mut.6.go",
			}},
		},
	))
}

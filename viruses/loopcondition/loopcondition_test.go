package loopcondition_test

import (
	"testing"

	"github.com/Disble/ditto/dittotesting"
	"github.com/Disble/ditto/viruses/loopcondition"
)

func TestLoopCondition(t *testing.T) {
	dittotesting.Run(t, dittotesting.NewScenarios(
		"Loop Condition",
		loopcondition.New(),
		dittotesting.Mutations{
			"no mutations": {"source.0.go", []string{}},
			"one mutation": {"source.1.go", []string{
				"source.1.mut.1.go",
			}},
			"two mutations": {"source.2.go", []string{
				"source.2.mut.1.go",
				"source.2.mut.2.go",
			}},
			"many mutations": {"source.3.go", []string{
				"source.3.mut.1.go",
				"source.3.mut.2.go",
				"source.3.mut.3.go",
			}},
		},
	))
}

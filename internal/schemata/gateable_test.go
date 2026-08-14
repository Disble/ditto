package schemata_test

import (
	"github.com/Disble/ditto/viruses"
	"github.com/Disble/ditto/viruses/comparison"
	"github.com/Disble/ditto/viruses/comparisoninvert"
	"github.com/Disble/ditto/viruses/comparisonreplace"
	"github.com/Disble/ditto/viruses/integerdecrement"
	"github.com/Disble/ditto/viruses/integerincrement"
)

// gateableViruses are the five whose mutations schemata can gate: three that
// rewrite a comparison, and two that change an integer literal. The other nine
// mutate statements or produce expressions that are not bools, and are refused
// by design.
func gateableViruses() []viruses.Virus {
	return []viruses.Virus{
		comparison.New(),
		comparisoninvert.New(),
		comparisonreplace.New(),
		integerincrement.New(),
		integerdecrement.New(),
	}
}

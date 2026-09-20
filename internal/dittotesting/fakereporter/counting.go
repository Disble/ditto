package fakereporter

import (
	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/result"
)

// Counting and Countless are the two answers a reporter can give when asked how
// much it scored, and every decorator has to be checked against both.
//
// They live here because three packages need the same pair: the release that
// reads the count, and each decorator that has to forward it. A decorator that
// drops a capability is refused by nothing — backlog entry 12 — so the check is
// written once and applied where each decorator lives.

// Counting reports fixed counts. The fields are named Scored and Outside because
// Total and Unmeasured are the methods the optional interfaces ask for, and a
// struct cannot have both a field and a method of the same name.
type Counting struct {
	Scored int

	// Outside is what Unmeasured reports: mutants outside what the test command
	// compiles.
	Outside int
}

func (Counting) AddDiagnostic(*ditto.Diagnostic) {}
func (Counting) Summarize() result.Result[any]   { return result.Err[any]("") }

// Total is the count this reporter was built with.
func (r Counting) Total() int { return r.Scored }

// Unmeasured is how much of the scope the command could not execute.
func (r Counting) Unmeasured() int { return r.Outside }

// Countless is every reporter that existed before the count did, and any a
// caller supplies.
type Countless struct{}

func (Countless) AddDiagnostic(*ditto.Diagnostic) {}
func (Countless) Summarize() result.Result[any]   { return result.Err[any]("") }

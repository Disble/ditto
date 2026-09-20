package verbosereporter

import (
	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/result"
)

type VerboseReporter struct {
	logger   ditto.Logger
	delegate ditto.Reporter
}

func New(logger ditto.Logger, delegate ditto.Reporter) *VerboseReporter {
	return &VerboseReporter{
		logger:   logger,
		delegate: delegate,
	}
}

func (r *VerboseReporter) AddDiagnostic(diagnostic *ditto.Diagnostic) {
	r.logger.Logf("registering diagnostic…")
	r.delegate.AddDiagnostic(diagnostic)
}

func (r *VerboseReporter) Summarize() result.Result[any] {
	r.logger.Logf("summarizing report…")

	return r.delegate.Summarize()
}

// Total forwards the count the reporter beneath scored.
//
// A decorator that drops a capability is refused by nothing -- backlog entry 12,
// and the reason this is written twice rather than assumed once. Without it the
// count is unreadable exactly when the stack is deepest, which is the gated run
// that found the problem in the first place.
func (r *VerboseReporter) Total() int {
	counted, ok := r.delegate.(interface{ Total() int })
	if !ok {
		return -1
	}

	return counted.Total()
}

// Unmeasured forwards how much of the scope the command could not execute, with
// the same reason and the same sentinel: an unreadable count is negative rather
// than zero, because zero is the answer that lets a run pass.
func (r *VerboseReporter) Unmeasured() int {
	counted, ok := r.delegate.(interface{ Unmeasured() int })
	if !ok {
		return -1
	}

	return counted.Unmeasured()
}

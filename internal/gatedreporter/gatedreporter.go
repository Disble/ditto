// Package gatedreporter says how much of a run came from one compilation.
//
// It exists because the gated path could stop engaging without anything saying
// so. `Gated` and `FellBack` are exact counters, and until this they were
// counters nobody printed: a run that gated every mutant and a run that gated
// none produced identical output, and the difference was found only by dropping
// a panic into `TestAll` by hand and watching it not fire. A number that is not
// printed is a number that cannot go wrong in front of anyone.
package gatedreporter

import (
	"strconv"

	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/result"
)

// Gates is the pair of counters a gated laboratory keeps.
type Gates interface {
	Gated() int
	FellBack() int
}

type GatedReporter struct {
	logger   ditto.Logger
	gates    Gates
	delegate ditto.Reporter
}

func New(logger ditto.Logger, gates Gates, delegate ditto.Reporter) *GatedReporter {
	return &GatedReporter{
		logger:   logger,
		gates:    gates,
		delegate: delegate,
	}
}

func (r *GatedReporter) AddDiagnostic(diagnostic *ditto.Diagnostic) {
	r.delegate.AddDiagnostic(diagnostic)
}

// Summarize reports the counts after the verdicts, because they describe how the
// run was executed rather than what it found. It never changes the verdict.
func (r *GatedReporter) Summarize() result.Result[any] {
	res := r.delegate.Summarize()

	gated, fellBack := r.gates.Gated(), r.gates.FellBack()

	total := gated + fellBack
	if total == 0 {
		return res
	}

	r.logger.Logf("┃ Gated: %s of %d mutants ran from one compilation; %d kept their own.",
		count(gated), total, fellBack)

	return res
}

// count spells zero out. `Gated: 0 of 7` scans as a formatting placeholder;
// `none of 7` does not, and this line exists to be noticed when it says zero.
func count(gated int) string {
	if gated == 0 {
		return "none"
	}

	return strconv.Itoa(gated)
}

// Total forwards the count the reporter beneath scored.
//
// A decorator that drops a capability is refused by nothing -- backlog entry 12,
// and the reason this is written twice rather than assumed once. Without it the
// count is unreadable exactly when the stack is deepest, which is the gated run
// that found the problem in the first place.
func (r *GatedReporter) Total() int {
	counted, ok := r.delegate.(interface{ Total() int })
	if !ok {
		return -1
	}

	return counted.Total()
}

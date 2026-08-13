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

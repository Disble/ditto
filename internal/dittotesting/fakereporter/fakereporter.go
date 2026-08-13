package fakereporter

import (
	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/result"
)

type FakeReporter struct {
	diagnostics []*ditto.Diagnostic
	summary     *Summary
}

func New() *FakeReporter {
	return &FakeReporter{
		diagnostics: []*ditto.Diagnostic{},
		summary:     nil,
	}
}

func (r *FakeReporter) AddDiagnostic(diagnostic *ditto.Diagnostic) {
	r.diagnostics = append(r.diagnostics, diagnostic)
}

func (r *FakeReporter) Summarize() result.Result[any] {
	survived := 0
	killed := 0

	for _, diagnostic := range r.diagnostics {
		if diagnostic.IsOk() {
			killed++
		} else {
			survived++
		}
	}

	r.summary = &Summary{
		Survived: survived,
		Killed:   killed,
	}

	if survived > 0 {
		return result.Err[any]("")
	}

	return result.Ok[any](nil)
}

func (r *FakeReporter) GetSummary() *Summary {
	return r.summary
}

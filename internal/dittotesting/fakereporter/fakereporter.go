package fakereporter

import (
	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/result"
	"github.com/Disble/ditto/internal/verdict"
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
	nonViable := 0

	for _, diagnostic := range r.diagnostics {
		// The same exclusion the shipped reporter applies. A double that scores
		// differently from the thing it stands in for measures the old rules,
		// and every test through it would keep agreeing with a version of ditto
		// that no longer exists. A mutant that never compiled is out of both
		// sides -- see internal/consolereporter and docs/metrics.md.
		if diagnostic.IsOk() && diagnostic.Reason() == verdict.BuildFailed {
			nonViable++

			continue
		}

		if diagnostic.IsOk() {
			killed++
		} else {
			survived++
		}
	}

	r.summary = &Summary{
		Survived:  survived,
		Killed:    killed,
		NonViable: nonViable,
	}

	if survived > 0 {
		return result.Err[any]("")
	}

	return result.Ok[any](nil)
}

func (r *FakeReporter) GetSummary() *Summary {
	return r.summary
}

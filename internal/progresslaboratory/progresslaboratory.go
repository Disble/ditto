// Package progresslaboratory says which mutant is running, as it runs.
//
// Backlog entry 22: a release used to print nothing between "Releasing Ditto…"
// and the report, so a run advancing normally and a run that was stuck produced
// identical bytes. A ten-minute wait was reported as a hang because there was
// nothing on screen to tell the two apart.
//
// It is a laboratory decorator rather than a line in Release, and the reason is
// an invariant the goldens hold: `ditto run` has to say exactly what
// ditto.Release says. Release stacks testingtlaboratory on top, which implements
// TestAll whether or not anything beneath it can batch — so a line printed by
// the caller lands in a different order on the two paths, while a line printed
// by a laboratory lands in the same one on both.
package progresslaboratory

import (
	"github.com/Disble/ditto/internal/color"
	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/result"
)

// ProgressLaboratory announces one mutant at a time.
type ProgressLaboratory struct {
	logger   ditto.Logger
	delegate ditto.Laboratory
}

// batching is the same decorator over a laboratory that can take a whole file at
// once. It exists as a second type because Go decides at compile time whether a
// value satisfies BatchLaboratory, and a decorator that always claimed to batch
// would make every laboratory beneath it look batchable — which is the silent
// failure testingtlaboratory.TestAll already documents from the other side.
type batching struct {
	*ProgressLaboratory
}

// New returns a decorator that batches exactly when what it wraps does.
func New(logger ditto.Logger, delegate ditto.Laboratory) ditto.Laboratory {
	announcing := &ProgressLaboratory{logger: logger, delegate: delegate}

	if _, ok := delegate.(ditto.BatchLaboratory); ok {
		return &batching{ProgressLaboratory: announcing}
	}

	return announcing
}

// Test names the mutant BEFORE handing it over. A line printed once a mutant has
// finished still cannot say which mutant a stall is inside, which is the whole
// of what this is for.
func (l *ProgressLaboratory) Test(
	repository ditto.Repository,
	file *gomutatedfile.GoMutatedFile,
) future.Future[result.Result[string]] {
	l.announce(file)

	return l.delegate.Test(repository, file)
}

// TestAll names every mutant of the file up front.
//
// A gated file is compiled once and its mutants selected at run time, so there
// is no per-mutant moment to interleave with. What can still be said is which
// mutants the compilation is about to answer for.
func (l *batching) TestAll(
	repository ditto.Repository,
	files []*gomutatedfile.GoMutatedFile,
) []future.Future[result.Result[string]] {
	for _, file := range files {
		l.announce(file)
	}

	return l.delegate.(ditto.BatchLaboratory).TestAll(repository, files) //nolint:forcetypeassert // New only builds this type when the delegate batches
}

func (l *ProgressLaboratory) announce(file *gomutatedfile.GoMutatedFile) {
	l.logger.Logf("%s   %s", color.Yellow("┃"), file.Label())
}

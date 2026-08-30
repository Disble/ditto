// Package confirminglaboratory re-runs a mutant that died by assertion, once.
//
// Backlog entry 27. verifyBaseline is a sync.Once: it catches a suite that is
// already red before anything is scored, and refuses rather than reporting. It
// cannot catch one that goes red at mutant 37. With no retry anywhere, a
// spurious failure during a mutant run classified as verdict.Assertion and
// became a false kill indistinguishable from a real one in the report — the
// same defect as the compile-failure kill, arriving from the other direction.
//
// It is opt-in, and the reason is cost: confirming doubles the price of every
// kill it applies to. What keeps that price arguable rather than obvious is that
// only ASSERTION kills can be flakes. A mutant that never built already leaves
// the score on both sides, and a deadline kill is one ditto fired its own clock
// for, so neither is re-run. The reported repository sees roughly one run in
// five go red on a cold suite; a repository that never flakes pays nothing for
// leaving this off, and nothing but time for turning it on.
package confirminglaboratory

import (
	"github.com/Disble/ditto/internal/color"
	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/result"
	"github.com/Disble/ditto/internal/verdict"
)

// ConfirmingLaboratory asks twice before it believes a kill.
type ConfirmingLaboratory struct {
	logger   ditto.Logger
	delegate ditto.Laboratory
}

// batching is the same decorator over a laboratory that takes a whole file at
// once. It is a second type for the reason progresslaboratory's is: Go decides
// at compile time whether a value satisfies BatchLaboratory, and a decorator
// that always claimed to would make every laboratory beneath it look batchable.
type batching struct {
	*ConfirmingLaboratory
}

// New returns a decorator that batches exactly when what it wraps does.
func New(logger ditto.Logger, delegate ditto.Laboratory) ditto.Laboratory {
	confirming := &ConfirmingLaboratory{logger: logger, delegate: delegate}

	if _, ok := delegate.(ditto.BatchLaboratory); ok {
		return &batching{ConfirmingLaboratory: confirming}
	}

	return confirming
}

func (l *ConfirmingLaboratory) Test(
	repository ditto.Repository,
	file *gomutatedfile.GoMutatedFile,
) future.Future[result.Result[string]] {
	return future.Resolved(l.confirm(repository, file, l.delegate.Test(repository, file).Await()))
}

// TestAll confirms each of a batch's kills one at a time.
//
// The re-run goes through Test rather than the batch, which for a gated file
// means the mutant is recompiled on its own. That is the slow path on purpose:
// the question being asked is whether this one verdict holds, and answering it
// from the same shared compilation that produced it would be asking the same
// run twice.
func (l *batching) TestAll(
	repository ditto.Repository,
	files []*gomutatedfile.GoMutatedFile,
) []future.Future[result.Result[string]] {
	batched := l.delegate.(ditto.BatchLaboratory).TestAll(repository, files) //nolint:forcetypeassert // New only builds this type when the delegate batches
	confirmed := make([]future.Future[result.Result[string]], len(files))

	for i, file := range files {
		confirmed[i] = future.Resolved(l.confirm(repository, file, batched[i].Await()))
	}

	return confirmed
}

// confirm re-runs one mutant when, and only when, a re-run could say anything.
//
// A survivor is never re-run: a flake manufactures failures, and nothing about a
// spurious failure makes a mutant the tests DID catch look like it escaped. So
// the asymmetry is deliberate — a mutant that survives either run survived.
func (l *ConfirmingLaboratory) confirm(
	repository ditto.Repository,
	file *gomutatedfile.GoMutatedFile,
	first result.Result[string],
) result.Result[string] {
	// Ok means the command failed, which for a mutant is a kill.
	if !first.IsOk() {
		return first
	}

	if verdict.ReasonOf(result.Output(first)) != verdict.Assertion {
		return first
	}

	second := l.delegate.Test(repository, file).Await()
	if second.IsOk() {
		return first
	}

	// Said out loud rather than folded silently into the survivor list, because
	// a verdict that changed on a re-run is evidence about the SUITE and not
	// only about the mutant.
	l.logger.Logf(
		"%s   %s died once and survived on confirmation; counted as a survivor, and the suite is flaky here.",
		color.Yellow("┃"),
		file.Label(),
	)

	return second
}

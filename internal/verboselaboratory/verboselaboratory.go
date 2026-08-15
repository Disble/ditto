package verboselaboratory

import (
	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/result"
)

type VerboseLaboratory struct {
	logger   ditto.Logger
	delegate ditto.Laboratory
}

func New(logger ditto.Logger, delegate ditto.Laboratory) *VerboseLaboratory {
	return &VerboseLaboratory{
		logger:   logger,
		delegate: delegate,
	}
}

func (l *VerboseLaboratory) Test(
	repository ditto.Repository,
	file *gomutatedfile.GoMutatedFile,
) future.Future[result.Result[string]] {
	l.logger.Logf("running laboratory tests for '%s'", file)
	fut := l.delegate.Test(repository, file)
	l.logger.Logf("laboratory result for '%s': %+v", file, fut.Await())

	return fut
}

// TestAll forwards a whole file's mutants when the laboratory below can take
// them, and keeps one mutant at a time otherwise.
//
// Without this the batch stops here, silently. `Release` wraps the stack in this
// decorator whenever `testing.Verbose` is set, and `Ditto.Release` reaches the
// gated laboratory only by asserting each layer to BatchLaboratory — so a layer
// that implements Test and not TestAll turns the gated path off and reports
// nothing, because the run still works, one compilation per mutant, exactly as
// if nothing had been wired at all.
//
// Measured on the golden fixture before this existed: 4 of 7 mutants gated
// without `-v`, none of 7 with it. `go test -v` is what CI runs.
// docs/experiments/forwarding-the-batch.md.
//
// The logging stays on the one-at-a-time path. A batch is one compilation for
// several mutants, so there is no per-mutant run below this to announce and
// nothing to await without serialising what was just batched.
func (l *VerboseLaboratory) TestAll(
	repository ditto.Repository,
	files []*gomutatedfile.GoMutatedFile,
) []future.Future[result.Result[string]] {
	if len(files) == 0 {
		return nil
	}

	batched, ok := l.delegate.(ditto.BatchLaboratory)
	if !ok {
		results := make([]future.Future[result.Result[string]], 0, len(files))
		for _, file := range files {
			results = append(results, l.Test(repository, file))
		}

		return results
	}

	l.logger.Logf("running laboratory tests for %d mutants of '%s' from one compilation", len(files), files[0])

	return batched.TestAll(repository, files)
}

package verbosetestrunner

import (
	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/laboratory"
	"github.com/Disble/ditto/internal/result"
)

type VerboseTestRunner struct {
	logger   ditto.Logger
	delegate laboratory.TestRunner
}

func New(logger ditto.Logger, delegate laboratory.TestRunner) *VerboseTestRunner {
	return &VerboseTestRunner{
		logger:   logger,
		delegate: delegate,
	}
}

func (r *VerboseTestRunner) Test(repository ditto.TemporaryRepository) result.Result[string] {
	r.logger.Logf("running tests on '%s'…", repository.Root())
	output := r.delegate.Test(repository)

	if output.IsOk() {
		r.logger.Logf("tests for '%s' failed; mutant killed", repository.Root())
	} else {
		r.logger.Logf("tests for '%s' passed; mutant survived", repository.Root())
	}

	return output
}

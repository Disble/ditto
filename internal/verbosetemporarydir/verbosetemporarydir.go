package verbosetemporarydir

import (
	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/laboratory"
)

type VerboseTemporaryDir struct {
	logger   ditto.Logger
	delegate laboratory.TemporaryDirectory
}

func New(logger ditto.Logger, delegate laboratory.TemporaryDirectory) *VerboseTemporaryDir {
	return &VerboseTemporaryDir{
		logger:   logger,
		delegate: delegate,
	}
}

func (d *VerboseTemporaryDir) New() string {
	dir := d.delegate.New()
	d.logger.Logf("setting up new temporary directory at '%s'", dir)

	return dir
}

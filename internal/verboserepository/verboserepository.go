package verboserepository

import (
	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/gosourcefile"
)

type VerboseRepository struct {
	logger   ditto.Logger
	delegate ditto.Repository
}

func New(logger ditto.Logger, delegate ditto.Repository) *VerboseRepository {
	return &VerboseRepository{
		logger:   logger,
		delegate: delegate,
	}
}

func (r *VerboseRepository) ListGoSourceFiles() []*gosourcefile.GoSourceFile {
	r.logger.Logf("listing go source files…")
	files := r.delegate.ListGoSourceFiles()
	r.logger.Logf("found %d source files: %s", len(files), files)

	return files
}

func (r *VerboseRepository) LinkAllToTemporaryRepository(temporaryPath string) ditto.TemporaryRepository {
	r.logger.Logf("linking all files to temporary path '%s'…", temporaryPath)
	repository := r.delegate.LinkAllToTemporaryRepository(temporaryPath)
	r.logger.Logf("linked all files to temporary path '%s'", temporaryPath)

	return NewVerboseTemporaryRepository(r.logger, repository)
}

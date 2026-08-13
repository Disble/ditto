package verboserepository

import "github.com/Disble/ditto/internal/ditto"

type VerboseTemporaryRepository struct {
	logger   ditto.Logger
	delegate ditto.TemporaryRepository
}

func NewVerboseTemporaryRepository(logger ditto.Logger, delegate ditto.TemporaryRepository) *VerboseTemporaryRepository {
	return &VerboseTemporaryRepository{
		logger:   logger,
		delegate: delegate,
	}
}

func (t *VerboseTemporaryRepository) Root() string {
	return t.delegate.Root()
}

func (t *VerboseTemporaryRepository) Overwrite(filePath string, data []byte) {
	t.logger.Logf("overwriting '%s'…", filePath)
	t.delegate.Overwrite(filePath, data)
}

func (t *VerboseTemporaryRepository) Remove() {
	t.logger.Logf("removing '%s'…", t.Root())
	t.delegate.Remove()
}

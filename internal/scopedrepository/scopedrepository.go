package scopedrepository

import (
	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/gosourcefile"
)

// ScopedRepository narrows a release to named byte ranges of named files.
//
// It exists for the case ditto is built for: a change is staged, only the lines
// it touched are worth measuring, and every mutant outside them costs a full
// run of the test command to confirm something the change did not ask about.
//
// The ranges are handed to each file rather than kept in one set, because byte
// offsets only mean anything against the file they were measured in. A scope
// that has lost track of which file a range came from makes every file answer
// to every range, and the work grows as the square of the number of files
// rather than in proportion to them.
type ScopedRepository struct {
	ranges   map[string][]gosourcefile.Range
	delegate ditto.Repository
}

func New(ranges map[string][]gosourcefile.Range, delegate ditto.Repository) *ScopedRepository {
	return &ScopedRepository{
		ranges:   ranges,
		delegate: delegate,
	}
}

// ListGoSourceFiles keeps only the files the scope names, each restricted to
// its own ranges.
//
// A file with no entry is dropped rather than mutated whole: it holds no
// changed line, so every mutant in it would measure code the change never
// touched.
func (r *ScopedRepository) ListGoSourceFiles() []*gosourcefile.GoSourceFile {
	scoped := []*gosourcefile.GoSourceFile{}

	for _, file := range r.delegate.ListGoSourceFiles() {
		ranges, named := r.ranges[file.String()]
		if !named {
			continue
		}

		scoped = append(scoped, file.Restrict(ranges))
	}

	return scoped
}

func (r *ScopedRepository) LinkAllToTemporaryRepository(temporaryPath string) ditto.TemporaryRepository {
	return r.delegate.LinkAllToTemporaryRepository(temporaryPath)
}

package ignoredrepository

import (
	"regexp"

	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/gosourcefile"
)

type FilteredRepository struct {
	patterns []*regexp.Regexp
	delegate ditto.Repository
}

func New(patterns []*regexp.Regexp, delegate ditto.Repository) *FilteredRepository {
	return &FilteredRepository{
		patterns: patterns,
		delegate: delegate,
	}
}

func (r *FilteredRepository) ListGoSourceFiles() []*gosourcefile.GoSourceFile {
	filtered := []*gosourcefile.GoSourceFile{}

FILE_LOOP:
	for _, file := range r.delegate.ListGoSourceFiles() {
		for _, pattern := range r.patterns {
			if pattern.MatchString(file.String()) {
				continue FILE_LOOP
			}
		}

		filtered = append(filtered, file)
	}

	return filtered
}

func (r *FilteredRepository) LinkAllToTemporaryRepository(temporaryPath string) ditto.TemporaryRepository {
	return r.delegate.LinkAllToTemporaryRepository(temporaryPath)
}

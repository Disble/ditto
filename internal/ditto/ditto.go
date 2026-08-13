package ditto

import (
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/goinfectedfile"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/gosourcefile"
	"github.com/Disble/ditto/internal/result"
	"github.com/Disble/ditto/viruses"
)

type Logger interface {
	Logf(message string, args ...any)
}

type Repository interface {
	ListGoSourceFiles() []*gosourcefile.GoSourceFile
	LinkAllToTemporaryRepository(temporaryPath string) TemporaryRepository
}

type TemporaryRepository interface {
	Root() string
	Overwrite(filePath string, data []byte)
	Remove()
}

type Laboratory interface {
	Test(repository Repository, file *gomutatedfile.GoMutatedFile) future.Future[result.Result[string]]
}

type ScoreCalculator func(total, killed int) float32

type Diagnostic struct {
	res  future.Future[result.Result[string]]
	file *gomutatedfile.GoMutatedFile
}

func (d *Diagnostic) IsOk() bool {
	return d.res.Await().IsOk()
}

func (d *Diagnostic) Diff(differ gomutatedfile.Differ) string {
	return d.file.Diff(differ)
}

func (d *Diagnostic) Label() string {
	return d.file.Label()
}

func NewDiagnostic(res future.Future[result.Result[string]], file *gomutatedfile.GoMutatedFile) *Diagnostic {
	return &Diagnostic{
		res:  res,
		file: file,
	}
}

type Reporter interface {
	AddDiagnostic(diagnostic *Diagnostic)
	Summarize() result.Result[any]
}

type Ditto struct {
	repository Repository
	laboratory Laboratory
	reporter   Reporter
}

func New(repository Repository, laboratory Laboratory, reporter Reporter) *Ditto {
	return &Ditto{
		repository: repository,
		laboratory: laboratory,
		reporter:   reporter,
	}
}

func (o *Ditto) Release(viri ...viruses.Virus) {
	sources := o.repository.ListGoSourceFiles()

	var incubated []*goinfectedfile.GoInfectedFile

	for _, source := range sources {
		incubated = append(incubated, source.Incubate(viri...)...)
	}

	for _, infectedFile := range incubated {
		mutatedFile := infectedFile.Mutate()
		res := o.laboratory.Test(o.repository, mutatedFile)
		diagnostic := NewDiagnostic(res, mutatedFile)
		o.reporter.AddDiagnostic(diagnostic)
	}
}

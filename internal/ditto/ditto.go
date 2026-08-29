package ditto

import (
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/goinfectedfile"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/gosourcefile"
	"github.com/Disble/ditto/internal/result"
	"github.com/Disble/ditto/internal/verdict"
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

// BatchLaboratory is handed every mutant of one file together.
//
// One compilation can only serve several mutants if the compiler is given all of
// them at once, and the gated path needs exactly that. It is a separate
// interface rather than a change to Laboratory so that every laboratory that
// exists today goes on being asked one mutant at a time, unchanged.
//
// The batch cannot be assembled lower down. Test returns a future, and a
// laboratory could in principle buffer and resolve later — but testingtlaboratory
// awaits inside each subtest, immediately, so a laboratory waiting for mutants
// that have not been submitted yet would wait forever.
type BatchLaboratory interface {
	TestAll(repository Repository, files []*gomutatedfile.GoMutatedFile) []future.Future[result.Result[string]]
}

type ScoreCalculator func(total, killed int) float32

type Diagnostic struct {
	res  future.Future[result.Result[string]]
	file *gomutatedfile.GoMutatedFile
}

func (d *Diagnostic) IsOk() bool {
	return d.res.Await().IsOk()
}

// Reason is why this mutant died, and Unknown when ditto was not told.
//
// Ditto recognises a killed mutant by a non-zero exit, which a mutant that never
// compiled also produces. Measured on internal/schemata/instrument.go: 78
// mutants, 50 reported killed, 10 of which did not compile and 1 of which hung
// until its timeout -- 22% of the kills credited to tests that never ran. See
// docs/metrics.md, metric 2.
//
// A survivor has no death to explain, so it answers Unknown too. The reason is
// about a kill.
func (d *Diagnostic) Reason() verdict.Reason {
	res := d.res.Await()
	if !res.IsOk() {
		return verdict.Unknown
	}

	return verdict.ReasonOf(result.Output(res))
}

func (d *Diagnostic) Diff(differ gomutatedfile.Differ) string {
	return d.file.Diff(differ)
}

func (d *Diagnostic) Label() string {
	return d.file.Label()
}

// Address and Change are what a survivor is reported as before any diff is
// rendered: where it is, and what it wrote there.
func (d *Diagnostic) Address() string { return d.file.Address() }

// Virus names the mutation operator behind this diagnostic, which is the unit a
// non-viable mutant is fixed in. docs/metrics.md metric 1.
func (d *Diagnostic) Virus() string  { return d.file.Virus() }
func (d *Diagnostic) Change() string { return d.file.Change() }

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

// Release mutates every source file and reports what each mutant did.
//
// Mutants are kept together by the file they came from, because a laboratory
// that compiles once for a whole file has to receive the file's mutants at once.
// The order they are reported in is unchanged: sources are walked in order, and
// each file's mutants in the order its viruses produced them.
func (o *Ditto) Release(viri ...viruses.Virus) {
	for _, source := range o.repository.ListGoSourceFiles() {
		mutants := mutate(source.Incubate(viri...))
		if len(mutants) == 0 {
			continue
		}

		for i, res := range o.test(mutants) {
			o.reporter.AddDiagnostic(NewDiagnostic(res, mutants[i]))
		}
	}
}

func mutate(infected []*goinfectedfile.GoInfectedFile) []*gomutatedfile.GoMutatedFile {
	mutants := make([]*gomutatedfile.GoMutatedFile, 0, len(infected))

	for _, one := range infected {
		mutants = append(mutants, one.Mutate())
	}

	return mutants
}

// test asks for the whole batch when the laboratory can take one, and falls back
// to one mutant at a time otherwise — which is what every laboratory shipped so
// far does, so nothing that worked stops working.
func (o *Ditto) test(mutants []*gomutatedfile.GoMutatedFile) []future.Future[result.Result[string]] {
	if batched, ok := o.laboratory.(BatchLaboratory); ok {
		return batched.TestAll(o.repository, mutants)
	}

	results := make([]future.Future[result.Result[string]], 0, len(mutants))
	for _, mutant := range mutants {
		results = append(results, o.laboratory.Test(o.repository, mutant))
	}

	return results
}

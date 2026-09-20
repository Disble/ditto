package ditto

import (
	"github.com/Disble/ditto/internal/color"
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

// CommandScope answers whether the configured test command can execute the
// package that owns a source path.
//
// It is a separate interface rather than part of Laboratory, for the reason
// BatchLaboratory is: a laboratory that cannot answer is still a laboratory, and
// the release then runs exactly what it ran before. An implementation is allowed
// to answer true because it does not know -- see laboratory.Executes -- and the
// release treats that as "everything is executed", which is what ditto assumed
// until it could ask.
type CommandScope interface {
	Executes(repository Repository, relativePath string) bool
}

type Diagnostic struct {
	res  future.Future[result.Result[string]]
	file *gomutatedfile.GoMutatedFile

	// unmeasured is a mutant ditto chose not to run at all, because the test
	// command cannot compile the package that owns it. It is not a verdict about
	// the mutant, which is why it does not travel as one: the score leaves it out
	// on both sides and the report names it. See NewUnmeasuredDiagnostic.
	unmeasured bool
}

func (d *Diagnostic) IsOk() bool {
	return d.res.Await().IsOk()
}

// Unmeasured reports a mutant that was never run because the test command does
// not compile the package that owns it.
//
// Read it BEFORE IsOk. An unmeasured diagnostic is shaped like a survivor -- that
// is what it would have been, before ditto could tell the difference -- so a
// consumer that ignores this reads exactly what it read before this existed,
// which is the graceful half. The shipped reporter does not ignore it.
func (d *Diagnostic) Unmeasured() bool { return d.unmeasured }

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

// Path is the repository-relative source file this mutant came from. The report
// groups the mutants it could not measure by the directory this names.
func (d *Diagnostic) Path() string { return d.file.Path() }

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

// NewUnmeasuredDiagnostic is a mutant ditto did not run, because the configured
// test command does not compile the package that owns it.
//
// Nothing the command runs can kill it, so its survival is not evidence about
// anybody's tests: the score leaves it out of the numerator and the denominator,
// exactly as it leaves out a mutant that never compiled, and the report names it
// instead. Measured on the report that asked for it: 43 mutants, 20 survivors,
// 17 of them in one package the command never builds, printed as a score of 0.53
// against a bar of 0.80. docs/reports/ditto-mutation-scope.md.
//
// It carries the result ditto would have reported before it could tell the
// difference -- a survivor -- so a consumer that does not know about Unmeasured
// reads what it read before, and none of them meets a nil.
func NewUnmeasuredDiagnostic(file *gomutatedfile.GoMutatedFile) *Diagnostic {
	return &Diagnostic{
		res:        future.Resolved(result.Err[string]("")),
		file:       file,
		unmeasured: true,
	}
}

type Reporter interface {
	AddDiagnostic(diagnostic *Diagnostic)
	Summarize() result.Result[any]
}

type Ditto struct {
	logger     Logger
	repository Repository
	laboratory Laboratory
	reporter   Reporter

	// scope is how the release asks whether the configured test command can
	// execute a package at all. Nil when nothing can answer, and the release then
	// runs every mutant of the scope, which is what it did before this existed.
	scope CommandScope
}

func New(logger Logger, repository Repository, laboratory Laboratory, reporter Reporter, scopes ...CommandScope) *Ditto {
	var scope CommandScope

	if len(scopes) > 0 {
		scope = scopes[0]
	}

	return &Ditto{
		logger:     logger,
		repository: repository,
		laboratory: laboratory,
		reporter:   reporter,
		scope:      scope,
	}
}

// Release mutates every source file and reports what each mutant did.
//
// Mutants are kept together by the file they came from, because a laboratory
// that compiles once for a whole file has to receive the file's mutants at once.
// The order they are reported in is unchanged: sources are walked in order, and
// each file's mutants in the order its viruses produced them.
//
// A file whose package the test command cannot execute is not run at all: its
// mutants are recorded as unmeasured, and the report says so. Running them would
// buy a survivor nobody can kill with a full run of the suite, and the number
// that came back would be a mixture of two different questions.
func (o *Ditto) Release(viri ...viruses.Virus) {
	for _, source := range o.repository.ListGoSourceFiles() {
		mutants := mutate(source.Incubate(viri...))
		if len(mutants) == 0 {
			continue
		}

		if !o.executes(mutants[0].Path()) {
			for _, mutant := range mutants {
				o.reporter.AddDiagnostic(NewUnmeasuredDiagnostic(mutant))
			}

			continue
		}

		// Said before the file's mutants run, not after, because this is the
		// number a reader needs in order to know what the silence that follows
		// is going to cost. It is also the multiplier for the baseline duration
		// the laboratory prints -- the two factors of the bill, adjacent, which
		// is what nobody had when a ten-minute wait was reported as a hang.
		o.logger.Logf("%s %s — %d mutants", color.Yellow("┃"), mutants[0].Path(), len(mutants))

		for i, res := range o.test(mutants) {
			o.reporter.AddDiagnostic(NewDiagnostic(res, mutants[i]))
		}
	}
}

// executes reports whether the configured test command can execute the package
// that owns a source path.
//
// Everything is executed when nothing can answer the question, and that is not a
// fallback: a command that is not `go test -json` has no readable package scope,
// and ditto has no business failing a run over a question it cannot read.
func (o *Ditto) executes(relativePath string) bool {
	if o.scope == nil {
		return true
	}

	return o.scope.Executes(o.repository, relativePath)
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

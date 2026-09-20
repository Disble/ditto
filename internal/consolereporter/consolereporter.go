package consolereporter

import (
	"path"
	"sort"
	"strings"

	"github.com/Disble/ditto/internal/color"
	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/result"
	"github.com/Disble/ditto/internal/verdict"
)

type ConsoleReporter struct {
	logger           ditto.Logger
	differ           gomutatedfile.Differ
	calculator       ditto.ScoreCalculator
	minimumThreshold float32
	diagnostics      []*ditto.Diagnostic

	// total is what the last Summarize scored, after the mutants that never
	// compiled were removed from both sides. Zero and "below threshold" are
	// different answers -- a scope with nothing mutable in it is not a suite
	// that failed -- and the score alone cannot tell them apart, because the
	// calculator reports -1 for both an empty run and nothing else.
	total int

	// unmeasured is how many mutants left the score for the other reason a
	// mutant cannot be judged: the test command does not compile the package
	// that owns it. See logUnmeasured.
	unmeasured int
}

func New(
	logger ditto.Logger,
	differ gomutatedfile.Differ,
	calculator ditto.ScoreCalculator,
	minimumThreshold float32,
) *ConsoleReporter {
	return &ConsoleReporter{
		logger:           logger,
		differ:           differ,
		calculator:       calculator,
		minimumThreshold: minimumThreshold,
		diagnostics:      []*ditto.Diagnostic{},
	}
}

// Total is how many mutants the last Summarize scored. It is read through an
// optional interface, the way the temporary directory's RemoveAll and the gated
// laboratory's counters are, so no reporter is forced to answer.
func (r *ConsoleReporter) Total() int { return r.total }

// Unmeasured is how many mutants the last Summarize left out because the test
// command does not compile the package that owns them.
//
// It is a separate count from Total for the reason the exclusion exists: those
// mutants are not a smaller denominator to read a score against, they are the
// part of the scope this run could not measure. The run fails on a non-zero count
// whatever the score says, and the exit code is how a gate hears it.
func (r *ConsoleReporter) Unmeasured() int { return r.unmeasured }

func (r *ConsoleReporter) AddDiagnostic(diagnostic *ditto.Diagnostic) {
	r.diagnostics = append(r.diagnostics, diagnostic)
}

func (r *ConsoleReporter) Summarize() result.Result[any] {
	counted := r.count()

	total := counted.killed + counted.survived
	r.total = total
	r.unmeasured = counted.unmeasured

	// Addresses first, diffs after. Survivors are the only part of this report
	// anybody acts on, and printing them after the diffs turned the report into
	// an index into the log: the author scrolled back through every rendered
	// diff to recover where each survivor landed.
	r.logAddresses(counted.survivors)

	for _, diagnostic := range counted.survivors {
		r.logDiff(diagnostic)
	}

	res := result.Ok[any](nil)
	scoreColor := color.BoldGreen
	scoreIcon := "✓"
	score := r.calculator(total, counted.killed)

	// A run that could not measure part of its scope did not pass, whatever the
	// score over the rest says. The score is a measurement of a smaller
	// population than the scope, so it cannot be the green one: the marker is
	// about the run, and the line below says what left it out.
	if score < r.minimumThreshold || counted.unmeasured > 0 {
		res = result.Err[any]("")
		scoreColor = color.BoldRed
		scoreIcon = "⨯"
	}

	r.logger.Logf("┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓")
	r.logger.Logf("┃ • "+color.Bold("Total")+": %8d                    ┃", total)
	r.logger.Logf("┃ • "+color.Bold("Killed")+": %7d                    ┃", counted.killed)
	r.logger.Logf("┃ • "+color.Bold("Survived")+": %5d                    ┃", counted.survived)

	// The composition of the score, beside the score: a reader who sees 26 has
	// to see that the scope held 43. Printed only when there is something to say,
	// because a line on every run is a line people stop reading.
	if counted.unmeasured > 0 {
		r.logger.Logf("┃ • "+color.Bold("Unmeasured")+": %8d               ┃", counted.unmeasured)
	}

	r.logger.Logf("┠┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┨")
	r.logger.Logf("┃ " + scoreColor("%s Score: %8.2f (minimum: %.2f)", scoreIcon, score, r.minimumThreshold) + "    ┃")
	r.logger.Logf("┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛")

	r.logUnmeasured(counted.unmeasured, len(r.diagnostics), counted.byDirectory)
	r.logUnearned(counted.nonViable, len(r.diagnostics), counted.byVirus)

	return res
}

// tally is what one pass over the diagnostics found: how many mutants of each
// kind, and the breakdowns the report names them by.
type tally struct {
	killed     int
	survived   int
	unmeasured int
	nonViable  int

	survivors   []*ditto.Diagnostic
	byVirus     map[string]int
	byDirectory map[string]int
}

// count is one pass over the diagnostics: what each mutant was, and why.
//
// One pass, because the report asks three questions about the same list and the
// exclusions have to be applied identically to all three. The order of the cases
// is the rule:
//
//   - Unmeasured comes first. It is shaped like a survivor and is not one: it
//     never ran, so counting it either way would answer a question this run did
//     not ask.
//   - A mutant that never compiled leaves the numerator AND the denominator. The
//     kill predicate is undefined for a program that does not exist: Zhu, Hall &
//     May, ACM Computing Surveys 29(4) 1997, Def 3.1 -- S = D / (M − E) -- and
//     gremlins, cargo-mutants, Stryker and go-mutesting all exclude it. It is
//     counted and named instead, because it is a defect of the GENERATOR with a
//     benchmark to answer to.
//   - Everything else is a kill or a survivor, which is what the score is.
func (r *ConsoleReporter) count() tally {
	counted := tally{
		byVirus:     map[string]int{},
		byDirectory: map[string]int{},
	}

	for _, diagnostic := range r.diagnostics {
		switch {
		case diagnostic.Unmeasured():
			counted.unmeasured++
			counted.byDirectory[path.Dir(diagnostic.Path())]++
		case diagnostic.IsOk() && diagnostic.Reason() == verdict.BuildFailed:
			counted.nonViable++
			counted.byVirus[diagnostic.Virus()]++
		case diagnostic.IsOk():
			counted.killed++
		default:
			counted.survived++
			counted.survivors = append(counted.survivors, diagnostic)
		}
	}

	return counted
}

// logUnmeasured says how much of the scope the score above does not cover, and
// names the packages that left it.
//
// This is the one number the report cannot compute for itself. Ditto scopes
// mutants to a change and judges them with a single configured command, and when
// that command does not compile a mutant's package the mutant is not badly
// tested, it is unmeasured: nothing the command runs can kill it, so it survives,
// and a score counting it mixes "your tests missed this" with "your command
// cannot see this". Measured on the report that asked for it: 43 mutants, 23
// killed, 20 survived, and 17 of those survivors in one package the command never
// builds -- a run whose ceiling was 0.605, printed as 0.53 against a bar of 0.80.
// docs/reports/ditto-mutation-scope.md.
//
// It says nothing when there is nothing to say, for the reason logUnearned does.
// The packages are named because they are the unit of the fix: one line per
// package, most mutants first, and no line at all for a package whose every
// mutant was measured.
func (r *ConsoleReporter) logUnmeasured(unmeasured, generated int, byDirectory map[string]int) {
	if unmeasured == 0 {
		return
	}

	r.logger.Logf("┃ %s", color.BoldRed(
		"%d of the %d mutants in this scope are never compiled by this test command, so",
		unmeasured, generated))
	r.logger.Logf("┃ %s", color.BoldRed("nothing it runs can kill them and they are out of the score entirely:"))

	directories := make([]string, 0, len(byDirectory))
	for directory := range byDirectory {
		directories = append(directories, directory)
	}

	sort.Slice(directories, func(i, j int) bool {
		if byDirectory[directories[i]] == byDirectory[directories[j]] {
			return directories[i] < directories[j]
		}

		return byDirectory[directories[i]] > byDirectory[directories[j]]
	})

	for _, directory := range directories {
		r.logger.Logf("┃   %s (%d)", directory, byDirectory[directory])
	}

	r.logger.Logf("┃ %s", color.Bold(
		"Name every package the scope mutates in --test-command, or narrow the scope."))
}

// logAddresses prints one line per survivor: where it is, what hit it, and what
// it wrote. It is the first thing on screen, and it says nothing at all when
// nothing survived.
func (r *ConsoleReporter) logAddresses(survivors []*ditto.Diagnostic) {
	if len(survivors) == 0 {
		return
	}

	r.logger.Logf("┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╍┅")
	r.logger.Logf("┃ 🧬 " + color.BoldRed("Survivors"))
	r.logger.Logf("┠┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄")

	for _, survivor := range survivors {
		line := survivor.Label()
		if change := survivor.Change(); change != "" {
			line += " (" + change + ")"
		}

		r.logger.Logf("┃ %s", line)
	}

	r.logger.Logf("┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╍┅")
}

func (r *ConsoleReporter) logDiff(diagnostic *ditto.Diagnostic) {
	r.logger.Logf("┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╍┅")
	r.logger.Logf("┃ 🧬 "+color.BoldRed("Mutant survived:")+" %s", diagnostic.Label())
	r.logger.Logf("┠┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄")

	diff := []string{}
	for line := range strings.SplitSeq(diagnostic.Diff(r.differ), "\n") {
		diff = append(diff, "┃ "+line)
	}

	r.logger.Logf(strings.Join(diff, "\n"))
	r.logger.Logf("┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╍┅")
}

// logUnearned says how many of the kills above nobody's test earned.
//
// A mutant that never compiled makes the test command exit non-zero, which is
// how ditto recognises a kill, so it is scored as caught by a suite that never
// ran it. Measured on one file: 10 of 50 kills, plus a hung mutant, 22% in all.
// The number people act on carried them silently.
//
// It says nothing when there are none, because a line printed on every run is a
// line people stop reading. And it can only see what the test command told it:
// the reason is read from `go test -json`, and a command that emits something
// else yields no reason rather than a guess. See docs/metrics.md, metric 2.
func (r *ConsoleReporter) logUnearned(unearned, generated int, byVirus map[string]int) {
	if unearned == 0 {
		return
	}

	r.logger.Logf("┃ %s", color.BoldRed(
		"%d of the %d mutants generated never compiled, and are out of the score entirely.", unearned, generated))

	// The rate has an external benchmark -- Major 1.8%, PIT 0% -- and a rate
	// alone names no work. The virus is what somebody fixes.
	viruses := make([]string, 0, len(byVirus))
	for virus := range byVirus {
		viruses = append(viruses, virus)
	}

	sort.Slice(viruses, func(i, j int) bool { return byVirus[viruses[i]] > byVirus[viruses[j]] })

	for _, virus := range viruses {
		r.logger.Logf("┃   %d from %s", byVirus[virus], virus)
	}
}

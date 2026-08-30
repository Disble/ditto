package consolereporter

import (
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

func (r *ConsoleReporter) AddDiagnostic(diagnostic *ditto.Diagnostic) {
	r.diagnostics = append(r.diagnostics, diagnostic)
}

func (r *ConsoleReporter) Summarize() result.Result[any] {
	var killed, survived int

	survivors := []*ditto.Diagnostic{}

	for _, diagnostic := range r.diagnostics {
		// A mutant that never compiled leaves the numerator AND the denominator.
		// The kill predicate is undefined for a program that does not exist:
		// Zhu, Hall & May, ACM Computing Surveys 29(4) 1997, Def 3.1 --
		// S = D / (M − E) -- and gremlins, cargo-mutants, Stryker and
		// go-mutesting all exclude it. It is counted and named below instead,
		// because it is a defect of the GENERATOR with a benchmark to answer to.
		if diagnostic.IsOk() && diagnostic.Reason() == verdict.BuildFailed {
			continue
		}

		if diagnostic.IsOk() {
			killed++
		} else {
			survived++

			survivors = append(survivors, diagnostic)
		}
	}

	total := killed + survived
	r.total = total

	// Addresses first, diffs after. Survivors are the only part of this report
	// anybody acts on, and printing them after the diffs turned the report into
	// an index into the log: the author scrolled back through every rendered
	// diff to recover where each survivor landed.
	r.logAddresses(survivors)

	for _, diagnostic := range survivors {
		r.logDiff(diagnostic)
	}

	unearned := 0
	byVirus := map[string]int{}

	for _, diagnostic := range r.diagnostics {
		if diagnostic.IsOk() && diagnostic.Reason() == verdict.BuildFailed {
			unearned++
			byVirus[diagnostic.Virus()]++
		}
	}

	res := result.Ok[any](nil)
	scoreColor := color.BoldGreen
	scoreIcon := "✓"
	score := r.calculator(total, killed)

	if score < r.minimumThreshold {
		res = result.Err[any]("")
		scoreColor = color.BoldRed
		scoreIcon = "⨯"
	}

	r.logger.Logf("┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓")
	r.logger.Logf("┃ • "+color.Bold("Total")+": %8d                    ┃", total)
	r.logger.Logf("┃ • "+color.Bold("Killed")+": %7d                    ┃", killed)
	r.logger.Logf("┃ • "+color.Bold("Survived")+": %5d                    ┃", survived)
	r.logger.Logf("┠┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┨")
	r.logger.Logf("┃ " + scoreColor("%s Score: %8.2f (minimum: %.2f)", scoreIcon, score, r.minimumThreshold) + "    ┃")
	r.logger.Logf("┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛")

	r.logUnearned(unearned, len(r.diagnostics), byVirus)

	return res
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

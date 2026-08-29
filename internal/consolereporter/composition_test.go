package consolereporter_test

import (
	"strings"
	"testing"

	"github.com/Disble/ditto/internal/consolereporter"
	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/dittotesting/fakelogger"
	"github.com/Disble/ditto/internal/dittotesting/stubdiffer"
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/result"
	"github.com/Disble/ditto/internal/scorecalculator"
)

const buildFailedStream = `{"Action":"build-fail"}
{"Action":"fail","FailedBuild":"example.com/x/pkg [example.com/x/pkg.test]"}
`

func killWith(output string) *ditto.Diagnostic {
	return ditto.NewDiagnostic(
		future.Resolved(result.Ok(output)),
		gomutatedfile.New("Fake", "pkg/a.go",
			[]byte("package pkg\n\nvar n = 1\n"),
			[]byte("package pkg\n\nvar n = 2\n")),
	)
}

// A kill nobody's test earned has to appear in the report. Silence here is how
// 22% of the kills on one measured file passed for test coverage.
func TestTheReportNamesKillsNoTestEarned(t *testing.T) {
	t.Parallel()

	logger := fakelogger.New()
	reporter := consolereporter.New(logger, stubdiffer.New(""), scorecalculator.New(), 0.0)

	reporter.AddDiagnostic(killWith(buildFailedStream))
	reporter.AddDiagnostic(killWith(buildFailedStream))
	reporter.AddDiagnostic(killWith("--- FAIL: TestAdd\n"))
	reporter.Summarize()

	printed := strings.Join(logger.LoggedLines(), "\n")
	if !strings.Contains(printed, "never compiled") {
		t.Fatalf("the report says nothing about kills no test earned:\n%s", printed)
	}

	if !strings.Contains(printed, "2") {
		t.Fatalf("the report does not carry the count:\n%s", printed)
	}
}

// And it stays quiet when there is nothing to say, because a line printed on
// every run is a line people stop reading.
func TestTheReportIsQuietWhenEveryKillWasEarned(t *testing.T) {
	t.Parallel()

	logger := fakelogger.New()
	reporter := consolereporter.New(logger, stubdiffer.New(""), scorecalculator.New(), 0.0)

	reporter.AddDiagnostic(killWith(`{"Action":"fail","Test":"TestAdd"}`))
	reporter.Summarize()

	if printed := strings.Join(logger.LoggedLines(), "\n"); strings.Contains(printed, "never compiled") {
		t.Fatalf("the report invented a problem:\n%s", printed)
	}
}

// Metric 1 is a rate with an external benchmark -- Major 1.8%, PIT 0% -- and a
// rate alone names no work. The virus that produced the mutant is what somebody
// fixes, so the report has to carry it.
func TestTheReportNamesTheVirusThatProducedThem(t *testing.T) {
	t.Parallel()

	logger := fakelogger.New()
	reporter := consolereporter.New(logger, stubdiffer.New(""), scorecalculator.New(), 0.0)

	reporter.AddDiagnostic(killFrom("Integer Decrement", buildFailedStream))
	reporter.AddDiagnostic(killFrom("Integer Decrement", buildFailedStream))
	reporter.AddDiagnostic(killFrom("Range Break", buildFailedStream))
	reporter.Summarize()

	printed := strings.Join(logger.LoggedLines(), "\n")
	for _, want := range []string{"Integer Decrement", "Range Break"} {
		if !strings.Contains(printed, want) {
			t.Fatalf("the report does not name %q, so nobody knows what to fix:\n%s", want, printed)
		}
	}
}

func killFrom(virus, output string) *ditto.Diagnostic {
	return ditto.NewDiagnostic(
		future.Resolved(result.Ok(output)),
		gomutatedfile.New(virus, "pkg/a.go",
			[]byte("package pkg\n\nvar n = 1\n"),
			[]byte("package pkg\n\nvar n = 2\n")),
	)
}

// A mutant that never compiled leaves the numerator AND the denominator.
//
// Not a modelling choice: the kill predicate is undefined for a program that
// does not exist. Zhu, Hall & May, ACM Computing Surveys 29(4) 1997, Def 3.1:
// S = D / (M - E). Stryker states the same formula with the classes named --
// "Compile error: The mutant caused a compile error... It is not represented in
// your mutation score" -- and gremlins, cargo-mutants, Stryker.NET and
// go-mutesting all exclude it. Naming it in the report was not enough: labelling
// is not excluding, and the score kept carrying it.
func TestANonViableMutantLeavesTheScoreEntirely(t *testing.T) {
	t.Parallel()

	logger := fakelogger.New()
	reporter := consolereporter.New(logger, stubdiffer.New(""), scorecalculator.New(), 0.0)

	reporter.AddDiagnostic(killWith(`{"Action":"fail","Test":"TestAdd"}`))
	reporter.AddDiagnostic(killWith(buildFailedStream))
	reporter.AddDiagnostic(killWith(buildFailedStream))
	reporter.Summarize()

	printed := strings.Join(logger.LoggedLines(), "\n")

	// One earned kill of one viable mutant. Counting the two that never built
	// gives 3 of 3 and a perfect score for a suite that caught one thing.
	if !strings.Contains(printed, "Score:     1.00") {
		t.Fatalf("the score still carries the mutants that never built:\n%s", printed)
	}

	if !strings.Contains(printed, "Total:        1") {
		t.Fatalf("the denominator still carries them:\n%s", printed)
	}
}

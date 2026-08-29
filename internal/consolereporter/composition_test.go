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

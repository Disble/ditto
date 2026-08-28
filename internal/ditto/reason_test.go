package ditto_test

import (
	"testing"

	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/result"
	"github.com/Disble/ditto/internal/verdict"
)

const buildFailedStream = `{"ImportPath":"example.com/x/pkg [example.com/x/pkg.test]","Action":"build-fail"}
{"Action":"fail","Package":"example.com/x/pkg","FailedBuild":"example.com/x/pkg [example.com/x/pkg.test]"}
`

const assertionStream = `{"Action":"run","Package":"example.com/x/pkg","Test":"TestAdd"}
{"Action":"fail","Package":"example.com/x/pkg","Test":"TestAdd","Elapsed":0}
`

func diagnosticFor(t *testing.T, res result.Result[string]) *ditto.Diagnostic {
	t.Helper()

	mutated := gomutatedfile.New(
		"Fake",
		"pkg/a.go",
		[]byte("package pkg\n\nvar n = 1\n"),
		[]byte("package pkg\n\nvar n = 2\n"),
	)

	return ditto.NewDiagnostic(future.Resolved(res), mutated)
}

// A killed mutant that never compiled is the defect metric 2 exists to count:
// the suite is credited with catching a mutation it never ran.
func TestADiagnosticCarriesWhyItDied(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name string
		res  result.Result[string]
		want verdict.Reason
	}{
		{"a kill from a package that would not build", result.Ok(buildFailedStream), verdict.BuildFailed},
		{"a kill a test earned", result.Ok(assertionStream), verdict.Assertion},
		{"a kill from a command that says nothing structured", result.Ok("FAIL\n"), verdict.Unknown},
		{"a survivor, which died of nothing", result.Err[string](""), verdict.Unknown},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := diagnosticFor(t, testCase.res).Reason(); got != testCase.want {
				t.Fatalf("Reason = %q, want %q", got, testCase.want)
			}
		})
	}
}

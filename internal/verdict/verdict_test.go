package verdict_test

import (
	"strings"
	"testing"

	"github.com/Disble/ditto/internal/verdict"
)

// The captured shapes are real. They were taken from go1.27 runs against a
// fixture built for this: one package that does not compile, one whose test
// fails an assertion. See docs/experiments/what-the-field-already-decided.md.
const buildFailed = `{"ImportPath":"example.com/x/pkg [example.com/x/pkg.test]","Action":"build-output","Output":"# example.com/x/pkg\n"}
{"ImportPath":"example.com/x/pkg [example.com/x/pkg.test]","Action":"build-output","Output":"pkg/a.go:3:37: undefined: notAThing\n"}
{"ImportPath":"example.com/x/pkg [example.com/x/pkg.test]","Action":"build-fail"}
{"Action":"start","Package":"example.com/x/pkg"}
{"Action":"output","Package":"example.com/x/pkg","Output":"FAIL\texample.com/x/pkg [build failed]\n"}
{"Action":"fail","Package":"example.com/x/pkg","Elapsed":0,"FailedBuild":"example.com/x/pkg [example.com/x/pkg.test]"}
`

const assertionFailed = `{"Action":"start","Package":"example.com/x/pkg"}
{"Action":"run","Package":"example.com/x/pkg","Test":"TestAdd"}
{"Action":"output","Package":"example.com/x/pkg","Test":"TestAdd","Output":"    a_test.go:8: no\n"}
{"Action":"fail","Package":"example.com/x/pkg","Test":"TestAdd","Elapsed":0}
{"Action":"fail","Package":"example.com/x/pkg","Elapsed":0.2}
`

// notJSON is what any test command that is not `go test -json` produces. It is
// the honest answer, not a fallback: ditto cannot name a reason it was never
// given, and reporting Assertion here would be the guess this metric exists to
// forbid.
const notJSON = "--- FAIL: TestAdd (0.00s)\n    a_test.go:8: no\nFAIL\n"

func TestReasonOf(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name   string
		output string
		want   verdict.Reason
	}{
		{"a package that does not compile", buildFailed, verdict.BuildFailed},
		{"a test that fails an assertion", assertionFailed, verdict.Assertion},
		{"output that is not the JSON stream", notJSON, verdict.Unknown},
		{"nothing at all", "", verdict.Unknown},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := verdict.ReasonOf(testCase.output); got != testCase.want {
				t.Fatalf("ReasonOf = %q, want %q", got, testCase.want)
			}
		})
	}
}

// The whole point of the metric: a build failure must not be reported as a kill
// a test earned. This is the assertion that fails today, on every path.
func TestABuildFailureIsNotAnAssertion(t *testing.T) {
	t.Parallel()

	if verdict.ReasonOf(buildFailed) == verdict.Assertion {
		t.Fatal("a mutant that never compiled was credited to a test")
	}
}

// Text exists because the reason is only available when the test command emits
// the JSON stream, and a stream is not something to show a human. A refusal
// that prints raw events instead of the compiler's own words is worse than the
// one it replaced.
func TestTextRebuildsWhatAHumanNeedsToRead(t *testing.T) {
	t.Parallel()

	got := verdict.Text(buildFailed)

	for _, want := range []string{"undefined: notAThing", "[build failed]"} {
		if !contains(got, want) {
			t.Fatalf("Text lost %q:\n%s", want, got)
		}
	}

	if contains(got, `"Action"`) {
		t.Fatalf("Text handed back the stream instead of the message:\n%s", got)
	}
}

// Output that was never the stream comes back untouched: there is nothing to
// rebuild, and mangling it would lose the only thing the reader has.
func TestTextLeavesPlainOutputAlone(t *testing.T) {
	t.Parallel()

	if got := verdict.Text(notJSON); got != notJSON {
		t.Fatalf("Text = %q, want it unchanged", got)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && strings.Contains(haystack, needle)
}

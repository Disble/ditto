// Package verdict names why a mutant's test command failed.
//
// Ditto recognises a killed mutant by a non-zero exit, and a mutant that does
// not compile exits non-zero too. Measured on internal/schemata/instrument.go:
// 78 mutants, 50 reported killed, of which 10 never compiled and 1 hung until
// its timeout -- 22% of the kills were not assertions. A number people act on
// that quietly contains kills no test earned is the defect docs/metrics.md
// metric 2 exists to close.
//
// The reason is read from `go test -json`, never from prose. Exit codes cannot
// carry it: measured on go1.27, `go test`, `go build`, `go vet` and
// `go test -json` all exit 1 whether a test failed or the package would not
// compile. See docs/experiments/what-the-field-already-decided.md.
package verdict

import (
	"bufio"
	"encoding/json"
	"strings"
)

// Reason is why a test command reported failure, which is how ditto recognises
// a killed mutant.
type Reason string

const (
	// Assertion is a test that ran and failed. The only reason that credits the
	// suite with catching the mutation.
	Assertion Reason = "assertion"

	// BuildFailed is a mutant that never became a program. The kill predicate is
	// undefined for it, and every source the field offers removes it from the
	// score rather than counting it -- see docs/metrics.md.
	BuildFailed Reason = "build-failed"

	// Deadline is a mutant ditto stopped itself, because its test command ran
	// past the time the unmutated suite needed. It is a KILL with its own
	// reason, not an unjudged one -- PIT's TIMED_OUT(true), Stryker's TimedOut
	// and Infection's TIMED_OUT all agree, and a suite that never returns is a
	// difference the tests did notice.
	//
	// It needs no parsing: ditto fired the clock, so ditto knows.
	Deadline Reason = "deadline"

	// Unknown is what ditto says when it was not told. A test command that is
	// not `go test -json` carries no structured reason, and guessing Assertion
	// there would manufacture exactly the unearned kills this package exists to
	// count. Unknown is the honest answer and the metric counts it.
	Unknown Reason = "unknown"
)

// event is the part of `go test -json` this needs. The rest is ignored on
// purpose: a field ditto does not read is a field that cannot make it wrong.
//
// The field names are Go's own, capitalised, which is what `go test -json`
// emits; tagliatelle wants camelCase and is overruled because this struct
// decodes somebody else's format rather than defining one.
type event struct {
	Output      string `json:"Output"`      //nolint:tagliatelle // go test -json emits these names
	Action      string `json:"Action"`      //nolint:tagliatelle // go test -json emits these names
	FailedBuild string `json:"FailedBuild"` //nolint:tagliatelle // go test -json emits these names
}

// DeadlineMarker is what a runner writes into the captured output when it
// stopped the command itself.
//
// Reading it back is not the prose-reading this package refuses. Ditto wrote
// this sentence; the ban is on inferring a verdict from somebody else's words,
// and a runner that knows it fired its own clock is not inferring anything. It
// is a sentence rather than a code because the same output goes to a reader.
const DeadlineMarker = "this mutant was killed by the deadline, not by an assertion"

// ReasonOf classifies the captured output of one test command.
//
// Two independent signals mark a build failure, and either is enough: the
// `build-fail` action, and a `fail` event carrying FailedBuild. Measured on
// go1.27, neither appears when a test fails for real.
func ReasonOf(output string) Reason {
	if strings.Contains(output, DeadlineMarker) {
		return Deadline
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	scanner.Buffer(make([]byte, 0, bufferSize), maxLine)

	sawStream := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "{") {
			continue
		}

		var decoded event
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			continue
		}

		if decoded.Action == "" {
			continue
		}

		sawStream = true

		if decoded.Action == actionBuildFail || decoded.FailedBuild != "" {
			return BuildFailed
		}
	}

	if !sawStream {
		return Unknown
	}

	return Assertion
}

// actionBuildFail is the event `go test -json` emits when a package under test
// does not compile.
const actionBuildFail = "build-fail"

// bufferSize and maxLine size the scanner. A build failure's output line can
// carry a whole compiler diagnostic, and a line the scanner refuses is a reason
// ditto would not see.
const (
	bufferSize = 64 * 1024
	maxLine    = 4 * 1024 * 1024
)

// Text rebuilds what a human would have seen, from a stream meant for a machine.
//
// The reason is only available when the test command emits `go test -json`, and
// a refusal that prints raw events instead of the compiler's own words is worse
// than the one it replaced. Output that was never the stream comes back
// untouched: there is nothing to rebuild, and mangling it would lose the only
// thing the reader has.
func Text(output string) string {
	scanner := bufio.NewScanner(strings.NewReader(output))
	scanner.Buffer(make([]byte, 0, bufferSize), maxLine)

	rebuilt := strings.Builder{}
	sawStream := false

	for scanner.Scan() {
		line := scanner.Text()

		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "{") {
			continue
		}

		var decoded event
		if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil || decoded.Action == "" {
			continue
		}

		sawStream = true

		rebuilt.WriteString(decoded.Output)
	}

	if !sawStream {
		return output
	}

	return rebuilt.String()
}

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
	Action      string `json:"Action"`      //nolint:tagliatelle // go test -json emits these names
	FailedBuild string `json:"FailedBuild"` //nolint:tagliatelle // go test -json emits these names
}

// ReasonOf classifies the captured output of one test command.
//
// Two independent signals mark a build failure, and either is enough: the
// `build-fail` action, and a `fail` event carrying FailedBuild. Measured on
// go1.27, neither appears when a test fails for real.
func ReasonOf(output string) Reason {
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

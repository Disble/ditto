package ditto_test

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/Disble/ditto/internal/gosourcefile"
	"github.com/Disble/ditto/internal/schemata"
	"github.com/Disble/ditto/viruses"
	"github.com/Disble/ditto/viruses/arithmetic"
	"github.com/Disble/ditto/viruses/comparison"
	"github.com/Disble/ditto/viruses/comparisoninvert"
	"github.com/Disble/ditto/viruses/comparisonreplace"
	"github.com/Disble/ditto/viruses/integerdecrement"
	"github.com/Disble/ditto/viruses/integerincrement"
	"github.com/Disble/ditto/viruses/rangebreak"
)

// docs/experiments/instrumentation-fidelity.md.
//
// Everything measured about the two paths so far compared totals. This compares
// which mutants are in which column, by label.

const fidelitySource = `package calc

func Grade(score int) string {
	if score > 50 {
		return "pass"
	}

	return "fail"
}

func Bonus(score int) int {
	if score >= 90 {
		return 10
	}

	return 0
}

func Label(kind int) string {
	switch kind {
	case 1:
		return "one"
	case 2:
		return "two"
	}

	return "other"
}

func Join(left, right string) string {
	return left + "-" + right
}

func FirstNonEmpty(values []string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}
`

const fidelityTests = `package calc_test

import (
	"testing"

	"scratch/calc"
)

func TestGrade(t *testing.T) {
	if calc.Grade(80) != "pass" {
		t.Fatal("80 should pass")
	}

	if calc.Grade(10) != "fail" {
		t.Fatal("10 should fail")
	}
}

func TestLabel(t *testing.T) {
	if calc.Label(1) != "one" {
		t.Fatal("1 should be one")
	}
}

func TestJoin(t *testing.T) {
	if calc.Join("a", "b") != "a-b" {
		t.Fatal("a and b should join")
	}
}
`

const fidelityExtraTest = `
func TestBonus(t *testing.T) {
	if calc.Bonus(95) != 10 {
		t.Fatal("95 should earn the bonus")
	}
}
`

// TestInstrumentationFidelity is a probe. Its assertions are properties that
// should hold forever — unlike TestChangedScope's — but it runs four whole
// releases and does not fit the gate's budget. The golden guards the same
// agreement on a small fixture, in the gate, on every commit.
func TestInstrumentationFidelity(t *testing.T) {
	if os.Getenv("DITTO_PROBE") != "1" {
		t.Skip("set DITTO_PROBE=1 to run the probes; see docs/experiments/instrumentation-fidelity.md")
	}

	t.Run("control: the comparison can see a difference", func(t *testing.T) {
		fewer := survivorsOf(t, runScratch(t, fidelitySource, fidelityTests, false))
		more := survivorsOf(t, runScratch(t, fidelitySource, fidelityTests+fidelityExtraTest, false))

		if equalSets(fewer, more) {
			t.Fatalf("adding a test changed nothing, so this comparison proves nothing\n%v", fewer)
		}

		t.Logf("without the extra test %d survivors, with it %d", len(fewer), len(more))
	})

	t.Run("H1: nothing selected behaves as the original", func(t *testing.T) {
		untouched, errUntouched := scratchSuite(t, fidelitySource, fidelityTests)
		instrumented, errInstrumented := scratchSuite(t, instrumentedSource(t), fidelityTests)

		if (errUntouched == nil) != (errInstrumented == nil) {
			t.Fatalf("H1 is falsified: untouched %v, instrumented %v\n--- untouched ---\n%s\n--- instrumented ---\n%s",
				errUntouched, errInstrumented, untouched, instrumented)
		}

		t.Logf("untouched and unselected agree: both %s", verdictOf(errUntouched))
	})

	t.Run("H2 and H3: the same mutants in the same columns", func(t *testing.T) {
		gated := runScratch(t, fidelitySource, fidelityTests, true)
		ordinary := runScratch(t, fidelitySource, fidelityTests, false)

		gatedSurvivors, ordinarySurvivors := survivorsOf(t, gated), survivorsOf(t, ordinary)

		t.Logf("gated   : total %d, killed %d, survived %d", gated.total, gated.killed, gated.survived)
		t.Logf("ordinary: total %d, killed %d, survived %d", ordinary.total, ordinary.killed, ordinary.survived)

		onlyGated := missingFrom(gatedSurvivors, ordinarySurvivors)
		onlyOrdinary := missingFrom(ordinarySurvivors, gatedSurvivors)

		for _, label := range onlyGated {
			t.Logf("  survived ONLY on the gated path   : %s", label)
		}

		for _, label := range onlyOrdinary {
			t.Logf("  survived ONLY on the ordinary path: %s", label)
		}

		if len(onlyGated)+len(onlyOrdinary) != 0 {
			t.Errorf("H2 is falsified: %d labels differ between the paths",
				len(onlyGated)+len(onlyOrdinary))
		}
	})
}

// instrumentedSource plans the file the way the gated path does, so that what is
// run here is the same bytes that path would compile.
func instrumentedSource(t *testing.T) string {
	t.Helper()

	infected := gosourcefile.New("calc/calc.go", []byte(fidelitySource)).Incubate(fidelityViruses()...)

	mutants := make([][]byte, 0, len(infected))
	for _, one := range infected {
		mutants = append(mutants, one.Mutate().Mutated())
	}

	planned := schemata.Plan([]byte(fidelitySource), mutants)

	gated := 0

	for _, id := range planned.Selector {
		if id != 0 {
			gated++
		}
	}

	// Control 2: the fixture must exercise both sides of the admission rule.
	if gated == 0 || gated == len(mutants) {
		t.Fatalf("the fixture gates %d of %d mutants, so it tests only one side",
			gated, len(mutants))
	}

	t.Logf("planner gates %d of %d mutants", gated, len(mutants))

	return string(planned.Instrumented)
}

func fidelityViruses() []viruses.Virus {
	return []viruses.Virus{
		arithmetic.New(), comparison.New(), comparisoninvert.New(), comparisonreplace.New(),
		integerdecrement.New(), integerincrement.New(), rangebreak.New(),
	}
}

var survivorLabel = regexp.MustCompile(`Mutant survived:\s*(.+)`)

func survivorsOf(t *testing.T, run release) []string {
	t.Helper()

	labels := []string{}

	for _, match := range survivorLabel.FindAllStringSubmatch(run.output, -1) {
		labels = append(labels, strings.TrimSpace(match[1]))
	}

	sort.Strings(labels)

	return labels
}

func missingFrom(these, those []string) []string {
	present := map[string]int{}
	for _, label := range those {
		present[label]++
	}

	missing := []string{}

	for _, label := range these {
		if present[label] == 0 {
			missing = append(missing, label)

			continue
		}

		present[label]--
	}

	return missing
}

func equalSets(these, those []string) bool {
	return len(missingFrom(these, those)) == 0 && len(missingFrom(those, these)) == 0
}

func verdictOf(err error) string {
	if err == nil {
		return "green"
	}

	return "red"
}

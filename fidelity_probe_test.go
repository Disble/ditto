package ditto_test

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/Disble/ditto/internal/goinfectedfile"
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

// unreadLocalSource carries the one class where the two paths were ever seen to
// disagree: a local read only in one operand of an `&&`. Comparison Replace
// swaps that operand for `true`, which leaves the local unread and the file does
// not compile — while the gate keeps the original expression as its unselected
// arm, so it does.
//
// The `&&` is the point. Comparison Replace fires only on `&&` and `||`, and
// replaces an operand rather than a whole comparison; a fixture without a
// logical operator cannot contain this class at all, whatever its comparisons
// look like. That is what the original case was: `err == nil && ...` becoming
// `true && ...`.
//
// Nothing tests Report, so a mutant that survives has nowhere else to die.
const unreadLocalSource = `package calc

func Grade(score int) string {
	if score > 50 {
		return "pass"
	}

	return "fail"
}

func Report(values []string, limit int) string {
	found := len(values)

	if found > 0 && limit > 0 {
		return "some"
	}

	return "none"
}
`

const unreadLocalTests = `package calc_test

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

		// H2 predicted the sets would be identical. They are not, and the
		// direction is what matters: mutants survive on the gated path that the
		// ordinary path could not compile, never the other way round.
		if len(onlyGated) == 0 {
			t.Errorf("H2 predicted identical sets and the paths now agree; " +
				"the disagreement this note recorded is gone")
		}

		if len(onlyOrdinary) != 0 {
			t.Errorf("%d labels survive only on the ORDINARY path, which is the "+
				"direction nothing has ever explained", len(onlyOrdinary))
		}
	})
}

// TestDisagreementClass is docs/experiments/disagreement-class.md: the fixture
// instrumentation-fidelity.md should have had.
func TestDisagreementClass(t *testing.T) {
	if os.Getenv("DITTO_PROBE") != "1" {
		t.Skip("set DITTO_PROBE=1 to run the probes; see docs/experiments/disagreement-class.md")
	}

	confirmTheFixtureCarriesTheClass(t)

	gated := runScratch(t, unreadLocalSource, unreadLocalTests, true)
	ordinary := runScratch(t, unreadLocalSource, unreadLocalTests, false)

	t.Logf("gated   : total %d, killed %d, survived %d", gated.total, gated.killed, gated.survived)
	t.Logf("ordinary: total %d, killed %d, survived %d", ordinary.total, ordinary.killed, ordinary.survived)

	onlyGated := missingFrom(survivorsOf(t, gated), survivorsOf(t, ordinary))
	onlyOrdinary := missingFrom(survivorsOf(t, ordinary), survivorsOf(t, gated))

	for _, label := range onlyGated {
		t.Logf("  survived ONLY on the gated path   : %s", label)
	}

	for _, label := range onlyOrdinary {
		t.Logf("  survived ONLY on the ordinary path: %s", label)
	}

	if len(onlyGated) == 0 && len(onlyOrdinary) == 0 {
		t.Errorf("H1 is falsified: the paths agree even here")
	}

	if len(onlyOrdinary) != 0 {
		t.Errorf("H2 is falsified: %d labels survive only on the ordinary path", len(onlyOrdinary))
	}
}

// confirmTheFixtureCarriesTheClass is every control this probe's result rests
// on, ticked before a single verdict is read.
func confirmTheFixtureCarriesTheClass(t *testing.T) {
	t.Helper()

	// The fault is confirmed present rather than assumed from a virus's name:
	// written by hand, and it must fail to build.
	replaced := strings.Replace(unreadLocalSource, "found > 0 &&", "true &&", 1)
	if output, err := scratchSuite(t, replaced, unreadLocalTests); err == nil {
		t.Fatalf("the unread local compiles, so this fixture tests nothing\n%s", output)
	}

	// Green before anything is mutated.
	if output, err := scratchSuite(t, unreadLocalSource, unreadLocalTests); err != nil {
		t.Fatalf("the fixture is not green, so no verdict below means anything\n%s", output)
	}

	// The one the first version of this note was missing: a mutation that fails
	// when written by hand proves nothing unless ditto GENERATES it.
	if broken := brokenMutantsOf(t, unreadLocalSource, unreadLocalTests); broken == 0 {
		t.Fatalf("ditto generates no mutant of this fixture that fails to build, " +
			"so the class this probe exists for is not in the population")
	}

	// A gated mutant only runs from the shared compilation if that compilation
	// succeeds. A file that does not build returns to the ordinary path, and the
	// two paths would then agree for a reason that has nothing to do with the
	// gate.
	if output, err := scratchSuite(t, plannedSource(t, unreadLocalSource), unreadLocalTests); err != nil {
		t.Fatalf("the instrumented file does not build, so the file falls back "+
			"and nothing below is about the gate\n%s", output)
	}

	t.Logf("the instrumented file builds and its suite is green")
}

// plannedSource is the bytes the gated path would compile for a file.
func plannedSource(t *testing.T, source string) string {
	t.Helper()

	infected := gosourcefile.New("calc/calc.go", []byte(source)).Incubate(fidelityViruses()...)

	mutants := make([][]byte, 0, len(infected))
	for _, one := range infected {
		mutants = append(mutants, one.Mutate().Mutated())
	}

	return string(schemata.Plan([]byte(source), mutants).Instrumented)
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

// brokenMutantsOf incubates the fixture the way a release does and reports how
// many of the mutants ditto actually produces fail to build, naming each one.
//
// It exists because a hand-written mutation that fails to compile says nothing
// about the population: the question is whether a virus writes that mutation.
func brokenMutantsOf(t *testing.T, source, tests string) int {
	t.Helper()

	infected := gosourcefile.New("calc/calc.go", []byte(source)).Incubate(fidelityViruses()...)
	broken := 0

	for index, one := range infected {
		mutant := one.Mutate()

		output, err := scratchSuite(t, string(mutant.Mutated()), tests)
		if err == nil {
			continue
		}

		if !strings.Contains(output, "declared and not used") {
			continue
		}

		broken++

		t.Logf("  ditto generates a mutant that leaves a local unread: %s (mutant %d of %d), gated: %v",
			mutant.Label(), index+1, len(infected), isGated(source, infected, index))
	}

	t.Logf("mutants generated %d, of which leave a local unread %d", len(infected), broken)

	return broken
}

// isGated asks the planner whether it admits one particular mutant's site. A
// mutant the planner refuses takes the ordinary path and carries whatever that
// path does with it, so this is what tells a fallback apart from a gate.
func isGated(source string, infected []*goinfectedfile.GoInfectedFile, index int) bool {
	mutants := make([][]byte, 0, len(infected))
	for _, one := range infected {
		mutants = append(mutants, one.Mutate().Mutated())
	}

	return schemata.Plan([]byte(source), mutants).Selector[index] != 0
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

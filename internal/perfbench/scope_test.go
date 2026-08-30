package perfbench_test

import (
	"sort"
	"testing"

	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/dittotesting/fakelogger"
	"github.com/Disble/ditto/internal/dittotesting/fakereporter"
	"github.com/Disble/ditto/internal/fsrepository"
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/gosourcefile"
	"github.com/Disble/ditto/internal/result"
	"github.com/Disble/ditto/internal/scopedrepository"
)

// addressCollectingLaboratory records WHICH mutant ran rather than how many.
//
// laboratoryRunsForOneChangedFunctionInEachOfTwoFiles counts 8 against a flat
// scope's 16, and that count is a PROXY for the property it protects. The
// property is that a scoped run reports no verdict at an address the change
// never touched -- docs/metrics.md metric 3 -- and a count cannot see an
// address. This asserts the addresses.
type addressCollectingLaboratory struct{ seen []string }

func (l *addressCollectingLaboratory) Test(
	_ ditto.Repository,
	mutated *gomutatedfile.GoMutatedFile,
) future.Future[result.Result[string]] {
	l.seen = append(l.seen, mutated.Label())

	return future.Resolved(result.Ok(""))
}

func addressesWithScope(t *testing.T, root string, ranges map[string][]gosourcefile.Range) []string {
	t.Helper()

	laboratory := &addressCollectingLaboratory{}

	ditto.New(fakelogger.New(), scopedrepository.New(ranges, fsrepository.New(root)), laboratory, fakereporter.New()).
		Release(defaultViruses()...)

	sort.Strings(laboratory.seen)

	return laboratory.seen
}

// TestAScopedRunMutatesOnlyWhatTheChangeTouched is metric 3's discriminator.
//
// The expectation needs no line arithmetic: what a two-file scope produces must
// be exactly what each file's own change produces on its own. Anything else is a
// verdict at an address the change never caused.
func TestAScopedRunMutatesOnlyWhatTheChangeTouched(t *testing.T) {
	t.Parallel()

	root := writeFixtureRepository(t)

	expected := union(
		addressesWithScope(t, root, map[string][]gosourcefile.Range{"pkg0/gate.go": {gateRange(t, root, 0)}}),
		addressesWithScope(t, root, map[string][]gosourcefile.Range{"pkg1/gate.go": {gateRange(t, root, 2)}}),
	)

	together := addressesWithScope(t, root, map[string][]gosourcefile.Range{
		"pkg0/gate.go": {gateRange(t, root, 0)},
		"pkg1/gate.go": {gateRange(t, root, 2)},
	})

	if len(expected) == 0 {
		t.Fatal("the scope produced no mutants, so this asserts nothing")
	}

	for _, address := range outside(together, expected) {
		t.Errorf("verdict at an address the change never touched: %s", address)
	}
}

// TestAFlatScopeReportsVerdictsTheChangeNeverCaused is the control. Without it
// the test above is a green that was never red: it proves the assertion can see
// the defect it is written against.
//
// Every fixture file has the same byte layout, so each range names an offset
// that exists in both files -- which is exactly the shape that makes a scope
// holding one flat set of ranges answer every file to every range.
func TestAFlatScopeReportsVerdictsTheChangeNeverCaused(t *testing.T) {
	t.Parallel()

	root := writeFixtureRepository(t)

	expected := union(
		addressesWithScope(t, root, map[string][]gosourcefile.Range{"pkg0/gate.go": {gateRange(t, root, 0)}}),
		addressesWithScope(t, root, map[string][]gosourcefile.Range{"pkg1/gate.go": {gateRange(t, root, 2)}}),
	)

	both := []gosourcefile.Range{gateRange(t, root, 0), gateRange(t, root, 2)}
	flat := addressesWithScope(t, root, map[string][]gosourcefile.Range{
		"pkg0/gate.go": both,
		"pkg1/gate.go": both,
	})

	strayed := outside(flat, expected)
	if len(strayed) == 0 {
		t.Fatal("a flat scope produced no verdict outside the change, so the assertion above cannot fail")
	}

	t.Logf("a flat scope reports %d verdicts at addresses the change never touched, for example %s",
		len(strayed), strayed[0])
}

func union(left, right []string) map[string]bool {
	all := make(map[string]bool, len(left)+len(right))
	for _, address := range append(append([]string{}, left...), right...) {
		all[address] = true
	}

	return all
}

func outside(got []string, expected map[string]bool) []string {
	stray := []string{}

	for _, address := range got {
		if !expected[address] {
			stray = append(stray, address)
		}
	}

	return stray
}

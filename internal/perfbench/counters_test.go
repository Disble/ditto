package perfbench

import (
	"encoding/json"
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/dittotesting/fakereporter"
	"github.com/Disble/ditto/internal/fsrepository"
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/result"
	"github.com/Disble/ditto/viruses"
	"github.com/Disble/ditto/viruses/arithmetic"
	"github.com/Disble/ditto/viruses/arithmeticassignment"
	"github.com/Disble/ditto/viruses/arithmeticassignmentinvert"
	"github.com/Disble/ditto/viruses/bitwise"
	"github.com/Disble/ditto/viruses/comparison"
	"github.com/Disble/ditto/viruses/comparisoninvert"
	"github.com/Disble/ditto/viruses/comparisonreplace"
	"github.com/Disble/ditto/viruses/floatdecrement"
	"github.com/Disble/ditto/viruses/floatincrement"
	"github.com/Disble/ditto/viruses/integerdecrement"
	"github.com/Disble/ditto/viruses/integerincrement"
	"github.com/Disble/ditto/viruses/loopbreak"
	"github.com/Disble/ditto/viruses/loopcondition"
	"github.com/Disble/ditto/viruses/rangebreak"
)

// baselinePath is where the recorded contract lives. Editing it is the only
// way to change a counter, which keeps every movement visible in review.
const baselinePath = "../../perf/baseline.json"

type baseline struct {
	Counters map[string]int `json:"counters"`
}

func loadBaseline(t *testing.T) baseline {
	t.Helper()

	raw, err := os.ReadFile(filepath.FromSlash(baselinePath))
	if err != nil {
		t.Fatalf("read the recorded baseline: %v", err)
	}

	var recorded baseline
	if err := json.Unmarshal(raw, &recorded); err != nil {
		t.Fatalf("decode the recorded baseline: %v", err)
	}

	return recorded
}

// assertCounter ratchets in both directions.
//
// A number that grew is a regression, and this library's whole purpose is the
// number not growing. A number that shrank is an improvement that has to be
// written down, because a gain nobody recorded is a gain that can be handed
// back later without anybody noticing.
func assertCounter(t *testing.T, name string, got int) {
	t.Helper()

	want, recorded := loadBaseline(t).Counters[name]
	if !recorded {
		t.Fatalf("counter %q measured %d but is absent from %s; record it", name, got, baselinePath)
	}

	switch {
	case got == want:
	case got > want:
		t.Errorf("REGRESSION %s: %d, baseline %d (+%d). Performance is this library's primary metric — this change costs more than the one before it.",
			name, got, want, got-want)
	default:
		t.Errorf("IMPROVED %s: %d, baseline %d (%d). Update %s to lock the gain in.",
			name, got, want, got-want, baselinePath)
	}
}

// defaultViruses mirrors the unexported set in release.go. Keeping a copy is
// deliberate: the counters below are meaningless unless the mutator set they
// were recorded against is pinned, and release.go does not export its own.
func defaultViruses() []viruses.Virus {
	return []viruses.Virus{
		arithmetic.New(), arithmeticassignment.New(), arithmeticassignmentinvert.New(),
		bitwise.New(), comparison.New(), comparisoninvert.New(), comparisonreplace.New(),
		floatdecrement.New(), floatincrement.New(), integerdecrement.New(),
		integerincrement.New(), loopbreak.New(), loopcondition.New(), rangebreak.New(),
	}
}

// countingLaboratory answers every mutant as killed without running anything,
// so the counters measure selection rather than the test command.
type countingLaboratory struct{ calls int }

func (l *countingLaboratory) Test(
	_ ditto.Repository,
	_ *gomutatedfile.GoMutatedFile,
) future.Future[result.Result[string]] {
	l.calls++

	return future.Resolved(result.Ok(""))
}

// fileCountingVirus counts parses. Each GoSourceFile.Incubate call parses the
// file and walks the tree exactly once, and every walk visits exactly one
// *ast.File, so the tally of those visits is the tally of parses.
type fileCountingVirus struct{ parses *int }

func (v fileCountingVirus) Incubate(node ast.Node, _ *types.Info) []*viruses.Infection {
	if _, isFile := node.(*ast.File); isFile {
		*v.parses++
	}

	return nil
}

func TestCounterFilesLinkedPerMutant(t *testing.T) {
	t.Parallel()

	root := writeFixtureRepository(t)
	temporary := t.TempDir()

	fsrepository.New(root).LinkAllToTemporaryRepository(temporary)

	// Laboratory.Test builds one of these per mutant, so this count is paid
	// once for every mutant in a run, not once per run.
	assertCounter(t, "filesLinkedPerMutant", countFiles(t, temporary))
}

func TestCounterSourceParsesPerRelease(t *testing.T) {
	t.Parallel()

	parses := 0
	virusCount := 3
	viri := make([]viruses.Virus, virusCount)

	for i := range virusCount {
		viri[i] = fileCountingVirus{parses: &parses}
	}

	root := writeFixtureRepository(t)
	ditto.New(fsrepository.New(root), &countingLaboratory{}, fakereporter.New()).Release(viri...)

	// With one parse per source file this equals fixtureSourceFiles. Anything
	// larger means the file is being re-parsed for every mutator.
	assertCounter(t, "sourceParsesPerReleaseWithThreeViruses", parses)
}

func TestCounterMutantsAndLaboratoryRunsPerRelease(t *testing.T) {
	t.Parallel()

	root := writeFixtureRepository(t)
	laboratory := &countingLaboratory{}
	reporter := fakereporter.New()

	ditto.New(fsrepository.New(root), laboratory, reporter).Release(defaultViruses()...)

	// One laboratory run is one execution of the test command, which is the
	// dominant cost of any run. This is the number a native staged scope has
	// to bring down.
	assertCounter(t, "laboratoryRunsPerReleaseWholeFixture", laboratory.calls)
}

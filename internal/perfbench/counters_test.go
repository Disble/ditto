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

// treeCountingVirus separates two costs that used to be the same number.
//
// Every walk of a parsed file visits exactly one *ast.File node, so counting
// those visits counts walks. Counting the *distinct* nodes counts parses,
// because a shared tree hands every mutator the same pointer and a re-parsed
// one cannot. While each file was parsed once per mutator the two were equal,
// which is what made an earlier version of this counter look like it measured
// parses when it only ever measured walks.
type treeCountingVirus struct {
	walks *int
	trees map[*ast.File]struct{}
}

func (v treeCountingVirus) Incubate(node ast.Node, _ *types.Info) []*viruses.Infection {
	if file, isFile := node.(*ast.File); isFile {
		*v.walks++
		v.trees[file] = struct{}{}
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

	walks := 0
	trees := map[*ast.File]struct{}{}
	virusCount := 3
	viri := make([]viruses.Virus, virusCount)

	for i := range virusCount {
		viri[i] = treeCountingVirus{walks: &walks, trees: trees}
	}

	root := writeFixtureRepository(t)
	ditto.New(fsrepository.New(root), &countingLaboratory{}, fakereporter.New()).Release(viri...)

	// One parse per source file is the floor: the tree does not depend on which
	// mutator is asking for it. Anything larger means the file is being
	// re-parsed per mutator.
	assertCounter(t, "sourceParsesPerReleaseWithThreeViruses", len(trees))

	// Walks are a separate cost and a separate number. Every mutator needs its
	// own pass, so this one is expected to scale with the mutator count.
	assertCounter(t, "astWalksPerReleaseWithThreeViruses", walks)
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

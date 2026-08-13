package perfbench_test

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
	"github.com/Disble/ditto/internal/fstemporarydir"
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/gosourcefile"
	"github.com/Disble/ditto/internal/laboratory"
	"github.com/Disble/ditto/internal/result"
	"github.com/Disble/ditto/internal/scopedrepository"
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

func TestCounterFilesLinkedPerSandbox(t *testing.T) {
	t.Parallel()

	root := writeFixtureRepository(t)
	temporary := t.TempDir()

	fsrepository.New(root).LinkAllToTemporaryRepository(temporary)

	// What one sandbox costs to build. It was named per-mutant while every
	// mutant built its own; that stopped being the same number the moment
	// sandboxes were pooled, so the two are counted separately now.
	assertCounter(t, "filesLinkedPerSandbox", countFiles(t, temporary))
}

// countingRepository records how many sandboxes a release actually builds.
type countingRepository struct {
	inner  ditto.Repository
	builds int
}

func (r *countingRepository) ListGoSourceFiles() []*gosourcefile.GoSourceFile {
	return r.inner.ListGoSourceFiles()
}

func (r *countingRepository) LinkAllToTemporaryRepository(path string) ditto.TemporaryRepository {
	r.builds++

	return r.inner.LinkAllToTemporaryRepository(path)
}

// silentRunner answers every mutant without running anything, so this measures
// sandbox construction rather than the test command.
type silentRunner struct{}

func (silentRunner) Test(ditto.TemporaryRepository) result.Result[string] {
	return result.Ok("")
}

func TestCounterSandboxesBuiltPerRelease(t *testing.T) {
	t.Parallel()

	root := writeFixtureRepository(t)
	repository := &countingRepository{inner: fsrepository.New(root)}

	temporaryDir := fstemporarydir.New("dittoperf-")

	t.Cleanup(func() { _ = temporaryDir.RemoveAll() })

	ditto.New(repository, laboratory.New(silentRunner{}, temporaryDir), fakereporter.New()).
		Release(defaultViruses()...)

	// One sandbox serves the whole run. This used to equal the mutant count,
	// which is the walk above paid once per mutant.
	assertCounter(t, "sandboxesBuiltPerRelease", repository.builds)
}

// releaseWithScope runs a release restricted to the given ranges and reports
// how many times the laboratory was asked to run the test command.
func releaseWithScope(t *testing.T, root string, ranges map[string][]gosourcefile.Range) int {
	t.Helper()

	laboratory := &countingLaboratory{}
	repository := scopedrepository.New(ranges, fsrepository.New(root))

	ditto.New(repository, laboratory, fakereporter.New()).Release(defaultViruses()...)

	return laboratory.calls
}

func TestCounterLaboratoryRunsForOneChangedFunction(t *testing.T) {
	t.Parallel()

	root := writeFixtureRepository(t)
	runs := releaseWithScope(t, root, map[string][]gosourcefile.Range{
		"pkg0/gate.go": {gateRange(t, root, 0)},
	})

	// Every mutant costs a full run of the test command, so this is the number
	// a change of one line should be charged: the mutators that fire on that
	// line, and nothing else in the repository.
	assertCounter(t, "laboratoryRunsForOneChangedFunction", runs)
}

func TestCounterLaboratoryRunsDoNotCollideAcrossFiles(t *testing.T) {
	t.Parallel()

	root := writeFixtureRepository(t)

	// Two files, each changed in a different place. Because every fixture file
	// has the same byte layout, the two ranges name offsets that exist in both
	// — which is exactly the shape that makes a scope holding one flat set of
	// ranges mutate each file at every range.
	runs := releaseWithScope(t, root, map[string][]gosourcefile.Range{
		"pkg0/gate.go": {gateRange(t, root, 0)},
		"pkg1/gate.go": {gateRange(t, root, 2)},
	})

	// Two changed lines cost twice one changed line. A flat scope would charge
	// four, because each file would answer to both ranges, and the bill would
	// grow as the square of the number of files rather than in proportion.
	assertCounter(t, "laboratoryRunsForOneChangedFunctionInEachOfTwoFiles", runs)
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

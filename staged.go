package ditto

import (
	"fmt"
	"os"

	"github.com/Disble/ditto/internal/staged"
)

// Prefixes narrows which files of a change a scoped run is worth asking about.
//
// It is the pair the two scoped subcommands expose as --exclude-prefix and
// --include-prefix, and it is a type rather than two slice arguments because the
// two are one decision: what this run is about. Empty is no narrowing at all,
// which is what both entry points did before this existed.
//
// Include is the one to reach for when a release refuses a scope its test
// command cannot compile: narrowing the run to the package the command names is
// cheaper and more precise than widening the command, and it is one flag instead
// of one per package the change happens to touch.
type Prefixes struct {
	// Exclude names repository-relative prefixes never worth mutating.
	Exclude []string
	// Include names the only repository-relative prefixes worth mutating. Empty
	// means every prefix, so a run with no Include mutates everything Exclude
	// left.
	Include []string
}

// StagedPlan is what a staged change justifies mutating, before anything runs.
type StagedPlan struct {
	// Root is the repository the plan was read from.
	Root string
	// Files are the staged sources worth mutating, repository-relative with
	// forward slashes.
	Files []string
	// Ranges is the scope, keyed by file. A file mapped to no ranges is mutated
	// whole, which is what failing open means.
	Ranges map[string][]Range
	// Derived is false when the diff could not be turned into byte ranges and
	// the plan fell back to whole files. Reason then says why.
	Derived bool
	Reason  string
}

// ScopeNotice is what a run has to say about its own scope, and empty when the
// scope held.
//
// When the diff cannot be turned into byte ranges the plan falls back to whole
// files -- and widens EVERY file, not only the one it could not read. Verdicts
// then come back at addresses the change never touched, which is a lie about
// scope rather than merely a wider bill: measured on the fixture in
// internal/perfbench, a scope that does not keep each range beside its file
// strays to eight such addresses.
//
// Derived and Reason have been on this plan since the beginning and only --dry
// ever printed them, so a real run widened in silence. docs/metrics.md metric 3.
func (p StagedPlan) ScopeNotice() string {
	if p.Derived {
		return ""
	}

	return "ditto: the staged scope could not be derived, so every staged file is mutated as a " +
		"whole -- survivors may sit on lines this change never touched. Reason: " + p.Reason
}

// Mutable reports whether there is anything to do.
func (p StagedPlan) Mutable() bool { return len(p.Files) > 0 }

// PlanStaged answers what a staged change justifies, and changes nothing.
//
// It is the whole of `--dry`: the question "what would this cost" is worth
// asking on its own, and answering it must not write a sandbox or start a suite.
// It therefore does not read `.ditto.json` either -- that names what a sandbox
// needs, and this builds none. See RunStaged.
func PlanStaged(directory string, prefixes Prefixes) (StagedPlan, error) {
	repository, err := staged.New(staged.OSRunner{}, directory)
	if err != nil {
		return StagedPlan{}, fmt.Errorf("reading the repository: %w", err)
	}

	files, err := repository.Files(prefixes.Exclude, prefixes.Include)
	if err != nil {
		return StagedPlan{}, fmt.Errorf("reading the staged files: %w", err)
	}

	plan := StagedPlan{Root: repository.Root(), Files: files, Ranges: map[string][]Range{}}
	if len(files) == 0 {
		return plan, nil
	}

	if err := repository.RejectPartial(files); err != nil {
		return StagedPlan{}, fmt.Errorf("checking the staged files: %w", err)
	}

	scope, err := repository.ScopeOf(files)
	if err != nil {
		return StagedPlan{}, fmt.Errorf("reading the staged scope: %w", err)
	}

	plan.Ranges = rangesFrom(scope.Ranges)
	plan.Derived = scope.Derived
	plan.Reason = scope.Reason

	return plan, nil
}

// RunStaged mutates exactly what a staged change justifies.
//
// The release is pointed at a checkout of the index rather than at the worktree,
// and that is the part that must not be dropped. Measured on a fixture built for
// it: against the worktree, with one tracked file left dirty and unstaged, seven
// of eight verdicts moved. The staged-file check does not cover that case,
// because the file that moved them was never staged.
//
// A repository that does not build from its index alone names the generated
// paths git does not carry in a `.ditto.json` at its root:
//
//	{"generated": ["frontend/dist", "frontend/wailsjs"]}
//
// Those are copied from the working tree after the index is materialised, and
// each copy is announced. Naming a path git tracks is refused, because the index
// version is the one a staged run measures.
//
// Options are applied after the scope and the root, so a caller can set a
// threshold or a test command but cannot quietly point the run somewhere else.
func RunStaged(directory string, prefixes Prefixes, options ...Option) error {
	plan, err := PlanStaged(directory, prefixes)
	if err != nil {
		return err
	}

	if !plan.Mutable() {
		return nil
	}

	return runInSandbox(directory, plan, options)
}

// runInSandbox is everything a scoped release does once its scope is known.
//
// Staged and changed differ in exactly one thing — which pair of trees the diff
// is read from — and shared everything below it: the same sandbox built from the
// index, the same `.ditto.json` for what git does not carry, the same notice
// when the diff could not be turned into ranges. Keeping one copy is not tidying:
// two copies of this drift, and a drift here means one entry point measuring
// different bytes than the other while both report the same kind of number.
func runInSandbox(directory string, plan StagedPlan, options []Option) error {
	repository, err := staged.New(staged.OSRunner{}, directory)
	if err != nil {
		return fmt.Errorf("reading the repository: %w", err)
	}

	sandbox, err := repository.Materialize()
	if err != nil {
		return fmt.Errorf("materialising the content to measure: %w", err)
	}
	defer sandbox.Close()

	// What git does not carry, named one path at a time by the repository. See
	// staged.Config: the index stays what is measured, and this only fills what
	// it never had.
	config, err := repository.LoadConfig()
	if err != nil {
		return fmt.Errorf("reading the repository configuration: %w", err)
	}

	// Said before anything runs, because a reader who sees survivors at lines
	// they never touched deserves to know the scope widened before they go
	// looking for the bug.
	if notice := plan.ScopeNotice(); notice != "" {
		fmt.Fprintln(os.Stdout, notice)
	}

	copied, err := repository.CopyGenerated(sandbox, config.Generated)
	if err != nil {
		return fmt.Errorf("copying generated paths: %w", err)
	}

	// Said out loud, because these are the only bytes in the sandbox that did
	// not come from the index, and a reader deciding whether to trust a verdict
	// deserves to know which ones those are.
	for _, name := range copied {
		fmt.Fprintf(os.Stdout, "ditto: %s is generated and untracked; copied from the working tree.\n", name)
	}

	scoped := append([]Option{
		WithRepositoryRoot(sandbox.Root),
		WithChangedRanges(plan.Ranges),
	}, options...)

	return Run(scoped...)
}

// rangesFrom converts the internal span type into the published one. The two are
// kept apart so an internal package never becomes part of this module's surface.
func rangesFrom(ranges map[string][]staged.Range) map[string][]Range {
	converted := make(map[string][]Range, len(ranges))

	for path, spans := range ranges {
		fileRanges := make([]Range, 0, len(spans))
		for _, span := range spans {
			fileRanges = append(fileRanges, Range{Start: span.Start, End: span.End})
		}

		converted[path] = fileRanges
	}

	return converted
}

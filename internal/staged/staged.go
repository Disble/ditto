// Package staged answers three questions about a change that is staged but not
// committed: which files it touches, which bytes of them, and which bytes the
// suite should be run against.
//
// The third is the one that is easy to miss and expensive to get wrong. Mutants
// come from the staged content, but the test command runs against a whole tree,
// and a tracked file left dirty in the worktree decides what the suite proves.
// Measured on a fixture built for it: pointed at the worktree instead of the
// index, seven of eight verdicts moved — a score of 0.13 against 1.00 for the
// identical eight mutants of an identical file. A guard that only inspects the
// staged paths cannot see that, because the file that moved them was never
// staged.
//
// So the index is materialised and the release is pointed at that. It is not an
// optimisation and it is not belt-and-braces; it is the difference between
// measuring the change and measuring the desk it was written on.
package staged

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// Range is a half-open byte span within one file, counted from its first byte.
type Range struct {
	Start int
	End   int
}

// Scope is what a staged change justifies mutating.
//
// Ranges is keyed by file and stays keyed by file. A byte offset only means
// something against the file it was measured in, so a scope that has lost track
// of which file a range came from makes every file answer to every range: the
// mutant count then grows as the square of the file count, and the extra ones
// land in code no diff touched.
type Scope struct {
	Files   []string
	Ranges  map[string][]Range
	Derived bool
	Reason  string
}

// Runner is the seam over running a process, so the git work can be tested
// without a repository.
type Runner interface {
	Output(dir, name string, args ...string) ([]byte, error)
}

// gitEnvironment is what git exports to a hook, and in a linked worktree it
// exports them as absolute paths. Anything spawned from a hook inherits them, so
// a command meant for the directory it was handed addresses the hook's
// repository instead — and then succeeds, which is the whole problem.
//
// Measured twice before this list existed: once leaving a stray commit on a live
// branch, once writing core.bare and an identity into a shared config.
var gitEnvironment = []string{ //nolint:gochecknoglobals // one fixed list, read only
	"GIT_DIR",
	"GIT_INDEX_FILE",
	"GIT_WORK_TREE",
	"GIT_OBJECT_DIRECTORY",
	"GIT_COMMON_DIR",
}

// OSRunner runs processes with the inherited git addressing removed.
type OSRunner struct{}

func (OSRunner) Output(dir, name string, args ...string) ([]byte, error) {
	command := exec.Command(name, args...) //nolint:noctx // there is no cancellation contract here yet
	command.Dir = dir
	command.Env = withoutGitEnvironment(os.Environ())

	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, bytes.TrimSpace(output))
	}

	return output, nil
}

func withoutGitEnvironment(environment []string) []string {
	kept := make([]string, 0, len(environment))

	for _, variable := range environment {
		name, _, _ := strings.Cut(variable, "=")
		if !contains(gitEnvironment, name) {
			kept = append(kept, variable)
		}
	}

	return kept
}

func contains(names []string, name string) bool {
	return slices.Contains(names, name)
}

// Repository is a checkout the staged questions can be asked of.
type Repository struct {
	root   string
	runner Runner
}

// New binds the questions to a checkout. The root is resolved by git rather than
// taken on trust, so a command run from a subdirectory means the same thing.
func New(runner Runner, directory string) (*Repository, error) {
	output, err := runner.Output(directory, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("resolving the repository root: %w", err)
	}

	return &Repository{root: strings.TrimSpace(string(output)), runner: runner}, nil
}

// Root is the directory every other answer is relative to.
func (r *Repository) Root() string { return r.root }

// git runs one git command in the repository root. The error is returned as it
// arrives: the runner already names the command and carries git own output, and
// wrapping it again would say the same thing twice.
func (r *Repository) git(args ...string) ([]byte, error) {
	return r.runner.Output(r.root, "git", args...) //nolint:wrapcheck // the runner already names the command and its output
}

// Files lists the staged Go sources worth mutating: not tests, because they are
// the oracle rather than the subject, and not anything under a prefix the caller
// excluded.
func (r *Repository) Files(excludedPrefixes []string) ([]string, error) {
	output, err := r.git("diff", "--cached", "--name-only", "--diff-filter=ACMR", "-z")
	if err != nil {
		return nil, fmt.Errorf("listing the staged files: %w", err)
	}

	return selectMutable(splitNUL(output), excludedPrefixes), nil
}

func selectMutable(files, excludedPrefixes []string) []string {
	selected := []string{}

	for _, file := range files {
		file = strings.ReplaceAll(file, "\\", "/")
		if file != "" && isMutable(file, excludedPrefixes) {
			selected = append(selected, file)
		}
	}

	return selected
}

func isMutable(file string, excludedPrefixes []string) bool {
	if !strings.HasSuffix(file, ".go") || strings.HasSuffix(file, "_test.go") {
		return false
	}

	for _, prefix := range excludedPrefixes {
		if prefix != "" && strings.HasPrefix(file, prefix) {
			return false
		}
	}

	return true
}

// RejectPartial refuses a staged file that also has unstaged edits.
//
// The scope is derived from the index and the mutants are written into a copy of
// the index, so worktree-only edits to a staged file are content the verdict was
// never about. Reporting on it as though it were is worse than refusing.
func (r *Repository) RejectPartial(files []string) error {
	for _, file := range files {
		output, err := r.git("diff", "--name-only", "-z", "--", file)
		if err != nil {
			return fmt.Errorf("checking partial staging for %s: %w", file, err)
		}

		if len(output) > 0 {
			return fmt.Errorf( //nolint:err113 // the path is the message
				"ditto: %s is staged and also edited in the worktree; stage or discard the rest of it", file)
		}
	}

	return nil
}

// ScopeOf converts each staged diff into byte ranges of the staged content.
//
// It fails open rather than guessing. A diff that cannot be parsed, or a range
// that does not land on index bytes, produces a whole-file scope and says why —
// mutating too much is a cost, and mutating the wrong bytes is a wrong answer.
func (r *Repository) ScopeOf(files []string) (Scope, error) {
	scope := Scope{Files: files, Ranges: map[string][]Range{}, Derived: true}

	for _, file := range files {
		diff, err := r.git("-c", "core.quotePath=false", "diff", "--cached", "--no-ext-diff", "--no-renames", "-U0", "--", file)
		if err != nil {
			return Scope{}, fmt.Errorf("reading the staged diff for %s: %w", file, err)
		}

		content, err := r.git("show", ":"+file)
		if err != nil {
			return Scope{}, fmt.Errorf("reading the staged content of %s: %w", file, err)
		}

		lines, parseErr := changedLines(string(diff))
		if parseErr != nil || len(lines) == 0 {
			//nolint:nilerr // failing open IS the answer here: a scope that cannot be derived is not an error, it is a wider scope that says why
			return failOpen(files, "a staged diff yielded no usable range; mutating whole files"), nil
		}

		offsets := merge(offsetsOf(content, lines))
		if len(offsets) == 0 {
			return failOpen(files, "a staged line range did not land on index bytes; mutating whole files"), nil
		}

		scope.Ranges[file] = offsets
	}

	return scope, nil
}

func failOpen(files []string, reason string) Scope {
	ranges := map[string][]Range{}
	for _, file := range files {
		ranges[file] = nil
	}

	return Scope{Files: files, Ranges: ranges, Derived: false, Reason: reason}
}

var hunkHeader = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

type lineSpan struct{ first, last int }

// changedLines reads the new side of each hunk header. Only the new side
// matters: the old side names bytes that no longer exist and cannot be mutated.
func changedLines(diff string) ([]lineSpan, error) {
	spans := []lineSpan{}

	for line := range strings.SplitSeq(diff, "\n") {
		matches := hunkHeader.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		start, err := strconv.Atoi(matches[1])
		if err != nil {
			return nil, fmt.Errorf("reading the hunk start in %q: %w", line, err)
		}

		count := 1

		if matches[2] != "" {
			count, err = strconv.Atoi(matches[2])
			if err != nil {
				return nil, fmt.Errorf("reading the hunk count in %q: %w", line, err)
			}
		}

		if count == 0 {
			spans = append(spans, lineSpan{first: max(start, 1), last: start + 1})

			continue
		}

		spans = append(spans, lineSpan{first: start, last: start + count - 1})
	}

	return spans, nil
}

func offsetsOf(content []byte, spans []lineSpan) []Range {
	starts := []int{0}

	for index, value := range content {
		if value == '\n' && index+1 < len(content) {
			starts = append(starts, index+1)
		}
	}

	offsets := make([]Range, 0, len(spans))

	for _, span := range spans {
		if span.first > len(starts) {
			continue
		}

		start := starts[max(span.first, 1)-1]
		end := len(content)

		if span.last < len(starts) {
			end = starts[span.last]
		}

		if end > start {
			offsets = append(offsets, Range{Start: start, End: end})
		}
	}

	return offsets
}

func merge(ranges []Range) []Range {
	if len(ranges) == 0 {
		return nil
	}

	sorted := append([]Range(nil), ranges...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Start == sorted[j].Start {
			return sorted[i].End < sorted[j].End
		}

		return sorted[i].Start < sorted[j].Start
	})

	merged := []Range{sorted[0]}

	for _, next := range sorted[1:] {
		last := &merged[len(merged)-1]
		if next.Start <= last.End {
			last.End = max(last.End, next.End)

			continue
		}

		merged = append(merged, next)
	}

	return merged
}

func splitNUL(output []byte) []string {
	trimmed := strings.TrimSuffix(string(output), "\x00")
	if trimmed == "" {
		return nil
	}

	return strings.Split(trimmed, "\x00")
}

package staged

import (
	"fmt"
	"strings"
)

// The staged questions read the index. These read a RANGE, and they exist
// because backlog entry 21 measured what the index cannot answer: on a CI
// checkout nothing is staged, so a gate pointed at the staged scope skips and
// reports a green that measured nothing — the shape of failure this repository
// refuses. The change is still there; it is just already committed.
//
// Everything else is shared. The diff parsing, the byte offsets, the fail-open
// rule and the sandbox are the same, because the question is the same one asked
// of a different pair of trees.

// ChangedFiles lists the Go sources a range touched, on the same terms Files
// uses: not tests, and not anything under an excluded prefix.
func (r *Repository) ChangedFiles(baseRef string, excludedPrefixes []string) ([]string, error) {
	output, err := r.git("diff", "--name-only", "--diff-filter=ACMR", "-z", rangeOf(baseRef))
	if err != nil {
		return nil, fmt.Errorf("listing the files changed since %s: %w", baseRef, err)
	}

	return selectMutable(splitNUL(output), excludedPrefixes), nil
}

// ChangedScopeOf converts each file's range diff into byte ranges of HEAD.
//
// It fails open exactly as ScopeOf does, and for the same reason: mutating too
// much is a cost, and mutating the wrong bytes is a wrong answer.
func (r *Repository) ChangedScopeOf(baseRef string, files []string) (Scope, error) {
	scope := Scope{Files: files, Ranges: map[string][]Range{}, Derived: true}

	for _, file := range files {
		diff, err := r.git("-c", "core.quotePath=false", "diff", "--no-ext-diff", "--no-renames", "-U0",
			rangeOf(baseRef), "--", file)
		if err != nil {
			return Scope{}, fmt.Errorf("reading the diff for %s since %s: %w", file, baseRef, err)
		}

		content, err := r.git("show", "HEAD:"+file)
		if err != nil {
			return Scope{}, fmt.Errorf("reading the committed content of %s: %w", file, err)
		}

		lines, parseErr := changedLines(string(diff))
		if parseErr != nil || len(lines) == 0 {
			//nolint:nilerr // failing open IS the answer: a scope that cannot be derived is a wider scope that says why
			return failOpen(files, "a diff since "+baseRef+" yielded no usable range; mutating whole files"), nil
		}

		offsets := merge(offsetsOf(content, lines))
		if len(offsets) == 0 {
			return failOpen(files, "a changed line range did not land on committed bytes; mutating whole files"), nil
		}

		scope.Ranges[file] = offsets
	}

	return scope, nil
}

// rangeOf asks what the change ADDED rather than how two lines of work drifted
// apart. `base...HEAD` is the diff against their merge base, so a base that has
// moved on since the change was written does not widen the scope with somebody
// else's commits.
func rangeOf(baseRef string) string { return baseRef + "...HEAD" }

// RequireNothingStaged refuses a checkout whose index has moved away from HEAD.
//
// This is what makes the range scope safe to run in the existing sandbox, and
// the precise condition matters. That sandbox is `git checkout-index --all`, so
// it holds the INDEX; a range scope names bytes of HEAD. The two agree exactly
// while the index agrees with HEAD, which is a narrower thing than a clean
// worktree.
//
// It was written as a clean-worktree check first, and CI refuted that in
// fifty-one seconds: the Devbox install step modifies `devbox.lock`, a tracked
// file with nothing to do with any change, and the gate refused to run at all. A
// worktree modification is never written into the sandbox, so it cannot move a
// verdict; a STAGED one is, and would mean scoping against one tree while
// mutating another -- the defect already measured at seven of eight verdicts
// moving, and silent.
//
// The staged path needs no such check: it derives its scope from the index and
// mutates the index, and RejectPartial covers the one file that could disagree.
func (r *Repository) RequireNothingStaged() error {
	output, err := r.git("diff", "--cached", "--name-only")
	if err != nil {
		return fmt.Errorf("checking whether anything is staged: %w", err)
	}

	staged := strings.TrimSpace(string(output))
	if staged == "" {
		return nil
	}

	return fmt.Errorf( //nolint:err113 // the listing is the message
		"ditto: a range scope measures committed bytes, and this checkout has staged changes:\n%s", staged)
}

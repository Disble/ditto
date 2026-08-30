package dittotesting

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The range scope's whole subject is what git says, so its tests need real
// repositories rather than a double: a fake that agrees with my reading of git
// proves my reading rather than the behaviour. Two packages need them — the
// library's own tests and the command's — and one copy of this is enough.

// GitRepository builds a throwaway repository with one committed Go file, tagged
// `base`, and returns its path. It is removed with the test's temporary
// directory.
func GitRepository(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	Git(t, dir, "init")
	Git(t, dir, "config", "user.email", "fixture@example.com")
	Git(t, dir, "config", "user.name", "fixture")
	WriteFile(t, dir, "kept.go", "package fixture\n\nfunc Kept() int { return 1 }\n")
	Git(t, dir, "add", "-A")
	Git(t, dir, "commit", "-m", "base")
	Git(t, dir, "tag", "base")

	return dir
}

// GitRepositoryWithAChange is GitRepository plus one committed Go change after
// the tag, which is the shape most of these questions are asked of.
func GitRepositoryWithAChange(t *testing.T) string {
	t.Helper()

	dir := GitRepository(t)

	WriteFile(t, dir, "added.go", "package fixture\n\nfunc Added(a, b int) bool { return a > b }\n")
	Git(t, dir, "add", "-A")
	Git(t, dir, "commit", "-m", "add")

	return dir
}

// Git runs one git command in a fixture, with the addressing this process
// inherited removed.
//
// A hook exports GIT_DIR, GIT_INDEX_FILE and friends as absolute paths, and
// everything spawned below inherits them — so a fixture that kept them would
// quietly operate on the real checkout and succeed. They are REMOVED rather than
// blanked, because git rejects an empty GIT_DIR outright.
func Git(t *testing.T, dir string, args ...string) {
	t.Helper()

	command := exec.CommandContext(t.Context(), "git", args...)
	command.Dir = dir

	kept := make([]string, 0, len(os.Environ()))

	for _, variable := range os.Environ() {
		name, _, _ := strings.Cut(variable, "=")
		if !strings.HasPrefix(name, "GIT_") {
			kept = append(kept, variable)
		}
	}

	command.Env = kept

	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

// WriteFile puts one file into a fixture, creating nothing else.
func WriteFile(t *testing.T, dir, name, content string) {
	t.Helper()

	path := filepath.Join(dir, name)

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("creating the directory for %s: %v", name, err)
	}

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

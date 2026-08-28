package staged

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGit answers ls-files the way git does: non-zero when it knows nothing
// about a path, and the path itself when it does.
type fakeGit struct{ tracked map[string]bool }

func (g fakeGit) Output(_, _ string, args ...string) ([]byte, error) {
	path := args[len(args)-1]
	if g.tracked[filepath.ToSlash(path)] {
		return []byte(path + "\x00"), nil
	}

	return nil, os.ErrNotExist
}

func repositoryAt(t *testing.T, root string, tracked ...string) *Repository {
	t.Helper()

	known := map[string]bool{}
	for _, path := range tracked {
		known[path] = true
	}

	return &Repository{root: root, runner: fakeGit{tracked: known}}
}

// A repository with nothing to say says nothing, and that is not an error: most
// of them do build from their index and need no configuration at all.
func TestNoConfigIsNotAnError(t *testing.T) {
	t.Parallel()

	config, err := repositoryAt(t, t.TempDir()).LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if len(config.Generated) != 0 {
		t.Fatalf("generated = %v, want none", config.Generated)
	}
}

func TestConfigNamesTheGeneratedPaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	write(t, filepath.Join(root, ConfigName), `{"generated": ["frontend/dist", "frontend/wailsjs"]}`)

	config, err := repositoryAt(t, root).LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if len(config.Generated) != 2 || config.Generated[0] != "frontend/dist" {
		t.Fatalf("generated = %v", config.Generated)
	}
}

func TestUnreadableConfigIsAnErrorRatherThanAnEmptyOne(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	write(t, filepath.Join(root, ConfigName), "{ this is not json")

	if _, err := repositoryAt(t, root).LoadConfig(); err == nil {
		t.Fatal("a malformed configuration was read as an empty one")
	}
}

func TestGeneratedPathsReachTheSandbox(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	write(t, filepath.Join(root, "frontend", "dist", "index.html"), "<!doctype html>")
	write(t, filepath.Join(root, "frontend", "dist", "assets", "app.js"), "console.log(1)")

	sandbox := &Sandbox{Root: t.TempDir()}

	copied, err := repositoryAt(t, root).CopyGenerated(sandbox, []string{"frontend/dist"})
	if err != nil {
		t.Fatalf("CopyGenerated: %v", err)
	}

	if len(copied) != 1 {
		t.Fatalf("copied = %v, want the one path", copied)
	}

	// The nested file matters: a copy that flattens or stops at the first level
	// would still satisfy `go:embed` on the directory and lose the bundle.
	if got := read(t, filepath.Join(sandbox.Root, "frontend", "dist", "assets", "app.js")); got != "console.log(1)" {
		t.Fatalf("nested file = %q", got)
	}
}

// This is the guard, and it is the whole reason the setting is safe. A tracked
// path has an index version, and a staged run measures that one. Letting the
// working tree's win here would reopen the hole the sandbox exists to close --
// measured at 7 of 8 verdicts moving when a release read the worktree instead.
func TestATrackedPathIsRefused(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	write(t, filepath.Join(root, "internal", "calc", "calc.go"), "package calc")

	_, err := repositoryAt(t, root, "internal/calc").
		CopyGenerated(&Sandbox{Root: t.TempDir()}, []string{"internal/calc"})
	if err == nil {
		t.Fatal("a tracked path was copied from the working tree")
	}

	if !strings.Contains(err.Error(), "tracked by git") {
		t.Fatalf("the refusal does not say why: %v", err)
	}
}

// A named path that is not there is refused rather than skipped. It was named
// because the build needs it, so a sandbox quietly missing it produces exactly
// the failure this setting exists to prevent.
func TestAMissingPathIsRefusedRatherThanSkipped(t *testing.T) {
	t.Parallel()

	_, err := repositoryAt(t, t.TempDir()).
		CopyGenerated(&Sandbox{Root: t.TempDir()}, []string{"frontend/dist"})
	if err == nil {
		t.Fatal("a missing path was skipped")
	}

	if !strings.Contains(err.Error(), "frontend/dist") {
		t.Fatalf("the refusal does not name the path: %v", err)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}

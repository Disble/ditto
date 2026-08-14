package ditto_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestReleaseGolden runs a whole release against a throwaway copy of
// testdata/goldenproject and compares what ditto printed, byte for byte, with
// testdata/golden/release.txt.
//
// It exists because the failure that costs the most here is a silent one: a
// change that keeps every unit test green while quietly flipping a mutant from
// killed to survived. perf/baseline.json ratchets what a run costs; nothing
// ratcheted what a run says. This does.
//
// The fixture is copied to a temporary directory and mutated there. It is never
// mutated where it sits, and the run is pointed at the copy — the rule from
// AGENTS.md, applied to ditto's own suite rather than quoted at other people.
func TestReleaseGolden(t *testing.T) {
	if testing.Short() {
		t.Skip("runs a full release: one test process per mutant")
	}

	moduleRoot := moduleRoot(t)
	project := t.TempDir()

	copyTree(t, filepath.Join(moduleRoot, "testdata", "goldenproject"), project)
	writeGoMod(t, project, moduleRoot)

	// Resolved rather than pinned here: ditto's own go.mod already decides these
	// versions, and a second copy of them in a fixture is a copy that goes stale
	// without anything failing.
	tidy := command(t, project, "go", "mod", "tidy")

	output, err := tidy.CombinedOutput()
	if err != nil {
		t.Fatalf("resolving the fixture's dependencies: %v\n%s", err, output)
	}

	binary := filepath.Join(project, "mutation.test")
	build := command(t, project, "go", "test", "-c", "-tags=mutation", "-o", binary, ".")

	output, err = build.CombinedOutput()
	if err != nil {
		t.Fatalf("building the fixture's mutation binary: %v\n%s", err, output)
	}

	run := command(t, project, binary, "-test.run", "TestMutation", "-test.count=1")

	output, err = run.CombinedOutput()
	if err != nil {
		t.Fatalf("running the release: %v\n%s", err, output)
	}

	got := strings.ReplaceAll(string(output), "\r\n", "\n")
	golden := filepath.Join(moduleRoot, "testdata", "golden", "release.txt")

	if os.Getenv("DITTO_GOLDEN_UPDATE") == "1" {
		writeFile(t, golden, got)
		t.Log("golden updated; rerun without DITTO_GOLDEN_UPDATE to check it")

		return
	}

	want := readFile(t, golden)
	if got != want {
		t.Fatalf("the release said something different.\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}

	// The gated path has to say exactly this too. It compiles the package once
	// and selects each mutant at run time instead of starting the test command
	// for every one, and a mutation tester that changed its verdicts to go
	// faster would have broken the only thing it is for.
	gated := command(t, project, binary, "-test.run", "TestMutation", "-test.count=1")
	gated.Env = append(gated.Env, "DITTO_GOLDEN_GATED=1")

	output, err = gated.CombinedOutput()
	if err != nil {
		t.Fatalf("running the gated release: %v\n%s", err, output)
	}

	if got := strings.ReplaceAll(string(output), "\r\n", "\n"); got != want {
		t.Fatalf("the gated release said something different.\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}

// command builds a command that cannot address the repository this test runs
// in. Git exports GIT_DIR and GIT_INDEX_FILE to hooks, absolute in a linked
// worktree, and everything spawned below inherits them — so a `git` call meant
// for the fixture would reach the real checkout instead, succeed, and say
// nothing. Removing them costs one line.
func command(t *testing.T, dir, name string, args ...string) *exec.Cmd {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), name, args...)
	cmd.Dir = dir
	cmd.Env = withoutGitEnvironment(os.Environ())

	return cmd
}

func withoutGitEnvironment(environment []string) []string {
	inherited := []string{
		"GIT_DIR=", "GIT_INDEX_FILE=", "GIT_WORK_TREE=",
		"GIT_OBJECT_DIRECTORY=", "GIT_COMMON_DIR=",
	}

	kept := make([]string, 0, len(environment))

	for _, variable := range environment {
		if !hasAnyPrefix(variable, inherited) {
			kept = append(kept, variable)
		}
	}

	return kept
}

func hasAnyPrefix(value string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}

	return false
}

// writeGoMod points the fixture at the module under test, so the golden records
// what this working tree does rather than what the last published tag does.
func writeGoMod(t *testing.T, project, moduleRoot string) {
	t.Helper()

	writeFile(t, filepath.Join(project, "go.mod"), `module goldenproject

go 1.25

require github.com/Disble/ditto v0.0.0

replace github.com/Disble/ditto => `+filepath.ToSlash(moduleRoot)+"\n")
}

func moduleRoot(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate this test file")
	}

	return filepath.Dir(thisFile)
}

func copyTree(t *testing.T, from, to string) {
	t.Helper()

	err := filepath.WalkDir(from, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relative, err := filepath.Rel(from, path)
		if err != nil {
			return fmt.Errorf("relative path of %s: %w", path, err)
		}

		target := filepath.Join(to, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o750)
		}

		copyFile(t, path, target)

		return nil
	})
	if err != nil {
		t.Fatalf("copying the fixture: %v", err)
	}
}

func copyFile(t *testing.T, from, to string) {
	t.Helper()

	writeFile(t, to, readFile(t, from))
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	return strings.ReplaceAll(string(content), "\r\n", "\n")
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(path), err)
	}

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

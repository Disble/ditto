package ditto_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/Disble/ditto"
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

	// The gated path has to say exactly this too, apart from the one line that
	// says how it ran. It compiles the package once and selects each mutant at
	// run time instead of starting the test command for every one, and a
	// mutation tester that changed its verdicts to go faster would have broken
	// the only thing it is for.
	gatedOutput := releaseOutput(t, project, binary, true, false)

	if got := withoutGateCount(gatedOutput); got != want {
		t.Fatalf("the gated release said something different.\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}

	// And it has to say that it gated something. The counts are the only thing
	// that separates a run which gated every mutant from one that gated none:
	// until they were printed, the two produced identical output.
	quiet := gateCount(t, gatedOutput)
	if quiet == 0 {
		t.Fatalf("the gated release gated nothing:\n%s", gatedOutput)
	}

	// The same run under -v, which is the flag CI runs and the one that used to
	// turn the gated path off without saying so: `Release` wraps the stack in
	// verboselaboratory, and a decorator that does not forward the batch makes
	// every laboratory beneath it look like one that cannot take it. Measured on
	// this fixture at 4 of 7 without -v and none of 7 with it —
	// docs/experiments/forwarding-the-batch.md. Nothing refused it, which is why
	// it cost three measurements before a hand-placed panic found it.
	verbose := gateCount(t, releaseOutput(t, project, binary, true, true))
	if verbose != quiet {
		t.Fatalf("the gated release gated %d mutants under -v and %d without it; "+
			"verbose is meant to change what is logged, not what runs", verbose, quiet)
	}
}

func releaseOutput(t *testing.T, project, binary string, gated, verbose bool) string {
	t.Helper()

	args := []string{"-test.run", "TestMutation", "-test.count=1"}
	if verbose {
		args = append(args, "-test.v")
	}

	run := command(t, project, binary, args...)
	if gated {
		run.Env = append(run.Env, "DITTO_GOLDEN_GATED=1")
	}

	output, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("running the release (gated=%v verbose=%v): %v\n%s", gated, verbose, err, output)
	}

	return strings.ReplaceAll(string(output), "\r\n", "\n")
}

var gateCountPattern = regexp.MustCompile(`┃ Gated: (none|\d+) of (\d+) mutants`)

// gateCount reads the line the gated reporter prints. Absent is a failure rather
// than a zero: a run that never printed it and a run that gated nothing are
// exactly the two cases this is here to separate.
func gateCount(t *testing.T, output string) int {
	t.Helper()

	match := gateCountPattern.FindStringSubmatch(output)
	if match == nil {
		t.Fatalf("the gated release printed no gate count at all:\n%s", output)
	}

	if match[1] == "none" {
		return 0
	}

	count, err := strconv.Atoi(match[1])
	if err != nil {
		t.Fatalf("gate count %q is not a number: %v", match[1], err)
	}

	return count
}

// withoutGateCount removes the one line a gated run adds, so everything else can
// be compared against the golden byte for byte. That line describes how the run
// executed; what the golden pins is what it found.
func withoutGateCount(output string) string {
	kept := []string{}

	for line := range strings.SplitSeq(output, "\n") {
		if gateCountPattern.MatchString(line) {
			continue
		}

		kept = append(kept, line)
	}

	return strings.Join(kept, "\n")
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

// TestRunSaysWhatReleaseSays holds the command line to the same golden the
// library entry point is held to.
//
// The two entry points share everything below the first line of each — one
// assembly, one order of decorators, one reporter — but sharing code is not the
// claim worth pinning. The claim is that a caller gets the same verdicts either
// way, and the only way to know that is to read what each one printed.
//
// What differs is the framing, not the report: a test binary ends its output
// with its own PASS, and a command does not. That single line is removed here
// and nothing else is normalised, so a report that drifted by one byte still
// fails.
func TestRunSaysWhatReleaseSays(t *testing.T) {
	if testing.Short() {
		t.Skip("runs a full release: one test process per mutant")
	}

	moduleRoot := moduleRoot(t)
	project := t.TempDir()

	copyTree(t, filepath.Join(moduleRoot, "testdata", "goldenproject"), project)
	writeGoMod(t, project, moduleRoot)

	tidy := command(t, project, "go", "mod", "tidy")

	output, err := tidy.CombinedOutput()
	if err != nil {
		t.Fatalf("resolving the fixture's dependencies: %v\n%s", err, output)
	}

	// Built outside the fixture on purpose: a binary is not Go source and would
	// not be mutated, but a command built into the tree it measures is a habit
	// that stops being harmless the moment somebody builds a `.go` file there.
	binary := filepath.Join(t.TempDir(), "ditto"+binarySuffix())
	build := command(t, moduleRoot, "go", "build", "-o", binary, "./cmd/ditto")

	output, err = build.CombinedOutput()
	if err != nil {
		t.Fatalf("building the command: %v\n%s", err, output)
	}

	run := command(t, project, binary,
		"run",
		"--root", project,
		"--test-command", "go test -count=1 ./calc",
		"--threshold", "0",
	)

	output, err = run.CombinedOutput()
	if err != nil {
		t.Fatalf("running the command: %v\n%s", err, output)
	}

	got := strings.ReplaceAll(string(output), "\r\n", "\n")
	want := strings.TrimSuffix(readFile(t, filepath.Join(moduleRoot, "testdata", "golden", "release.txt")), "PASS\n")

	if got != want {
		t.Fatalf("the command said something different from the release.\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}

func binarySuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}

	return ""
}

// TestRunReturnsTheRefusalInsteadOfPanicking is the difference between the two
// entry points that a caller can actually feel.
//
// A red baseline stops a release: every mutant would be scored killed and the
// report would say 1.00 for a run that tested nothing, so the laboratory panics
// rather than score it. Inside a test binary that panic is the verdict and reads
// correctly. From a command it does not — a process that panics prints a stack
// and looks like a defect, and the one thing a gate must never do is make its
// own refusal indistinguishable from being broken.
func TestRunReturnsTheRefusalInsteadOfPanicking(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the test command once")
	}

	project := t.TempDir()
	writeFile(t, filepath.Join(project, "go.mod"), "module redbaseline\n\ngo 1.27\n")
	writeFile(t, filepath.Join(project, "calc.go"), "package redbaseline\n\nfunc Add(a, b int) int { return a + b }\n")
	writeFile(t, filepath.Join(project, "calc_test.go"),
		"package redbaseline\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) { t.Fatal(\"red on purpose\") }\n")

	err := ditto.Run(
		ditto.WithRepositoryRoot(project),
		ditto.WithTestCommand("go test -count=1 ./..."),
		ditto.WithMinimumThreshold(0.0),
	)
	if err == nil {
		t.Fatal("a red baseline was scored instead of refused")
	}

	if !strings.Contains(err.Error(), "refusing to score against a red baseline") {
		t.Fatalf("the error does not name the red baseline: %v", err)
	}
}

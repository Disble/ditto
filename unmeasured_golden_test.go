package ditto_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestUnmeasuredScopeGolden pins what a release says — and what it exits with —
// when the configured test command cannot compile part of the scope it was
// pointed at.
//
// This is the report's own case, in a fixture that holds all three classes at
// once. Measured on the repository that reported it: 43 mutants, 23 killed, 20
// survived, and 17 of those survivors in one package the command never builds,
// printed as a score of 0.53 against a bar of 0.80 over a run whose ceiling was
// 0.605. docs/reports/ditto-mutation-scope.md.
//
// The fixture's middle package is the load-bearing one. `shared` is not named by
// the command either, and it is NOT unmeasured: the named package's tests import
// it, so its mutants are compiled into the binary that runs and the command can
// kill them. A check that compared names instead of reading the toolchain's
// closure would fail this test, which is why it is here rather than only in the
// unit tests of internal/commandscope.
func TestUnmeasuredScopeGolden(t *testing.T) {
	if testing.Short() {
		t.Skip("runs a full release: one test process per mutant")
	}

	moduleRoot := moduleRoot(t)
	project := t.TempDir()

	copyTree(t, filepath.Join(moduleRoot, "testdata", "unmeasuredproject"), project)

	// The fixture depends on nothing, not even on ditto: it is the tree a run is
	// pointed at, and the run under test is the shipped command.
	writeFile(t, filepath.Join(project, "go.mod"), "module unmeasuredproject\n\ngo 1.27\n")

	binary := filepath.Join(t.TempDir(), "ditto"+binarySuffix())
	build := command(t, moduleRoot, "go", "build", "-o", binary, "./cmd/ditto")

	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building the command: %v\n%s", err, output)
	}

	output, exitCode := runOutputAndExit(t, project, binary,
		"run", "--root", project,
		"--test-command", "go test -count=1 -json ./covered/",
		"--threshold", "0",
	)

	got := withoutClocks(output)
	golden := filepath.Join(moduleRoot, "testdata", "golden", "unmeasured.txt")

	if os.Getenv("DITTO_GOLDEN_UPDATE") == "1" {
		writeFile(t, golden, got)
		t.Log("golden updated; rerun without DITTO_GOLDEN_UPDATE to check it")

		return
	}

	want := readFile(t, golden)
	if got != want {
		t.Fatalf("the release said something different.\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}

	// The exit code is the half of this that a wrapper reads. A scope ditto could
	// not measure and a score below the bar were both 1 before this, and the
	// responses differ: the second is answered by testing more, and the first by
	// naming every package the scope mutates or by narrowing the scope.
	if exitCode != 3 {
		t.Fatalf("the run exited %d, want 3 for a scope the command cannot measure", exitCode)
	}

	// The control, and it is not decoration: the same fixture, the same mutants,
	// one command that names every package. Nothing is unmeasured, the run
	// passes, and the `shared` mutant is judged rather than set aside — so the
	// failure above is about the command's scope and not about the fixture.
	control, controlExit := runOutputAndExit(t, project, binary,
		"run", "--root", project,
		"--test-command", "go test -count=1 -json ./...",
		"--threshold", "0",
	)

	if controlExit != 0 {
		t.Fatalf("the control exited %d, want 0:\n%s", controlExit, control)
	}

	for _, quiet := range []string{"Unmeasured", "never compiled by this test command"} {
		if strings.Contains(control, quiet) {
			t.Fatalf("the control reported an unmeasurable scope (%q):\n%s", quiet, control)
		}
	}

	if !strings.Contains(control, "shared/shared.go") {
		t.Fatalf("the control never mentioned the package the narrow command also reaches:\n%s", control)
	}
}

// runOutputAndExit runs the shipped command and returns its output and its exit
// code, because one of them is the subject and the other is the evidence.
func runOutputAndExit(t *testing.T, dir, binary string, args ...string) (string, int) {
	t.Helper()

	run := command(t, dir, binary, args...)

	output, err := run.CombinedOutput()
	normalised := strings.ReplaceAll(string(output), "\r\n", "\n")

	if err == nil {
		return normalised, 0
	}

	var failed *exec.ExitError
	if !errors.As(err, &failed) {
		t.Fatalf("running the command: %v\n%s", err, output)
	}

	return normalised, failed.ExitCode()
}

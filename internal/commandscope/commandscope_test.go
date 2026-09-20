package commandscope_test

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Disble/ditto/internal/commandscope"
)

// scriptedToolchain is the toolchain's answer, pinned. It also counts how often
// it was asked, because "one process per release" is part of what the scope
// costs and a test is the only place that can hold it.
type scriptedToolchain struct {
	output  []byte
	err     error
	asked   int
	lastRun []string
}

func (s *scriptedToolchain) Output(_, name string, args ...string) ([]byte, error) {
	s.asked++

	s.lastRun = append([]string{name}, args...)

	return s.output, s.err
}

// streamOf is the smallest `go test -json` stream that names packages: the
// scope reads the names, and nothing else in an event is its business.
func streamOf(packages ...string) string {
	lines := make([]string, 0, len(packages))
	for _, name := range packages {
		lines = append(lines, `{"Action":"start","Package":"`+name+`"}`)
	}

	return strings.Join(lines, "\n") + "\n"
}

// TestExecutesReadsTheClosureAndNotJustTheNames is the case that decides whether
// the check is usable at all.
//
// A mutant in a dependency of a package the command names IS killable by that
// package's tests — the dependency is compiled into its test binary — so a set
// comparison against the names would accuse a run that was correct. Measured on
// the report's own change: the command named syncdiag and the change also touched
// packages it imports.
func TestExecutesReadsTheClosureAndNotJustTheNames(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	toolchain := &scriptedToolchain{output: []byte(strings.Join([]string{
		filepath.Join(root, "internal", "observability", "readcap"),
		filepath.Join(root, "internal", "observability", "syncdiag"),
		filepath.Join(root, "testdata", "ignored"),
		"/usr/local/go/src/fmt",
		"",
	}, "\n"))}

	scope := commandscope.New(streamOf("example.com/mod/internal/observability/syncdiag"), root, toolchain)

	named := "internal/observability/syncdiag/reader.go"
	if !scope.Executes(named) {
		t.Fatalf("the package the command names is not reported as executed: %q", named)
	}

	inClosure := "internal/observability/readcap/reader.go"
	if !scope.Executes(inClosure) {
		t.Fatalf("a package compiled into the named package's test binary is not reported as executed: %q", inClosure)
	}

	// The report's own case: staged, mutated, and never built by the command.
	unreachable := "internal/desktop/app_runtime_services.go"
	if scope.Executes(unreachable) {
		t.Fatalf("a package the command does not compile is reported as executed: %q", unreachable)
	}

	// Something the closure named that is not in this tree cannot be a mutant,
	// and the answer for one is not what clears the answer for the rest.
	if scope.Executes("fmt/print.go") {
		t.Fatal("a directory outside the tree was read as executed")
	}
}

// TestExecutesAsksOncePerScope holds the cost: one process per release, however
// many files ask. It is the difference between a check that is free in the
// common case and one that pays 0.28s per file.
func TestExecutesAsksOncePerScope(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	toolchain := &scriptedToolchain{output: []byte(filepath.Join(root, "calc"))}

	scope := commandscope.New(streamOf("example.com/mod/calc"), root, toolchain)

	for i := range 5 {
		scope.Executes("calc/calc.go")
		scope.Executes("other/other.go")

		if toolchain.asked != 1 {
			t.Fatalf("the toolchain was asked %d times after %d rounds; one process per release is the cost", toolchain.asked, i+1)
		}
	}

	// And the query has to be the closure query. -deps is what reaches a
	// dependency, and -test is what reaches one that only a test file imports —
	// the case module_scope.go already measured for the gated path.
	for _, flag := range []string{"-deps", "-test"} {
		if !slices.Contains(toolchain.lastRun, flag) {
			t.Fatalf("the toolchain was asked without %s: %v", flag, toolchain.lastRun)
		}
	}
}

// TestExecutesRefusesNothingWhenTheCommandToldItNothing is the fail-open rule,
// and it is the answer for every command that is not `go test -json`: make,
// gotestsum, a wrapper. A missing accusation leaves a number somebody chose an
// opaque command to get; a wrong one fails a run that was correct.
func TestExecutesRefusesNothingWhenTheCommandToldItNothing(t *testing.T) {
	t.Parallel()

	toolchain := &scriptedToolchain{output: []byte(filepath.Join(t.TempDir(), "calc"))}

	for _, stream := range []string{"", "ok  \texample.com/mod/calc\t0.2s\n", "make: Nothing to be done.\n"} {
		scope := commandscope.New(stream, t.TempDir(), toolchain)

		if !scope.Executes("internal/desktop/app.go") {
			t.Fatalf("a command that named no package refused one: %q", stream)
		}
	}

	if toolchain.asked != 0 {
		t.Fatalf("the toolchain was asked %d times for commands ditto cannot read; the check costs nothing there", toolchain.asked)
	}
}

// TestExecutesRefusesNothingWhenTheToolchainCannotAnswer is the other half of
// fail-open: a module the toolchain cannot read, no `go` binary, a query that
// fails. Unknown is not evidence of a mismatch.
func TestExecutesRefusesNothingWhenTheToolchainCannotAnswer(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		toolchain *scriptedToolchain
	}{
		{name: "the query failed", toolchain: &scriptedToolchain{err: errors.New("no go binary")}},
		{name: "the query answered nothing", toolchain: &scriptedToolchain{output: []byte("  \n")}},
	}

	for _, testcase := range cases {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			scope := commandscope.New(streamOf("example.com/mod/calc"), t.TempDir(), testcase.toolchain)

			if !scope.Executes("internal/desktop/app.go") {
				t.Fatal("a scope that could not be resolved refused a mutant")
			}
		})
	}
}

// The inherited git addressing is removed from every process ditto starts, and
// this is the guard for it: cmdtestrunner has the version that measures the
// damage, and the incident behind both is in AGENTS.md — a stray commit on a
// live branch, from a command that was meant for a temporary directory.
func TestOSRunnerStripsTheInheritedGitAddressing(t *testing.T) {
	decoy := t.TempDir()

	for variable, value := range map[string]string{
		"GIT_DIR":              filepath.Join(decoy, ".git"),
		"GIT_INDEX_FILE":       filepath.Join(decoy, ".git", "index"),
		"GIT_WORK_TREE":        decoy,
		"GIT_OBJECT_DIRECTORY": filepath.Join(decoy, "objects"),
		"GIT_COMMON_DIR":       filepath.Join(decoy, "common"),
	} {
		t.Setenv(variable, value)
	}

	output, err := commandscope.OSRunner{}.Output(t.TempDir(), "sh", "-c",
		`printf '%s|%s|%s|%s|%s' "$GIT_DIR" "$GIT_INDEX_FILE" "$GIT_WORK_TREE" "$GIT_OBJECT_DIRECTORY" "$GIT_COMMON_DIR"`)
	if err != nil {
		t.Fatalf("running the probe: %v", err)
	}

	if got := strings.TrimSpace(string(output)); got != "||||" {
		t.Fatalf("the subprocess inherited git's addressing: %q", got)
	}
}

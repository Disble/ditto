// Package commandscope answers which packages one test command executes.
//
// Ditto scopes mutants to a change and judges them with a single configured test
// command, and those are two different questions. A staged change that spans
// five packages, judged by a command that names one of them, produces mutants in
// packages the command never builds: nothing it runs can kill them, so they
// survive, and the score that comes back mixes "your tests missed this" with
// "your command cannot see this". Measured on the report that produced this
// package: 43 mutants, 23 killed, 20 survived, and 17 of those survivors in a
// package the command never executes — a run whose ceiling was 0.605, printed as
// 0.53 against a bar of 0.80. docs/reports/ditto-mutation-scope.md.
//
// Nothing here parses the command. Ditto asks the toolchain instead, twice:
//
//   - `go test -json` names every package it executes, and that stream is
//     already in hand: the baseline run every release makes once produces it.
//   - `go list -deps -test` names every package those compile in, which is the
//     side that decides whether a mutant could be killed at all. A mutant in a
//     dependency of a named package IS killable by that package's tests, so
//     naming the package is too narrow an answer; `-test` is what makes a
//     test-only import count.
//
// A command that emits no such stream — make, gotestsum, a wrapper script — is
// not guessed at: its scope is unknown, and unknown refuses nothing. A missing
// accusation costs a number somebody configured an opaque command to get; a
// wrong one fails a run that was correct, which is the direction this repository
// does not fail in.
package commandscope

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/Disble/ditto/internal/gotoolchain"
)

// Runner runs one process, so the toolchain call can be pinned in a test.
type Runner interface {
	Output(dir, name string, args ...string) ([]byte, error)
}

// Scope is what one test command compiles, read from the stream it produced.
type Scope struct {
	root     string
	executed []string
	runner   Runner

	mutex    sync.Mutex
	resolved bool
	compiled map[string]bool
}

// New reads the packages a `go test -json` stream named, and answers the
// toolchain's own question about the rest when it is asked.
//
// root is the directory the command ran in: the mutated tree, not the checkout
// it came from. Every answer is relative to it.
func New(output, root string, runner Runner) *Scope {
	return &Scope{
		root:     root,
		executed: executedPackages(output),
		runner:   runner,
	}
}

// Executes reports whether the command compiles the package that owns a
// repository-relative source path.
//
// The answer is per package and not per file. A file the package does not
// compile on this platform — one behind a build tag for another operating system
// — is inside a package the command executes, and its mutants stay survivors.
// That is the boundary of this check: it separates "the command never builds
// this code" from "nothing ran this code", and only the first is a scope
// mismatch.
func (s *Scope) Executes(relativePath string) bool {
	if len(s.executed) == 0 {
		// The command told us nothing: no `-json`, no stream, or a command that
		// is not `go test` at all.
		return true
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.resolved {
		s.resolved = true

		s.compiled = s.compiledDirectories()
	}

	if s.compiled == nil {
		// The toolchain could not answer — no `go` binary, a module it cannot
		// read. Unknown again, and unknown refuses nothing.
		return true
	}

	return s.compiled[normalizePath(path.Dir(relativePath), runtime.GOOS)]
}

// compiledDirectories asks the toolchain which directories the executed packages
// compile into a test binary.
//
// One process per release, and only for a command that emitted a stream:
// measured at 0.28-0.31s over this repository's 64 executed packages, against a
// release that pays one suite run per mutant. It runs after the baseline, which
// has already built the same tree, so the module graph and the build cache are
// warm.
func (s *Scope) compiledDirectories() map[string]bool {
	binary := gotoolchain.Path()
	if binary == "" {
		return nil
	}

	arguments := append([]string{"list", "-deps", "-test", "-e", "-f", "{{.Dir}}"}, s.executed...)

	output, err := s.runner.Output(s.root, binary, arguments...)
	if err != nil {
		return nil
	}

	if len(bytes.TrimSpace(output)) == 0 {
		// A toolchain that answered nothing has not answered. An empty set would
		// be read as "this command compiles nothing", which accuses every mutant
		// in the scope.
		return nil
	}

	directories := make(map[string]bool)

	for line := range strings.SplitSeq(string(output), "\n") {
		relative := relativeDirectory(s.root, strings.TrimSpace(line), runtime.GOOS)
		if relative == "" {
			continue
		}

		directories[normalizePath(relative, runtime.GOOS)] = true
	}

	return directories
}

// event is the part of `go test -json` this reads. The rest is ignored on
// purpose: a field ditto does not read is a field that cannot make it wrong.
// Same shape and same reason as the struct in internal/verdict.
type event struct {
	Package string `json:"Package"` //nolint:tagliatelle // go test -json emits these names
	Action  string `json:"Action"`  //nolint:tagliatelle // go test -json emits these names
}

// executedPackages reads the packages a stream named, in the order it named
// them.
//
// A stream names every package the command executes, and a package with no test
// files is one of them: measured on go1.27, `go test -json ./...` emits a start,
// an output line reading `[no test files]` and a skip for it. What it does not
// name is that package's dependencies, which never have a test binary of their
// own — which is why the closure below exists rather than a set comparison.
func executedPackages(output string) []string {
	seen := make(map[string]bool)
	names := []string{}

	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			// Compiler output and prose share the stream. A line that is not an
			// event is not an error, and refusing the scope over one would throw
			// away an answer a reader is about to act on.
			continue
		}

		var decoded event
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			continue
		}

		if decoded.Package == "" || decoded.Action == "" || seen[decoded.Package] {
			continue
		}

		seen[decoded.Package] = true
		names = append(names, decoded.Package)
	}

	return names
}

// relativeDirectory turns one absolute directory from the toolchain into a
// repository-relative one, and answers empty for everything outside the root:
// the closure carries the standard library and every dependency, and none of
// those can hold a mutant.
func relativeDirectory(root, directory, goos string) string {
	if directory == "" {
		return ""
	}

	normalizedRoot := normalizePath(root, goos)
	normalizedDirectory := normalizePath(directory, goos)

	if normalizedDirectory == normalizedRoot {
		return "."
	}

	// The separator is part of the prefix, so /treehouse is not under /tree.
	prefix := normalizedRoot + "/"
	if !strings.HasPrefix(normalizedDirectory, prefix) {
		return ""
	}

	return normalizedDirectory[len(prefix):]
}

// normalizePath is the form two directories are compared in, and the identity
// one package is filed under.
//
// Two things are folded here and both of them have produced a defect in this
// repository already. Separators: a path from the toolchain and a path ditto
// built are the same directory spelled two ways. Case: Windows compares paths
// case-insensitively, so the fold is not optional there — it is the difference
// between clearing a mutant and wrongfully accusing it, and a wrong accusation
// fails a run that was correct.
//
// The system is a parameter rather than a read of runtime.GOOS, the way the
// target OS is in the batch planner's binary name: the folding half only exists
// on Windows, CI runs on Linux, and a test that passes "windows" is the only way
// this host covers it.
func normalizePath(value, goos string) string {
	slashed := path.Clean(strings.ReplaceAll(value, `\`, "/"))
	if goos == windows {
		return strings.ToLower(slashed)
	}

	return slashed
}

// windows is the one GOOS whose paths fold case, named so the fold can be
// exercised where it does not run.
const windows = "windows"

// gitEnvironment is what git exports to a hook, and in a linked worktree it
// exports them as absolute paths. Everything spawned below inherits them, so a
// command meant for the sandbox addresses the hook's repository instead — and
// then succeeds, which is the whole problem.
//
// Measured twice in this repository before the list existed: once leaving a
// stray commit on a live branch, once writing core.bare and an identity into a
// shared config. The toolchain does not read git here, and this list is not an
// argument that it might: it is the rule every process ditto starts follows.
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
	command := exec.Command(name, args...) //nolint:noctx // there is no cancellation contract here: the query is bounded by go list's own work
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
		if slices.Contains(gitEnvironment, name) {
			continue
		}

		kept = append(kept, variable)
	}

	return kept
}

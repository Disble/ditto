// Package gobuildrunner runs a package's tests from a binary it builds itself,
// once, instead of invoking `go test` for every mutant.
//
// This is the capability ditto did not have. It knew a command string and ran
// it, so every mutant paid the fixed cost of starting `go test` — measured at
// 750-950 ms regardless of what the suite does, and 84% of a run. Compiling once
// only helps when the compiled thing stops changing between mutants, which is
// what internal/schemata is for; the two are useless apart.
package gobuildrunner

import (
	"errors"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/Disble/ditto/internal/cmdtestrunner"
	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/result"
)

// Selected is the environment variable a gated file reads to know which mutant
// it is being asked for. Zero, or absent, selects none.
const Selected = "DITTO_MUTANT"

// errNoToolchain is what a build reports when no `go` binary could be resolved.
var errNoToolchain = errors.New("ditto: no go toolchain found")

// GoBuildRunner builds the tests of one package and then runs that binary.
//
// One package, because `go test -c` compiles one package's tests. That is the
// shape of the case this is for: a change staged in one package, run while the
// code is still being written.
type GoBuildRunner struct {
	packagePath string
	mutant      int

	toolchain    string
	binary       string
	compilations int
	runs         int
}

func New(packagePath string) *GoBuildRunner {
	return &GoBuildRunner{packagePath: packagePath, toolchain: goToolchain()}
}

// Toolchain is the absolute path of the `go` binary this runner builds with, or
// empty when none could be found.
func (r *GoBuildRunner) Toolchain() string { return r.toolchain }

// goToolchain resolves the compiler once, to an absolute path, instead of naming
// it and letting the operating system search.
//
// Two reasons, pointing the same way. `exec.Command("go", …)` reads PATH at
// every call, so a directory an attacker can write to — or prepend — decides
// which compiler runs. That is SonarQube's go:S4036, and it is the same shape as
// git's inherited addressing that `environment` below strips: something ambient
// deciding what a subprocess really is.
//
// The other reason matters more here. Ditto exists to compare verdicts, and
// building a mutant's tests with a different toolchain from the one running the
// suite would make a disagreement that is nobody's mutation look like one that
// is. GOROOT is the toolchain that built this binary, so it is preferred, and
// PATH is the fallback for a GOROOT that is not on disk.
//
// It resolves to empty rather than panicking. A build that cannot happen is an
// answer the caller already knows how to take: Built stays false and the file
// falls back to the path ditto has always taken.
func goToolchain() string {
	name := "go"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}

	if root := build.Default.GOROOT; root != "" {
		candidate := filepath.Join(root, "bin", name)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}

	found, err := exec.LookPath("go")
	if err != nil {
		return ""
	}

	return found
}

// Select is which mutant the next run asks the binary for.
func (r *GoBuildRunner) Select(mutant int) { r.mutant = mutant }

// Compilations and Runs are exact counters, which is what the performance of
// this is allowed to be judged on. Wall clock varies by more than half on a
// working machine; these do not vary at all.
func (r *GoBuildRunner) Compilations() int { return r.compilations }
func (r *GoBuildRunner) Runs() int         { return r.runs }

// Built reports whether there is a binary to run. A package that does not
// compile leaves this false, and the caller falls back rather than being killed
// by a build it shares with every other mutant.
func (r *GoBuildRunner) Built() bool { return r.binary != "" }

func (r *GoBuildRunner) Test(repository ditto.TemporaryRepository) result.Result[string] {
	if r.binary == "" {
		if output, err := r.build(repository.Root()); err != nil {
			// A build failure is an answer, not a panic. Today ditto reads a
			// failing command as a killed mutant, which is how an uncompilable
			// mutation is scored as caught by a test that never ran; the caller
			// decides what this means, with Built to tell it apart.
			return result.Ok(output)
		}
	}

	r.runs++

	return r.run(repository.Root())
}

func (r *GoBuildRunner) build(root string) (string, error) {
	r.compilations++

	if r.toolchain == "" {
		return "ditto: no go toolchain found to build with", errNoToolchain
	}

	binary := filepath.Join(root, "ditto.test")

	// noctx wants CommandContext. The build has no cancellation contract today,
	// for the same reason the test command has none: giving it one is an API
	// decision rather than a lint fix. Recorded in docs/backlog.md.
	command := exec.Command(r.toolchain, "test", "-c", "-o", binary, r.packagePath) //nolint:noctx,gosec // resolved to an absolute path in goToolchain
	command.Dir = root
	command.Env = environment(r.mutant)

	output, err := command.CombinedOutput()
	if err != nil {
		return string(output), err
	}

	r.binary = binary

	return "", nil
}

// run works from the package's own directory, because `go test` does and a suite
// that reads anything by relative path depends on it. Running from the module
// root instead made an untouched package fail, which read exactly like the
// instrumentation having broken it.
func (r *GoBuildRunner) run(root string) result.Result[string] {
	// -test.timeout is passed because a test binary invoked directly takes 0 --
	// timeout disabled -- and only the `go test` driver injects the 10 minute
	// default. Without it a mutant that loops never returns and the release
	// never ends; loopcondition, loopbreak and rangebreak are all in the default
	// virus set. docs/backlog.md entry 15.
	command := exec.Command(r.binary, "-test.count=1", //nolint:gosec,noctx // the binary is the one just built
		"-test.timeout="+cmdtestrunner.DefaultDeadline.String())
	command.Dir = filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(r.packagePath, "./")))
	command.Env = environment(r.mutant)

	output, err := command.CombinedOutput()
	if err != nil {
		return result.Ok(string(output))
	}

	return result.Err[string](string(output))
}

// environment carries the selection and drops git's inherited addressing, for
// the reason internal/cmdtestrunner spells out: git exports GIT_DIR to hooks as
// an absolute path in a linked worktree, and a command meant for the sandbox
// reaches the real checkout instead — and succeeds.
func environment(mutant int) []string {
	inherited := []string{
		"GIT_DIR=", "GIT_INDEX_FILE=", "GIT_WORK_TREE=",
		"GIT_OBJECT_DIRECTORY=", "GIT_COMMON_DIR=", Selected + "=",
	}

	kept := make([]string, 0, len(os.Environ())+1)

	for _, variable := range os.Environ() {
		if !hasAnyPrefix(variable, inherited) {
			kept = append(kept, variable)
		}
	}

	return append(kept, Selected+"="+strconv.Itoa(mutant))
}

func hasAnyPrefix(value string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}

	return false
}

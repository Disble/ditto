// Package gotoolchain resolves the Go toolchain once, to an absolute path.
//
// Two packages ask it, and they have to agree. internal/gobuildrunner compiles a
// mutant's tests with it; internal/commandscope asks it which packages those
// tests compile. A scope resolved by one toolchain and a build made by another
// would compare answers from two different compilers, and the disagreement would
// look like a property of the module under test.
//
// Resolving is not tidiness. `exec.Command("go", …)` reads PATH at every call,
// so a directory an attacker can write to — or prepend — decides which compiler
// runs: SonarQube's go:S4036, CWE-426. GOROOT is the toolchain that built this
// binary, so it is preferred; PATH is the fallback for a GOROOT that is not on
// disk.
//
// It answers empty rather than failing. A caller that cannot build already knows
// what to do with that — `gobuildrunner` reports that the binary was never
// built, and a scope that cannot be resolved refuses nothing — and a panic here
// would turn a missing toolchain into a defect that looks like ditto's.
package gotoolchain

import (
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Path is the absolute path of the `go` binary to run, or empty when none could
// be found.
func Path() string {
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

package gotoolchain_test

import (
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Disble/ditto/internal/gotoolchain"
)

// TestPathPrefersGOROOT covers the resolution CI and a local run both depend on:
// the toolchain that built this binary is the one that compiles a mutant's
// tests. It is also the half that keeps PATH — writable by whoever can prepend a
// directory to it — from deciding which compiler runs (go:S4036, CWE-426).
func TestPathPrefersGOROOT(t *testing.T) {
	root := t.TempDir()

	// Both names, because the lookup appends .exe on Windows and does not
	// anywhere else: a test that created one of them would be silent about the
	// other platform, and this repository's worst bugs have been platform-shaped.
	writeFile(t, filepath.Join(root, "bin", "go"))
	writeFile(t, filepath.Join(root, "bin", "go.exe"))

	restore := setGOROOT(t, root)
	defer restore()

	got := gotoolchain.Path()
	if !strings.HasPrefix(got, filepath.Clean(root)) {
		t.Fatalf("Path() = %q, which is not inside the GOROOT it was given (%q)", got, root)
	}
}

// TestPathRefusesADirectoryNamedGo is the control for the guard inside the
// GOROOT branch: a path that exists but is not a file is not a toolchain. A
// directory there would otherwise be handed to exec.Command and fail at every
// call, which reads as a broken module rather than a broken GOROOT.
func TestPathRefusesADirectoryNamedGo(t *testing.T) {
	root := t.TempDir()

	for _, name := range []string{"go", "go.exe"} {
		if err := os.MkdirAll(filepath.Join(root, "bin", name), 0o750); err != nil {
			t.Fatalf("creating the decoy directory: %v", err)
		}
	}

	restore := setGOROOT(t, root)
	defer restore()

	got := gotoolchain.Path()
	if got == "" {
		t.Skip("no go binary on PATH to fall back to")
	}

	if strings.HasPrefix(got, filepath.Clean(root)) {
		t.Fatalf("Path() = %q, which is the directory it must refuse", got)
	}
}

// TestPathFallsBackToPath is the other half of the contract, and it is asserted
// against LookPath rather than against a literal: whatever PATH resolves `go` to
// is what a caller gets when GOROOT has no binary to offer.
func TestPathFallsBackToPath(t *testing.T) {
	restore := setGOROOT(t, filepath.Join(t.TempDir(), "no-toolchain-here"))
	defer restore()

	want, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go binary on PATH, so there is no fallback to assert")
	}

	if got := gotoolchain.Path(); got != want {
		t.Fatalf("Path() = %q, want the PATH answer %q", got, want)
	}
}

func writeFile(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(path), err)
	}

	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// setGOROOT points the resolution at a directory instead of the running
// toolchain's, and returns the restore. The value is asked for at every call, so
// nothing caches the real one.
func setGOROOT(t *testing.T, root string) func() {
	t.Helper()

	previous := build.Default.GOROOT
	build.Default.GOROOT = root

	return func() { build.Default.GOROOT = previous }
}

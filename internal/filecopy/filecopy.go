// Package filecopy puts one file somewhere else, as a real file.
//
// It is one function in its own package because two callers need exactly it and
// for the same reason. A sandbox is only a sandbox if what it holds is a copy:
// a symlink is a reference that `go:embed` refuses and a write follows through
// to the original, and a hard link shares the inode so a write reaches it too.
// See docs/experiments/the-sandbox-is-a-reference.md.
//
// fsrepository materialises the index this way, and staged fills in what git
// does not carry the same way. Written twice, the two would drift.
package filecopy

import (
	"fmt"
	"io/fs"
	"os"
)

// File puts source at destination as what it is: a copy for a regular file, and
// the same link again for a symlink.
//
// The entry is taken rather than looked up because both callers are already
// walking a tree and have it, and a stat per file is a cost this package exists
// inside the hot path of.
func File(source, destination string, entry fs.DirEntry) error {
	// WalkDir does not follow links, so one arrives here as a non-directory
	// entry. Read as a file, a link to a directory returns EISDIR and takes the
	// whole walk with it; a link to a file is followed silently and lands as a
	// regular file the tree never had. Neither is what was asked for.
	if entry.Type()&fs.ModeSymlink != 0 {
		return link(source, destination)
	}

	info, err := entry.Info()
	if err != nil {
		return fmt.Errorf("reading '%s': %w", source, err)
	}

	data, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("reading '%s': %w", source, err)
	}

	// G703 traces the destination back to that walk. Both ends are named by the
	// caller — the root it asked to mutate, and the temporary directory this run
	// created — and walking one into the other is what this is for.
	if err := os.WriteFile(destination, data, info.Mode().Perm()); err != nil { //nolint:gosec
		return fmt.Errorf("writing '%s': %w", destination, err)
	}

	return nil
}

// link reproduces one symlink, target and all.
//
// The raw target is the point. Rewritten to an absolute path it would resolve
// back to the tree being measured, and a suite writing through it would edit the
// repository ditto was asked to leave alone -- the write-through this package's
// doc comment exists to describe. Verbatim, a relative link resolves inside the
// sandbox, where the write lands on the copy.
//
// It is also what git does with a tracked symlink, so a full run and a staged
// run agree about what the repository is.
func link(source, destination string) error {
	target, err := os.Readlink(source)
	if err != nil {
		return fmt.Errorf("reading the link at '%s': %w", source, err)
	}

	// G122 wants a root-scoped API here. The target is the tree's own, kept
	// verbatim on purpose, and os.Root cannot express a link that leaves the
	// destination.
	if err := os.Symlink(target, destination); err != nil {
		return fmt.Errorf("recreating the link at '%s' to '%s': %w", destination, target, err)
	}

	return nil
}

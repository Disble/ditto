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

// File copies source to destination, keeping the source's permissions.
//
// The entry is taken rather than looked up because both callers are already
// walking a tree and have it, and a stat per file is a cost this package exists
// inside the hot path of.
func File(source, destination string, entry fs.DirEntry) error {
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

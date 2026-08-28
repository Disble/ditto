package fsrepository_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Disble/ditto/internal/fsrepository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// symlinkOrSkip makes one symlink, or skips: Windows refuses without developer
// mode, and a machine that cannot make one cannot be asked about them.
func symlinkOrSkip(t *testing.T, target, name string) {
	t.Helper()

	if err := os.Symlink(target, name); err != nil {
		t.Skipf("this machine cannot create symlinks: %v", err)
	}
}

func TestFSRepository_MaterializesSymlinks(t *testing.T) {
	// A symlink to a directory is reported by WalkDir as a non-directory entry,
	// because WalkDir does not follow links. Read as a file it returns EISDIR,
	// and a release over any repository holding one died before its first
	// mutant. See docs/experiments/a-symlink-in-the-tree.md.
	t.Run("a link to a directory does not stop the walk", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "target"), 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "target", "file.txt"), []byte("inside"), 0o600))
		symlinkOrSkip(t, filepath.Join(dir, "target"), filepath.Join(dir, "dirlink"))

		sandbox := t.TempDir()
		repository := fsrepository.NewWithStrategy(dir, "copy")

		assert.NotPanics(t, func() {
			repository.LinkAllToTemporaryRepository(sandbox)
		})
	})

	// The link, not what it points at. Following it would copy a nix profile or
	// a node_modules store into every sandbox, and it would disagree with
	// `ditto staged`, whose sandbox comes from git -- which materialises a
	// tracked symlink as a link and never as its target.
	t.Run("the link is recreated, not followed", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("original"), 0o600))
		symlinkOrSkip(t, "file.txt", filepath.Join(dir, "filelink"))

		sandbox := t.TempDir()
		repository := fsrepository.NewWithStrategy(dir, "copy")
		repository.LinkAllToTemporaryRepository(sandbox)

		target, err := os.Readlink(filepath.Join(sandbox, "filelink"))
		require.NoError(t, err, "the sandbox should hold a link")
		assert.Equal(t, "file.txt", target, "and it should carry the raw target")

		// Exact counts, because "it worked" is also what following the link
		// looked like: one file copied and one link reproduced, not two copies.
		assert.Equal(t, 1, repository.Relinked(), "links reproduced")
		assert.Equal(t, 1, repository.Copied(), "files copied")
	})

	// The raw target is what keeps a relative link inside the sandbox. Rewritten
	// to an absolute path it resolves back to the repository under measurement,
	// and a suite that writes through it edits the tree ditto was asked to
	// measure -- the defect docs/experiments/the-sandbox-is-a-reference.md exists
	// to close, reintroduced one path at a time.
	t.Run("writing through the link stays inside the sandbox", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("original"), 0o600))
		symlinkOrSkip(t, "file.txt", filepath.Join(dir, "filelink"))

		sandbox := t.TempDir()
		fsrepository.NewWithStrategy(dir, "copy").LinkAllToTemporaryRepository(sandbox)

		require.NoError(t, os.WriteFile(filepath.Join(sandbox, "filelink"), []byte("REWRITTEN BY THE SUITE"), 0o600))

		source, err := os.ReadFile(filepath.Join(dir, "file.txt"))
		require.NoError(t, err)
		assert.Equal(t, "original", string(source), "the repository under measurement must come back untouched")
	})
}

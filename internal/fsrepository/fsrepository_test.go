package fsrepository_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Disble/ditto/internal/fsrepository"
	"github.com/Disble/ditto/internal/gosourcefile"
	"github.com/stretchr/testify/assert"
)

func TestFSRepository(t *testing.T) {
	t.Run("panics when given root does not exist", func(t *testing.T) {
		assert.PanicsWithValue(t, "nonexistent: no such directory", func() {
			fsrepository.New("nonexistent")
		})
	})

	t.Run("panics when given root isn't a directory", func(t *testing.T) {
		dir := t.TempDir()
		assert.NoError(t, os.WriteFile(dir+"/not-a-dir", []byte("source data"), 0o600))

		assert.PanicsWithValue(t, dir+"/not-a-dir: not a directory", func() {
			fsrepository.New(dir + "/not-a-dir")
		})
	})
}

func TestFSRepository_ListGoSourceFiles(t *testing.T) {
	t.Run("empty source files", func(t *testing.T) {
		repository := fsrepository.New(t.TempDir())
		files := repository.ListGoSourceFiles()
		assert.Equal(t, []*gosourcefile.GoSourceFile{}, files)
	})

	t.Run("single source file", func(t *testing.T) {
		dir := t.TempDir()
		assert.NoError(t, os.WriteFile(dir+"/source.go", []byte("source data"), 0o600))

		repository := fsrepository.New(dir)
		files := repository.ListGoSourceFiles()
		assert.Equal(t, []*gosourcefile.GoSourceFile{
			gosourcefile.New("source.go", []byte("source data")),
		}, files)
	})

	t.Run("multiple source files", func(t *testing.T) {
		dir := t.TempDir()
		assert.NoError(t, os.WriteFile(dir+"/source1.go", []byte("source data 1"), 0o600))
		assert.NoError(t, os.WriteFile(dir+"/source2.go", []byte("source data 2"), 0o600))
		assert.NoError(t, os.WriteFile(dir+"/source3.go", []byte("source data 3"), 0o600))

		repository := fsrepository.New(dir)
		files := repository.ListGoSourceFiles()
		assert.Equal(t, []*gosourcefile.GoSourceFile{
			gosourcefile.New("source1.go", []byte("source data 1")),
			gosourcefile.New("source2.go", []byte("source data 2")),
			gosourcefile.New("source3.go", []byte("source data 3")),
		}, files)
	})

	t.Run("does not include non Go files", func(t *testing.T) {
		dir := t.TempDir()
		assert.NoError(t, os.WriteFile(dir+"/source1.go", []byte("source data 1"), 0o600))
		assert.NoError(t, os.WriteFile(dir+"/source2.rs", []byte("source data 2"), 0o600))

		repository := fsrepository.New(dir)
		files := repository.ListGoSourceFiles()
		assert.Equal(t, []*gosourcefile.GoSourceFile{
			gosourcefile.New("source1.go", []byte("source data 1")),
		}, files)
	})

	t.Run("does not include Go test files", func(t *testing.T) {
		dir := t.TempDir()
		assert.NoError(t, os.WriteFile(dir+"/source1.go", []byte("source data 1"), 0o600))
		assert.NoError(t, os.WriteFile(dir+"/source1_test.go", []byte("test data 1"), 0o600))

		repository := fsrepository.New(dir)
		files := repository.ListGoSourceFiles()
		assert.Equal(t, []*gosourcefile.GoSourceFile{
			gosourcefile.New("source1.go", []byte("source data 1")),
		}, files)
	})

	t.Run("recursive directories", func(t *testing.T) {
		dir := t.TempDir()
		assert.NoError(t, os.MkdirAll(dir+"/a/b", 0o700))
		assert.NoError(t, os.WriteFile(dir+"/source1.go", []byte("source data 1"), 0o600))
		assert.NoError(t, os.WriteFile(dir+"/a/source2.go", []byte("source data 2"), 0o600))
		assert.NoError(t, os.WriteFile(dir+"/a/b/source3.go", []byte("source data 3"), 0o600))

		repository := fsrepository.New(dir)
		files := repository.ListGoSourceFiles()
		assert.Equal(t, []*gosourcefile.GoSourceFile{
			gosourcefile.New("a/b/source3.go", []byte("source data 3")),
			gosourcefile.New("a/source2.go", []byte("source data 2")),
			gosourcefile.New("source1.go", []byte("source data 1")),
		}, files)
	})

	t.Run("relative root", func(t *testing.T) {
		dir := t.TempDir()
		assert.NoError(t, os.MkdirAll(dir+"/a/b", 0o700))

		assert.NoError(t, os.WriteFile(dir+"/readme.md", []byte("read me"), 0o600))
		assert.NoError(t, os.WriteFile(dir+"/source1.go", []byte("source data 1"), 0o600))
		assert.NoError(t, os.WriteFile(dir+"/source1_test.go", []byte("test data 1"), 0o600))
		assert.NoError(t, os.WriteFile(dir+"/a/source2.go", []byte("source data 2"), 0o600))
		assert.NoError(t, os.WriteFile(dir+"/a/source2_test.go", []byte("test data 2"), 0o600))
		assert.NoError(t, os.WriteFile(dir+"/a/b/source3.go", []byte("source data 3"), 0o600))
		assert.NoError(t, os.WriteFile(dir+"/a/b/source3_test.go", []byte("test data 3"), 0o600))

		repository := fsrepository.New(dir)
		files := repository.ListGoSourceFiles()
		assert.Equal(t, []*gosourcefile.GoSourceFile{
			gosourcefile.New("a/b/source3.go", []byte("source data 3")),
			gosourcefile.New("a/source2.go", []byte("source data 2")),
			gosourcefile.New("source1.go", []byte("source data 1")),
		}, files)
	})
}

func TestFSRepository_LinkAllToTemporaryRepository(t *testing.T) {
	dir := t.TempDir()
	assert.NoError(t, os.MkdirAll(dir+"/to-link/child_a/child_b", 0o700))

	assert.NoError(t, os.MkdirAll(dir+"/to-link/child_a/child_b", 0o700))
	assert.NoError(t, os.WriteFile(dir+"/to-link/readme.md", []byte(""), 0o600))
	assert.NoError(t, os.WriteFile(dir+"/to-link/makefile", []byte(""), 0o600))
	assert.NoError(t, os.WriteFile(dir+"/to-link/test_a.go", []byte(""), 0o600))
	assert.NoError(t, os.WriteFile(dir+"/to-link/test_b.go", []byte(""), 0o600))
	assert.NoError(t, os.WriteFile(dir+"/to-link/child_a/test_c.go", []byte(""), 0o600))
	assert.NoError(t, os.WriteFile(dir+"/to-link/child_a/child_b/test_d.go", []byte(""), 0o600))

	repository := fsrepository.New(dir + "/to-link")
	temporaryRepository := repository.LinkAllToTemporaryRepository(dir + "/linked")

	t.Run("creates a link of all files recursively", func(t *testing.T) {
		var files []string

		err := filepath.WalkDir(dir+"/linked", func(path string, entry fs.DirEntry, err error) error {
			assert.NoError(t, err)

			if entry.IsDir() {
				return nil
			}

			info, err := entry.Info()
			assert.NoError(t, err)

			// A regular file, and this is the assertion the sandbox exists for
			// rather than a note about how it is built. Go refuses to embed an
			// irregular file, so a sandbox of symlinks cannot build any package
			// carrying an embed directive — measured on a real repository, where
			// it reported a perfect score for nine mutants that never compiled.
			// A hard link is a second name for the same bytes and satisfies this;
			// a symlink does not.
			assert.True(t, info.Mode().IsRegular(),
				"%s is not a regular file, so go:embed would refuse it", path)

			files = append(files, path)

			return nil
		})
		assert.NoError(t, err)
		// WalkDir yields host-native paths, and LinkAllToTemporaryRepository
		// builds them with filepath.Join, so the expectation has to be built
		// the same way rather than with hardcoded forward slashes.
		assert.Equal(t, []string{
			filepath.Join(dir, "linked", "child_a", "child_b", "test_d.go"),
			filepath.Join(dir, "linked", "child_a", "test_c.go"),
			filepath.Join(dir, "linked", "makefile"),
			filepath.Join(dir, "linked", "readme.md"),
			filepath.Join(dir, "linked", "test_a.go"),
			filepath.Join(dir, "linked", "test_b.go"),
		}, files)
	})

	t.Run("results in a new temporary repository", func(t *testing.T) {
		assert.Equal(t, fsrepository.NewTemporary(dir+"/linked"), temporaryRepository)
	})
}

// TestASandboxWriteDoesNotReachTheSource is what makes a sandbox a sandbox.
//
// A suite writes files. A golden it decides to update, a fixture it rewrites, a
// cache it drops beside the source — all ordinary, and all of them land on the
// repository being measured if the sandbox is a reference to it rather than a
// copy of it. A symlink is written THROUGH to its target; a hard link shares the
// inode, so it is written through too.
//
// Measured before this existed, on a fixture whose test rewrites one tracked
// file: under links and hard links the source came back holding
// `REWRITTEN BY THE SUITE`, and under copies it came back untouched.
//
// ditto own mutant write is safe under all three, because Overwrite removes the
// path first. That is why nobody noticed: the only writer anyone checked was
// ditto.
func TestASandboxWriteDoesNotReachTheSource(t *testing.T) {
	for _, strategy := range []string{"", "copy"} {
		t.Run("strategy "+strategy, func(t *testing.T) {
			dir := t.TempDir()
			source := dir + "/source"

			assert.NoError(t, os.MkdirAll(source, 0o700))
			assert.NoError(t, os.WriteFile(source+"/golden.txt", []byte("ORIGINAL"), 0o600))

			repository := fsrepository.NewWithStrategy(source, strategy)
			repository.LinkAllToTemporaryRepository(dir + "/sandbox")

			// Exactly what a suite does when it updates a golden: an ordinary
			// write to an ordinary path, with no idea it is inside a sandbox.
			assert.NoError(t, os.WriteFile(dir+"/sandbox/golden.txt", []byte("REWRITTEN BY THE SUITE"), 0o600))

			still, err := os.ReadFile(source + "/golden.txt")
			assert.NoError(t, err)
			assert.Equal(t, "ORIGINAL", string(still),
				"a write inside the sandbox reached the repository being measured")
		})
	}

	t.Logf("running on %s", runtime.GOOS)
}

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

// TestDefaultStrategyHardLinksWhereItCan reports which path a sandbox actually
// took, on whichever platform is running it.
//
// It exists because the fallback is silent and indistinguishable from success: a
// hard link and a copy are both regular files, so every other assertion in this
// package passes either way. Without this, "hard links work on Linux" would be a
// claim CI cannot confirm or deny — it would go green on copies just as happily.
//
// The counters are deliberately not in perf/baseline.json. Hard links cannot
// cross a filesystem, so the split depends on where the temporary directory
// lives, and that file gates on integers that are the same on every machine.
func TestDefaultStrategyHardLinksWhereItCan(t *testing.T) {
	dir := t.TempDir()

	assert.NoError(t, os.MkdirAll(dir+"/source/nested", 0o700))
	assert.NoError(t, os.WriteFile(dir+"/source/a.go", []byte("package a\n"), 0o600))
	assert.NoError(t, os.WriteFile(dir+"/source/nested/b.go", []byte("package b\n"), 0o600))

	repository := fsrepository.New(dir + "/source")
	repository.LinkAllToTemporaryRepository(dir + "/sandbox")

	linked, copied := repository.HardLinked(), repository.Copied()
	t.Logf("materialised %d file(s) by hard link and %d by copy on %s", linked, copied, runtime.GOOS)

	assert.Equal(t, 2, linked+copied, "every file reaches the sandbox one way or the other")

	// Source and sandbox are both under one t.TempDir, so they are on the same
	// filesystem and the cheap path is available. A copy here means hard links
	// are unavailable on this platform, which is worth failing over rather than
	// discovering later in a timing.
	assert.Equal(t, 0, copied,
		"the temporary directory shares a filesystem with the source, so nothing should have needed copying")
}

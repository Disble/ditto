package fsrepository

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/gosourcefile"
)

type FSRepository struct {
	root string
}

func New(root string) *FSRepository {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		panic(err)
	}

	stat, err := os.Stat(absRoot)
	if errors.Is(err, fs.ErrNotExist) {
		panic(root + ": no such directory")
	}

	if err != nil {
		panic(err)
	}

	if !stat.IsDir() {
		panic(root + ": not a directory")
	}

	return &FSRepository{
		root: absRoot,
	}
}

// repositoryMetadata is Git's own directory. Nothing in it is Go source, so it
// contributes no mutants, and both walks below pay for it on every pass —
// LinkAllToTemporaryRepository once per mutant. Measured on one checkout: 164
// working files against 1324 objects under .git, so nine tenths of the walk
// was for files that cannot be mutated.
//
// Skipping it is also the safer answer. The temporary repository is built from
// symlinks, one per file, so linking Git's own directory hands anything running
// in the sandbox a live handle on the real repository's objects, config and
// refs. A test that runs `git` in there could write through those links to the
// checkout ditto was pointed at.
const repositoryMetadata = ".git"

func skipRepositoryMetadata(entry fs.DirEntry) error {
	if entry.Name() == repositoryMetadata {
		return fs.SkipDir
	}

	return nil
}

func (r *FSRepository) ListGoSourceFiles() []*gosourcefile.GoSourceFile {
	var paths []string

	err := filepath.WalkDir(r.root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return skipRepositoryMetadata(entry)
		}

		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}

		paths = append(paths, path)

		return nil
	})
	if err != nil {
		panic(err)
	}

	sort.Strings(paths)

	sourceFiles := make([]*gosourcefile.GoSourceFile, len(paths))

	for i, path := range paths {
		data, _ := os.ReadFile(path)
		relativePath, _ := filepath.Rel(r.root, path)
		// This string is the file's identity: IgnoreSourceFiles matches
		// patterns against it and every report prints it. Leaving the host
		// separator in makes that identity platform-dependent, so a pattern
		// written with '/' matches nothing on Windows and the same run reports
		// different names on different machines.
		sourceFiles[i] = gosourcefile.New(filepath.ToSlash(relativePath), data)
	}

	return sourceFiles
}

func (r *FSRepository) LinkAllToTemporaryRepository(temporaryPath string) ditto.TemporaryRepository {
	rootSize := len(strings.Split(r.root, string(os.PathSeparator)))

	err := filepath.WalkDir(r.root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return skipRepositoryMetadata(entry)
		}

		absolutePath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("failed getting absolute path for '%s': %w", path, err)
		}

		linkPath := filepath.Join(temporaryPath, filepath.Join(strings.Split(path, string(os.PathSeparator))[rootSize:]...))
		err = os.MkdirAll(filepath.Dir(linkPath), os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed creating directory tree for '%s': %w", linkPath, err)
		}

		err = os.Symlink(absolutePath, linkPath)
		if err != nil {
			return fmt.Errorf("failed creating link from '%s' to '%s': %w", path, linkPath, err)
		}

		return nil
	})
	if err != nil {
		panic(fmt.Errorf("failed scanning '%s': %w", r.root, err))
	}

	return NewTemporary(temporaryPath)
}

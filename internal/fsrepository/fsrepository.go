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
	// strategy is how each file reaches the sandbox: "link", "copy" or "hardlink".
	strategy string

	// hardLinked and copied are exact counters, so which strategy actually ran
	// can be read rather than assumed. A hard link and a copy are both regular
	// files, so nothing downstream can tell them apart -- and the fallback is
	// silent by design, which is exactly the kind of thing that goes unnoticed.
	//
	// They are NOT in perf/baseline.json: hard links cannot cross a filesystem,
	// so the split depends on where a temporary directory lives and is not the
	// machine-independent integer that file gates on.
	hardLinked int
	copied     int
}

// NewWithStrategy materialises sandboxes the named way. See materialize.
func NewWithStrategy(root, strategy string) *FSRepository {
	repository := New(root)
	repository.strategy = strategy

	return repository
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

		return r.materialize(absolutePath, linkPath, entry)
	})
	if err != nil {
		panic(fmt.Errorf("failed scanning '%s': %w", r.root, err))
	}

	return NewTemporary(temporaryPath)
}

// HardLinked and Copied report how each file reached the sandboxes this
// repository built. The fallback is silent by design, so without these nothing
// can say which path a run actually took.
func (r *FSRepository) HardLinked() int { return r.hardLinked }
func (r *FSRepository) Copied() int     { return r.copied }

// materialize puts one file into the sandbox, as a copy or as a link.
//
// The difference is not an implementation detail. A symlink is a reference to a
// file, not a copy of one, and the two part company wherever something inspects
// what a path *is* rather than what it holds. `go:embed` is the instance that
// found this: it refuses an irregular file, so a package with an embed directive
// cannot build in a linked sandbox at all.
func (r *FSRepository) materialize(source, destination string, entry fs.DirEntry) error {
	switch r.strategy {
	case "link":
		// The old strategy, kept reachable because it is what every release before
		// this one used and a measurement may want it back.
		//
		// G122 wants a root-scoped API here. Mirroring a repository as symlinks
		// is what this branch is for, and os.Root cannot express a link that
		// points outside the temporary tree on purpose.
		if err := os.Symlink(source, destination); err != nil {
			return fmt.Errorf("failed creating link from '%s' to '%s': %w", source, destination, err)
		}

		return nil
	case "copy":
		return copyFile(source, destination, entry)
	case "hardlink", "":
		// A hard link is a second name for the same bytes, so it IS a regular
		// file and costs no copy. It is safe here only because Overwrite removes
		// a file before writing a mutant: removing a name leaves the original
		// one alone. A write in place would corrupt the repository instead.
		if err := os.Link(source, destination); err == nil {
			r.hardLinked++

			return nil
		}

		// Hard links cannot cross a filesystem, and a temporary directory often
		// lives on another one. Falling back keeps the sandbox correct where the
		// cheap path is unavailable.
		r.copied++

		return copyFile(source, destination, entry)
	}

	return fmt.Errorf("%w: %q", errUnknownStrategy, r.strategy)
}

// errUnknownStrategy names a sandbox strategy nothing implements.
var errUnknownStrategy = errors.New("unknown sandbox strategy")

func copyFile(source, destination string, entry fs.DirEntry) error {
	info, err := entry.Info()
	if err != nil {
		return fmt.Errorf("failed reading '%s': %w", source, err)
	}

	data, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("failed reading '%s': %w", source, err)
	}

	// G703 traces the destination back to the walked repository. Both ends are
	// named by the caller — the root it asked to mutate and the temporary
	// directory this run created — and the walk is what the function is for.
	if err := os.WriteFile(destination, data, info.Mode().Perm()); err != nil { //nolint:gosec
		return fmt.Errorf("failed writing '%s': %w", destination, err)
	}

	return nil
}

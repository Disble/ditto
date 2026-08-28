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
	"github.com/Disble/ditto/internal/filecopy"
	"github.com/Disble/ditto/internal/gosourcefile"
)

type FSRepository struct {
	root string
	// strategy is how each file reaches the sandbox: "link", "copy" or "hardlink".
	strategy string

	// relinked counts the symlinks reproduced as symlinks. Nothing else can say
	// a sandbox held any, and a run over a tree full of them used to die.
	relinked int

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
func (r *FSRepository) Relinked() int   { return r.relinked }
func (r *FSRepository) Copied() int     { return r.copied }

// materialize puts one file into the sandbox. The default is a copy, and the
// reason is that the other two are not sandboxes.
//
// A symlink is a reference to a file and a hard link is a second name for one.
// Neither is a copy, and both part company with one wherever something inspects
// what a path IS or WRITES to it.
//
// Reading found the first instance: `go:embed` refuses an irregular file, so no
// package with an embed directive can build in a symlinked sandbox. That one
// fails loudly.
//
// Writing found the second, and it does not. A suite that writes a file — a
// golden it decides to update, a fixture it rewrites — writes THROUGH a symlink
// to the target and THROUGH a hard link to the shared inode. Measured on a
// fixture whose test rewrites one tracked file: under links and hard links the
// source repository came back holding `REWRITTEN BY THE SUITE`; under copies it
// came back untouched. A tool that silently edits the repository it was asked to
// measure is worse than one that reports the wrong number.
//
// ditto own mutant write is safe either way — Overwrite removes the path before
// writing, and removing a name leaves the original alone — which is exactly why
// this went unnoticed: the only writer anyone checked was ditto.
//
// The cost is a fixed one. On a 1960-file repository, copying adds ~15s to a
// release however many mutants it runs: 16.5s over four and 15.0s over nine.
func (r *FSRepository) materialize(source, destination string, entry fs.DirEntry) error {
	// A symlink is not a file to copy, whatever the strategy says. WalkDir does
	// not follow one, so it arrives here as a non-directory entry: read as a
	// file, a link to a directory returns EISDIR and kills the whole walk, and a
	// link to a file is silently followed and lands in the sandbox as a regular
	// file the repository never had.
	//
	// What goes in the sandbox is the link itself, carrying its raw target. That
	// is what git does with a tracked symlink, so a full run and a staged run
	// agree about what the repository is; and it is what keeps a relative link
	// resolving inside the sandbox, where a write through it reaches the copy
	// rather than the repository under measurement.
	//
	// See docs/experiments/a-symlink-in-the-tree.md.
	if entry.Type()&fs.ModeSymlink != 0 {
		return r.relink(source, destination, entry)
	}

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
	case "copy", "":
		r.copied++

		return filecopy.File(source, destination, entry) //nolint:wrapcheck // filecopy already names both paths
	case "hardlink":
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

		return filecopy.File(source, destination, entry) //nolint:wrapcheck // filecopy already names both paths
	}

	return fmt.Errorf("%w: %q", errUnknownStrategy, r.strategy)
}

// relink reproduces one symlink in the sandbox, target and all.
//
// Following it instead would copy whatever it points at: a nix profile, a pnpm
// store, a vendored checkout. The tree ditto measures would stop being the tree
// on disk, and it would do so silently.
func (r *FSRepository) relink(source, destination string, entry fs.DirEntry) error {
	if err := filecopy.File(source, destination, entry); err != nil {
		return err //nolint:wrapcheck // filecopy already names both paths
	}

	r.relinked++

	return nil
}

// errUnknownStrategy names a sandbox strategy nothing implements.
var errUnknownStrategy = errors.New("unknown sandbox strategy")

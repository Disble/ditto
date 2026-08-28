package staged

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ConfigName is the file a repository uses to name what the sandbox is missing.
const ConfigName = ".ditto.json"

// Config is what a repository tells ditto about itself.
//
// There is exactly one thing to say, and it exists because of one honest gap: a
// sandbox is built from the index, and some repositories do not build from their
// index alone. A generated directory the build needs — an embedded frontend
// bundle, generated bindings — is on disk and not in git, so the sandbox arrives
// without it and the package that needs it cannot compile.
//
// Reading the index is not the part to give up: it is what makes a verdict about
// the change rather than about the desk it was written on, measured at 7 of 8
// verdicts moving when a release read the worktree instead. So this does not
// widen what is read. It names, one path at a time, what git does not carry.
type Config struct {
	// Generated are repository-relative paths copied from the working tree into
	// the sandbox after the index is materialised.
	//
	// Every one of them must be untracked. A tracked path has an index version,
	// and letting the working tree's win is exactly the hole the sandbox exists
	// to close — so naming one here is refused rather than obeyed.
	Generated []string `json:"generated"`
}

// errTrackedPath reports a configured path that git already carries.
var errTrackedPath = errors.New("is tracked by git, so the index already carries it")

// LoadConfig reads the repository's configuration. A repository without one is
// not an error: most do build from their index, and those need to say nothing.
func (r *Repository) LoadConfig() (Config, error) {
	data, err := os.ReadFile(filepath.Join(r.root, ConfigName))
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, nil
	}

	if err != nil {
		return Config{}, fmt.Errorf("reading %s: %w", ConfigName, err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("reading %s: %w", ConfigName, err)
	}

	return config, nil
}

// CopyGenerated puts the configured paths into a sandbox and reports what it
// copied, so a run says which of its bytes did not come from the index.
//
// A path that does not exist is refused rather than skipped. It is named because
// the build needs it, and a sandbox quietly missing it produces the failure this
// setting was added to prevent.
func (r *Repository) CopyGenerated(sandbox *Sandbox, generated []string) ([]string, error) {
	copied := make([]string, 0, len(generated))

	for _, name := range generated {
		relative := filepath.FromSlash(strings.TrimSpace(name))
		if relative == "" {
			continue
		}

		if err := r.refuseTracked(relative); err != nil {
			return nil, err
		}

		source := filepath.Join(r.root, relative)
		if _, err := os.Stat(source); err != nil {
			return nil, fmt.Errorf("%s names %q, and it is not there: %w", ConfigName, name, err)
		}

		if err := copyTree(source, filepath.Join(sandbox.Root, relative)); err != nil {
			return nil, err
		}

		copied = append(copied, name)
	}

	return copied, nil
}

// refuseTracked keeps the setting to what it is for. Git knowing a path means the
// index has a version of it, and that version is the one a staged run measures.
func (r *Repository) refuseTracked(relative string) error {
	output, err := r.git("ls-files", "--error-unmatch", "-z", "--", relative)
	if err != nil {
		// git exits non-zero precisely when it knows none of the path, which is
		// the case this setting exists for. The error IS the answer here.
		return nil //nolint:nilerr // a non-zero exit means untracked, which is what this permits
	}

	if len(output) > 0 {
		return fmt.Errorf("%s names %q, which %w", ConfigName, filepath.ToSlash(relative), errTrackedPath)
	}

	return nil
}

func copyTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error { //nolint:wrapcheck // the walk's own errors already name the path
		if err != nil {
			return err
		}

		relative, err := filepath.Rel(source, path)
		if err != nil {
			return fmt.Errorf("resolving %s inside %s: %w", path, source, err)
		}

		target := filepath.Join(destination, relative)

		if entry.IsDir() {
			if err := os.MkdirAll(target, os.ModePerm); err != nil {
				return fmt.Errorf("creating %s: %w", target, err)
			}

			return nil
		}

		if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
			return fmt.Errorf("creating %s: %w", filepath.Dir(target), err)
		}

		return copyFile(path, target, entry)
	})
}

// copyFile writes one file into the sandbox as a real file, keeping its mode.
func copyFile(source, destination string, entry fs.DirEntry) error {
	info, err := entry.Info()
	if err != nil {
		return fmt.Errorf("reading %s: %w", source, err)
	}

	data, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("reading %s: %w", source, err)
	}

	if err := os.WriteFile(destination, data, info.Mode().Perm()); err != nil { //nolint:gosec // the destination is this run's sandbox
		return fmt.Errorf("writing %s: %w", destination, err)
	}

	return nil
}

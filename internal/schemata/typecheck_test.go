package schemata_test

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// typeChecker is H2 — go/types over the mutated file, with the package's
// dependencies resolved once from export data and reused for every mutant of
// that package. Nothing is spawned per mutant, which is the claim under test.
type typeChecker struct {
	files  map[string][]byte
	lookup func(string) (io.ReadCloser, error)
	ready  bool
	why    string
}

// newTypeChecker pays H2's whole subprocess budget: one `go list` per package,
// for the export data of its dependencies. That data does not change between
// mutants, because a mutation lives inside the package and never in what it
// imports.
func newTypeChecker(root, packageDir string) *typeChecker {
	checker := &typeChecker{files: map[string][]byte{}}

	entries, err := os.ReadDir(packageDir)
	if err != nil {
		checker.why = err.Error()

		return checker
	}

	for _, entry := range entries {
		if entry.IsDir() || !mutable(entry.Name()) {
			continue
		}

		source, err := os.ReadFile(filepath.Join(packageDir, entry.Name()))
		if err != nil {
			checker.why = err.Error()

			return checker
		}

		checker.files[filepath.Join(packageDir, entry.Name())] = source
	}

	exports, err := exportData(root, packageDir)
	if err != nil {
		checker.why = err.Error()

		return checker
	}

	checker.lookup = func(path string) (io.ReadCloser, error) {
		file, ok := exports[path]
		if !ok {
			return nil, os.ErrNotExist
		}

		return os.Open(file)
	}
	checker.ready = true

	return checker
}

// exportData asks the toolchain where each dependency's compiled export data
// lives. This is the one subprocess H2 is allowed, and it is per package.
func exportData(root, packageDir string) (map[string]string, error) {
	relative, err := filepath.Rel(root, packageDir)
	if err != nil {
		return nil, fmt.Errorf("locating %s under %s: %w", packageDir, root, err)
	}

	//nolint:gosec,noctx // the argument is a path inside the tree the caller named
	command := exec.Command("go", "list", "-export", "-deps",
		"-f", "{{.ImportPath}}\t{{.Export}}", "./"+filepath.ToSlash(relative))
	command.Dir = root

	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("listing export data for %s: %w", relative, err)
	}

	exports := map[string]string{}

	for line := range strings.SplitSeq(string(output), "\n") {
		path, file, found := strings.Cut(strings.TrimSpace(line), "\t")
		if found && file != "" {
			exports[path] = file
		}
	}

	return exports, nil
}

// refuses reports whether the package fails to type-check with this mutant's
// bytes in place of the file it came from.
func (c *typeChecker) refuses(mutatedPath string, mutated []byte) bool {
	if !c.ready {
		return false
	}

	fset := token.NewFileSet()
	parsed := make([]*ast.File, 0, len(c.files))

	for path, source := range c.files {
		if path == mutatedPath {
			source = mutated
		}

		file, err := parser.ParseFile(fset, path, source, parser.SkipObjectResolution)
		if err != nil {
			return true
		}

		parsed = append(parsed, file)
	}

	failed := false
	config := types.Config{
		Importer: importer.ForCompiler(fset, "gc", c.lookup),
		Error:    func(error) { failed = true },
	}

	_, _ = config.Check("mutant", fset, parsed, nil)

	return failed
}

// silent is the control H2 cannot be believed without: the checker must report
// nothing at all on the unmutated package. One that complains about its own
// setup would count every mutant as refused and look like a perfect mechanism.
func (c *typeChecker) silent(mutatedPath string) bool {
	return c.ready && !c.refuses(mutatedPath, c.files[mutatedPath])
}

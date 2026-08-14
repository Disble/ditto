package schemata_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Disble/ditto/internal/gosourcefile"
	"github.com/Disble/ditto/internal/schemata"
	"github.com/stretchr/testify/require"
)

// tally is what the run counts, all of it in exact integers.
type tally struct {
	gated, refused int
	files, broken  int
	failures       []string
}

// TestAdmitsWithoutBreakingTheBuild instruments every file of a real repository
// and asks the compiler whether the result still builds.
//
// It answers the question that decides how a site earns the gated path: whether
// one shared compilation per file is enough, or whether a failure has to be
// bisected. See docs/experiments/admitting-sites.md.
//
// Point DITTO_ADMIT_ROOT at a THROWAWAY COPY. This rewrites files in place, and
// every gosec finding it suppresses below is that instruction being carried out:
// the path comes from the caller on purpose, and the subprocess is the Go
// toolchain being asked about the caller's own tree.
func TestAdmitsWithoutBreakingTheBuild(t *testing.T) {
	root := os.Getenv("DITTO_ADMIT_ROOT")
	if root == "" {
		t.Skip("set DITTO_ADMIT_ROOT to a throwaway copy of a repository")
	}

	counted := &tally{}

	//nolint:gosec // rewriting the tree the caller named is what this probe is for
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !mutable(entry.Name()) {
			return err
		}

		return instrumentOne(t, counted, root, path)
	})
	require.NoError(t, err)

	t.Logf("mutants gated %d, refused %d", counted.gated, counted.refused)
	t.Logf("files with at least one gate %d, of which failed to compile %d", counted.files, counted.broken)

	for _, failure := range counted.failures {
		t.Logf("  broken: %s", failure)
	}
}

func mutable(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

func instrumentOne(t *testing.T, counted *tally, root, path string) error {
	t.Helper()

	original, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	planned := schemata.Plan(original, mutantsOf(t, path, original))
	here := 0

	for _, id := range planned.Selector {
		if id == 0 {
			counted.refused++
		} else {
			counted.gated++
			here++
		}
	}

	// Control: a file with nothing to gate comes back untouched. If instrumenting
	// a file that has nothing to instrument changed it, every number here would
	// be measuring the harness.
	if here == 0 {
		require.Equal(t, string(original), string(planned.Instrumented), path)

		return nil
	}

	counted.files++

	require.NoError(t, os.WriteFile(path, planned.Instrumented, 0o600))

	if ok, output := builds(root, filepath.Dir(path)); !ok {
		counted.broken++
		counted.failures = append(counted.failures, strings.TrimPrefix(path, root)+"\n"+output)
	}

	require.NoError(t, os.WriteFile(path, original, 0o600)) //nolint:gosec // restoring what was just read

	return nil
}

// mutantsOf renders every mutant ditto would make of one file, the same way a
// release does. Only the five viruses whose mutations this package can gate are
// used; the other nine are refused by design and would only pad the counts.
func mutantsOf(t *testing.T, path string, source []byte) [][]byte {
	t.Helper()

	infected := gosourcefile.New(path, source).Incubate(gateableViruses()...)

	mutants := make([][]byte, 0, len(infected))
	for _, one := range infected {
		mutants = append(mutants, one.Mutate().Mutated())
	}

	return mutants
}

func builds(root, packageDir string) (bool, string) {
	relative, err := filepath.Rel(root, packageDir)
	if err != nil {
		return false, err.Error()
	}

	//nolint:gosec,noctx // the argument is a path inside the tree the caller named
	command := exec.Command("go", "build", "./"+filepath.ToSlash(relative))
	command.Dir = root

	output, err := command.CombinedOutput()
	if err != nil {
		return false, string(output)
	}

	return true, ""
}

package perfbench

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Fixture shape. These numbers are part of the contract: the counters in
// perf/baseline.json only mean something against a tree of a known size, so
// changing any of them invalidates every recorded baseline.
const (
	fixtureSourceFiles     = 4
	fixtureFuncsPerFile    = 3
	fixtureGitFiles        = 5
	fixtureNonSourceFiles  = 2 // go.mod and README.md
	fixtureTotalFileCount  = fixtureSourceFiles + fixtureGitFiles + fixtureNonSourceFiles
	fixtureMutablePosition = "value > "
)

// writeFixtureRepository builds a synthetic repository whose shape is fixed by
// the constants above.
//
// It is synthetic on purpose. Measuring against ditto's own tree would make
// every counter drift whenever somebody adds a file, and a baseline that moves
// for unrelated reasons stops being a baseline.
//
// It includes a .git directory holding real files because the cost being
// tracked includes them: LinkAllToTemporaryRepository walks the whole root and
// links everything it finds, and nobody mutates a git object.
func writeFixtureRepository(t testing.TB) string {
	t.Helper()

	root := t.TempDir()

	write := func(name, content string) {
		t.Helper()

		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("go.mod", "module example.invalid/fixture\n\ngo 1.22\n")
	write("README.md", "# fixture\n")

	for file := range fixtureSourceFiles {
		var source strings.Builder

		fmt.Fprintf(&source, "package fixture%d\n", file)

		for fn := range fixtureFuncsPerFile {
			fmt.Fprintf(&source,
				"\nfunc Gate%d(value int) bool {\n\treturn %s%d\n}\n",
				fn, fixtureMutablePosition, (fn+1)*10)
		}

		write(fmt.Sprintf("pkg%d/gate.go", file), source.String())
	}

	for object := range fixtureGitFiles {
		write(fmt.Sprintf(".git/objects/%02d/object", object), "not a go file\n")
	}

	return root
}

// countFiles reports how many regular files a tree holds, which is the unit
// LinkAllToTemporaryRepository pays per mutant.
func countFiles(t testing.TB, root string) int {
	t.Helper()

	count := 0

	err := filepath.WalkDir(root, func(_ string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !entry.IsDir() {
			count++
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	return count
}

func TestFixtureHasTheShapeTheBaselineAssumes(t *testing.T) {
	t.Parallel()

	root := writeFixtureRepository(t)
	if got := countFiles(t, root); got != fixtureTotalFileCount {
		t.Fatalf("fixture holds %d files, want %d; every recorded counter assumes this shape", got, fixtureTotalFileCount)
	}
}

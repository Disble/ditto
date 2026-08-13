package perfbench_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Disble/ditto/internal/gosourcefile"
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
func writeFixtureRepository(tb testing.TB) string {
	tb.Helper()

	root := tb.TempDir()

	write := func(name, content string) {
		tb.Helper()

		path := filepath.Join(root, filepath.FromSlash(name))

		err := os.MkdirAll(filepath.Dir(path), 0o755)
		if err != nil {
			tb.Fatal(err)
		}

		// Fixture sources, readable on purpose: a mutation run reads them back.
		err = os.WriteFile(path, []byte(content), 0o644) //nolint:gosec
		if err != nil {
			tb.Fatal(err)
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
func countFiles(tb testing.TB, root string) int {
	tb.Helper()

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
		tb.Fatal(err)
	}

	return count
}

// gateRange returns the byte range of one gate's mutable line.
//
// Every fixture source file is written to the same byte layout — the package
// number, the gate number and the literal are all one character wide — so gate
// k occupies the same range in every file. That is deliberate: it is the shape
// that makes a scope which has lost track of which file a range came from
// mutate the wrong file, and the only way to prove it does not.
func gateRange(tb testing.TB, root string, gate int) gosourcefile.Range {
	tb.Helper()

	content, err := os.ReadFile(filepath.Join(root, "pkg0", "gate.go"))
	if err != nil {
		tb.Fatal(err)
	}

	marker := []byte("return " + fixtureMutablePosition)
	offset := 0

	for range gate + 1 {
		found := bytes.Index(content[offset:], marker)
		if found < 0 {
			tb.Fatalf("fixture holds fewer than %d mutable lines", gate+1)
		}

		offset += found + 1
	}

	start := offset - 1
	end := start + bytes.IndexByte(content[start:], '\n')

	return gosourcefile.Range{Start: start, End: end}
}

func TestGateRangesAreIdenticalAcrossFixtureFiles(t *testing.T) {
	t.Parallel()

	root := writeFixtureRepository(t)

	first, err := os.ReadFile(filepath.Join(root, "pkg0", "gate.go"))
	if err != nil {
		t.Fatal(err)
	}

	for file := 1; file < fixtureSourceFiles; file++ {
		other, err := os.ReadFile(filepath.Join(root, fmt.Sprintf("pkg%d", file), "gate.go"))
		if err != nil {
			t.Fatal(err)
		}

		if len(other) != len(first) {
			t.Fatalf("pkg%d/gate.go is %d bytes, pkg0/gate.go is %d; the collision counter needs them identical",
				file, len(other), len(first))
		}
	}
}

func TestFixtureHasTheShapeTheBaselineAssumes(t *testing.T) {
	t.Parallel()

	root := writeFixtureRepository(t)
	if got := countFiles(t, root); got != fixtureTotalFileCount {
		t.Fatalf("fixture holds %d files, want %d; every recorded counter assumes this shape", got, fixtureTotalFileCount)
	}
}

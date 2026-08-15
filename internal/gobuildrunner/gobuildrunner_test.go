package gobuildrunner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Disble/ditto/internal/dittotesting/fakerepository"
	"github.com/Disble/ditto/internal/gobuildrunner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The measured reason this exists: the fixed cost of starting `go test` is
// 750-950 ms per mutant whatever the suite costs, and it is the dominant cost of
// a run. Building once and running the binary per mutant removes it — but only
// if the build really happens once, which is a counter, not an impression.
func TestGoBuildRunner(t *testing.T) {
	t.Run("builds once however many mutants run", func(t *testing.T) {
		root := fixture(t, "package calc\n\nfunc Ok() bool { return true }\n", passing)
		runner := gobuildrunner.New("./calc")

		for range 5 {
			runner.Test(fakerepository.NewTemporaryAt(root))
		}

		assert.Equal(t, 1, runner.Compilations())
		assert.Equal(t, 5, runner.Runs())
	})

	// Naming the command and letting the operating system find it means PATH
	// decides which compiler runs, and PATH is read again at every call. Two
	// things follow, and they point the same way.
	//
	// The security one is SonarQube's go:S4036: a directory an attacker can write
	// to, or prepend, substitutes the binary. The correctness one matters more
	// here — ditto compares verdicts, so building a mutant's tests with a
	// different toolchain from the one running the suite would make a
	// disagreement that is nobody's mutation look like one that is.
	t.Run("builds with an absolute toolchain rather than whatever PATH resolves", func(t *testing.T) {
		toolchain := gobuildrunner.New("./calc").Toolchain()

		require.NotEmpty(t, toolchain, "no go toolchain was resolved")
		assert.True(t, filepath.IsAbs(toolchain), "toolchain %q is not an absolute path", toolchain)

		info, err := os.Stat(toolchain)
		require.NoError(t, err, "resolved toolchain does not exist")
		assert.False(t, info.IsDir(), "resolved toolchain is a directory")
	})

	t.Run("reads a passing suite as a mutant that survived", func(t *testing.T) {
		root := fixture(t, "package calc\n\nfunc Ok() bool { return true }\n", passing)

		outcome := gobuildrunner.New("./calc").Test(fakerepository.NewTemporaryAt(root))

		assert.False(t, outcome.IsOk(), "a passing suite means nothing caught the mutant")
	})

	t.Run("reads a failing suite as a mutant that was killed", func(t *testing.T) {
		root := fixture(t, "package calc\n\nfunc Ok() bool { return false }\n", passing)

		outcome := gobuildrunner.New("./calc").Test(fakerepository.NewTemporaryAt(root))

		assert.True(t, outcome.IsOk(), "a failing suite means a test caught it")
	})

	// Measured the hard way: `go test` runs a suite from the package's own
	// directory, and running the binary from the module root instead made an
	// untouched package fail. It looked exactly like the instrumentation having
	// broken something.
	t.Run("runs from the package's own directory", func(t *testing.T) {
		root := fixture(t,
			"package calc\n\nfunc Ok() bool { return true }\n",
			"package calc\n\nimport (\n\t\"os\"\n\t\"testing\"\n)\n\n"+
				"func TestReadsItsOwnFile(t *testing.T) {\n"+
				"\tif _, err := os.Stat(\"marker\"); err != nil {\n\t\tt.Fatal(err)\n\t}\n}\n",
		)
		require.NoError(t, os.WriteFile(filepath.Join(root, "calc", "marker"), []byte("here"), 0o600))

		outcome := gobuildrunner.New("./calc").Test(fakerepository.NewTemporaryAt(root))

		assert.False(t, outcome.IsOk(), "the suite should pass, and it only can from its own directory")
	})

	t.Run("hands the selected mutant to the binary", func(t *testing.T) {
		root := fixture(t,
			"package calc\n\nfunc Ok() bool { return true }\n",
			"package calc\n\nimport (\n\t\"os\"\n\t\"testing\"\n)\n\n"+
				"func TestSelected(t *testing.T) {\n"+
				"\tif os.Getenv(\"DITTO_MUTANT\") != \"7\" {\n\t\tt.Fatal(\"not selected\")\n\t}\n}\n",
		)

		runner := gobuildrunner.New("./calc")
		runner.Select(7)

		outcome := runner.Test(fakerepository.NewTemporaryAt(root))

		assert.False(t, outcome.IsOk(), "the suite passes only when it sees the selected mutant")
	})

	// A package that does not build has to answer, not panic. Under one shared
	// compilation this is how a bad instrumented file surfaces, and the caller
	// decides to fall back rather than being killed by it.
	t.Run("answers when the package does not build", func(t *testing.T) {
		root := fixture(t, "package calc\n\nfunc Ok() bool { return \n", passing)

		runner := gobuildrunner.New("./calc")
		outcome := runner.Test(fakerepository.NewTemporaryAt(root))

		assert.False(t, runner.Built())
		assert.Contains(t, outcome.String(), "calc")
	})
}

const passing = "package calc\n\nimport \"testing\"\n\n" +
	"func TestOk(t *testing.T) {\n\tif !Ok() {\n\t\tt.Fatal(\"not ok\")\n\t}\n}\n"

// fixture builds a throwaway module. Nothing here goes near a checkout with work
// in it.
func fixture(t *testing.T, source, test string) string {
	t.Helper()

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "calc"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n\ngo 1.25\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "calc", "calc.go"), []byte(source), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "calc", "calc_test.go"), []byte(test), 0o600))

	return root
}

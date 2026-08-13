package perfbench_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/Disble/ditto/internal/fsrepository"
)

// benchRoot is the repository the benchmarks below measure against. These
// numbers only mean something on a real tree, so there is no fixture default.
func benchRoot(b *testing.B) string {
	b.Helper()

	root := os.Getenv("OOZE_BENCH_ROOT")
	if root == "" {
		b.Skip("set OOZE_BENCH_ROOT to a real repository")
	}

	return root
}

// BenchmarkLinkAllToTemporaryRepository measures what one mutant costs before
// its test command even starts: Laboratory.Test builds a whole temporary
// repository per mutant and removes it afterwards.
func BenchmarkLinkAllToTemporaryRepository(b *testing.B) {
	root := benchRoot(b)
	repository := fsrepository.New(root)

	for i := 0; i < b.N; i++ {
		temporary, err := os.MkdirTemp("", "dittobench-")
		if err != nil {
			b.Fatal(err)
		}

		repository.LinkAllToTemporaryRepository(temporary).Remove()
	}
}

// BenchmarkListGoSourceFiles measures the walk that selects mutable files. It
// runs once per release, not per mutant.
func BenchmarkListGoSourceFiles(b *testing.B) {
	root := benchRoot(b)
	repository := fsrepository.New(root)

	for i := 0; i < b.N; i++ {
		repository.ListGoSourceFiles()
	}
}

// TestReportBenchShape prints what the walk actually traverses, which is the
// number the per-mutant cost scales with.
func TestReportBenchShape(t *testing.T) {
	root := os.Getenv("OOZE_BENCH_ROOT")
	if root == "" {
		t.Skip("set OOZE_BENCH_ROOT to a real repository")
	}

	var all, git, goSource int

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err //nolint:wrapcheck
		}

		all++

		if relative, relErr := filepath.Rel(root, path); relErr == nil &&
			len(relative) >= 4 && relative[:4] == ".git" {
			git++
		}

		if filepath.Ext(path) == ".go" {
			goSource++
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("files walked per mutant: %d (of which .git: %d), go sources: %d", all, git, goSource)
}

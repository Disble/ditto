package verdict_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Disble/ditto/internal/verdict"
)

// The constants in verdict_test.go were captured by hand from go1.27, and a
// captured shape is a claim about a toolchain rather than a fact about the one
// running. This asks the real `go test` both questions and reads its real
// answer, so a Go release that changes the stream fails here rather than
// silently turning every unearned kill back into an assertion.
func TestAgainstTheRealGoTest(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name   string
		source string
		want   verdict.Reason
	}{
		{
			name:   "a package that does not compile",
			source: "package pkg\n\nfunc Add(a, b int) int { return a + notAThing }\n",
			want:   verdict.BuildFailed,
		},
		{
			name:   "a test that fails an assertion",
			source: "package pkg\n\nfunc Add(a, b int) int { return a - b }\n",
			want:   verdict.Assertion,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := writeModule(t, testCase.source)

			command := exec.Command("go", "test", "-count=1", "-json", "./...") //nolint:noctx // the fixture this test just wrote
			command.Dir = root

			output, err := command.CombinedOutput()
			if err == nil {
				t.Fatal("the fixture was supposed to fail")
			}

			if got := verdict.ReasonOf(string(output)); got != testCase.want {
				t.Fatalf("ReasonOf = %q, want %q\n--- output ---\n%s", got, testCase.want, output)
			}
		})
	}
}

// TestTheExitCodeCannotCarryTheReason is the control, and it is the reason this
// package parses anything at all. If a Go release ever did distinguish the two
// by exit status, the JSON parsing would be dead weight and this test says so.
func TestTheExitCodeCannotCarryTheReason(t *testing.T) {
	t.Parallel()

	broken := exitCodeOf(t, writeModule(t, "package pkg\n\nfunc Add(a, b int) int { return a + notAThing }\n"))
	failing := exitCodeOf(t, writeModule(t, "package pkg\n\nfunc Add(a, b int) int { return a - b }\n"))

	if broken != failing {
		t.Fatalf("go test now distinguishes them by exit code: %d for a broken build, %d for a failed test", broken, failing)
	}
}

func exitCodeOf(t *testing.T, root string) int {
	t.Helper()

	command := exec.Command("go", "test", "-count=1", "./...") //nolint:noctx // the fixture this test just wrote
	command.Dir = root

	if err := command.Run(); err != nil {
		if exitError, ok := errors.AsType[*exec.ExitError](err); ok {
			return exitError.ExitCode()
		}

		t.Fatalf("running the fixture: %v", err)
	}

	return 0
}

func writeModule(t *testing.T, source string) string {
	t.Helper()

	root := t.TempDir()
	write(t, filepath.Join(root, "go.mod"), "module example.com/x\n\ngo 1.25\n")
	write(t, filepath.Join(root, "pkg", "a.go"), source)
	write(t, filepath.Join(root, "pkg", "a_test.go"),
		"package pkg\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(1, 2) != 3 {\n\t\tt.Fatal(\"no\")\n\t}\n}\n")

	return root
}

func write(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

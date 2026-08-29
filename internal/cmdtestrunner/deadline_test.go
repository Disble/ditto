package cmdtestrunner_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Disble/ditto/internal/cmdtestrunner"
	"github.com/Disble/ditto/internal/dittotesting/fakerepository"
	"github.com/Disble/ditto/internal/result"
)

// A command that never returns must not hang the release.
//
// Ditto imposed no deadline anywhere before this: `grep -rn timeout
// --include=*.go` found one doc comment, and gobuildrunner runs a test binary
// directly, where -test.timeout defaults to 0 and a looping mutant never comes
// back. loopcondition, loopbreak and rangebreak are all in the default virus
// set, so writing an endless loop is routine. docs/backlog.md entry 15.
func TestACommandThatNeverReturnsIsStopped(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeBlockingProgram(t, root)

	runner := cmdtestrunner.NewWithDeadline(300*time.Millisecond, "go", "run", ".")
	done := make(chan result.Result[string], 1)

	go func() { done <- runner.Test(fakerepository.NewTemporaryAt(root)) }()

	select {
	case res := <-done:
		// A mutant the deadline stopped is a kill -- PIT, Stryker and Infection
		// all agree -- and it carries its own reason rather than passing for an
		// assertion.
		if !res.IsOk() {
			t.Fatal("a mutant stopped by the deadline is a kill")
		}

		if !strings.Contains(result.Output(res), "deadline") {
			t.Fatalf("the output does not say the deadline fired: %q", result.Output(res))
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the deadline did not fire; a run over this mutant would never end")
	}
}

// The control: the same runner must not stop a command that finishes inside the
// deadline, or every slow-but-honest suite becomes a false kill.
func TestACommandThatFinishesIsLeftAlone(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeQuickProgram(t, root)

	res := cmdtestrunner.NewWithDeadline(60*time.Second, "go", "run", ".").
		Test(fakerepository.NewTemporaryAt(root))

	if res.IsOk() {
		t.Fatalf("a command that succeeded was read as a kill: %q", result.Output(res))
	}
}

func writeBlockingProgram(t *testing.T, root string) {
	t.Helper()
	// A busy loop, not `select {}`: Go's runtime detects an all-goroutines-asleep
	// deadlock and exits on its own, which is not a hang and would have let this
	// test pass for the wrong reason. Measured -- it did, on Linux, while Windows
	// timed out first. A spinning loop is also the real case: loopcondition is in
	// the default virus set and writing one is what it does.
	writeProgram(t, root, "package main\n\nfunc main() {\n\tfor {\n\t}\n}\n")
}

func writeQuickProgram(t *testing.T, root string) {
	t.Helper()
	writeProgram(t, root, "package main\n\nfunc main() {}\n")
}

func writeProgram(t *testing.T, root, main string) {
	t.Helper()

	for name, content := range map[string]string{
		"go.mod":  "module example.com/blocking\n\ngo 1.25\n",
		"main.go": main,
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

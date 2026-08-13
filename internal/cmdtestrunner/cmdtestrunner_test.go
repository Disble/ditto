package cmdtestrunner_test

import (
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Disble/ditto/internal/cmdtestrunner"
	"github.com/Disble/ditto/internal/dittotesting/fakerepository"
	"github.com/Disble/ditto/internal/result"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gitEnvironment is named here rather than imported from the code under test.
// A test that asks the implementation which variables to check would keep
// passing if the implementation forgot one.
var gitEnvironment = []string{ //nolint:gochecknoglobals
	"GIT_DIR",
	"GIT_INDEX_FILE",
	"GIT_WORK_TREE",
	"GIT_OBJECT_DIRECTORY",
	"GIT_COMMON_DIR",
}

func TestCMDTestRunner(t *testing.T) {
	t.Run("has a positive result when subprocess exists unsuccessfully", func(t *testing.T) {
		temporaryRepository := fakerepository.NewTemporaryAt(t.TempDir())

		output := cmdtestrunner.New("sh", "-c", "printf 'tests failed'; exit 1").Test(temporaryRepository)
		assert.Equal(t, result.Ok("tests failed"), output)
	})

	t.Run("has a negative result when subprocess exists successfully", func(t *testing.T) {
		temporaryRepository := fakerepository.NewTemporaryAt(t.TempDir())

		output := cmdtestrunner.New("sh", "-c", "printf 'tests passed'; exit 0").Test(temporaryRepository)
		assert.Equal(t, result.Err[string]("tests passed"), output)
	})

	t.Run("runs within the given directory context", func(t *testing.T) {
		dir := t.TempDir()
		temporaryRepository := fakerepository.NewTemporaryAt(dir)

		output := cmdtestrunner.New("sh", "-c", "basename $(pwd)").Test(temporaryRepository)
		assert.Equal(t, result.Err[string](filepath.Base(dir)+"\n"), output)
	})

	t.Run("keeps git's inherited environment away from the subprocess", func(t *testing.T) {
		temporaryRepository := fakerepository.NewTemporaryAt(t.TempDir())

		for _, variable := range gitEnvironment {
			t.Setenv(variable, "somewhere/else/.git")
		}

		output := cmdtestrunner.New("sh", "-c",
			`printf '%s|%s|%s|%s|%s' "$GIT_DIR" "$GIT_INDEX_FILE" "$GIT_WORK_TREE" "$GIT_OBJECT_DIRECTORY" "$GIT_COMMON_DIR"`,
		).Test(temporaryRepository)

		assert.Equal(t, result.Err[string]("||||"), output)
	})

	// The assertion above is about five names. This one is about the damage they
	// do, and it is the test that would have caught the original incident: git
	// exports these to hooks, absolute in a linked worktree, so a command meant
	// for the sandbox reaches the real checkout instead — and succeeds, which is
	// why nobody noticed twice.
	t.Run("cannot reach a repository outside the sandbox", func(t *testing.T) {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("git is not available")
		}

		decoy := newDecoyRepository(t)
		before := stateOf(t, decoy)

		t.Setenv("GIT_DIR", filepath.Join(decoy, ".git"))
		t.Setenv("GIT_INDEX_FILE", filepath.Join(decoy, ".git", "index"))

		temporaryRepository := fakerepository.NewTemporaryAt(t.TempDir())
		cmdtestrunner.New("sh", "-c",
			`git config user.name intruder; git commit --allow-empty -m intruder`,
		).Test(temporaryRepository)

		assert.Equal(t, before, stateOf(t, decoy), "the subprocess reached the decoy repository")
	})

	t.Run("makes all environment variables available to the subprocess", func(t *testing.T) {
		temporaryRepository := fakerepository.NewTemporaryAt(t.TempDir())

		t.Setenv("TEST_VAR_1", "test_value_1")

		output := cmdtestrunner.New("sh", "-c", "printf $TEST_VAR_1").Test(temporaryRepository)
		assert.Equal(t, result.Err[string]("test_value_1"), output)

		t.Setenv("TEST_VAR_2", "test_value_2")

		output = cmdtestrunner.New("sh", "-c", "printf $TEST_VAR_2").Test(temporaryRepository)
		assert.Equal(t, result.Err[string]("test_value_2"), output)
	})
}

// newDecoyRepository builds a repository nothing under test is allowed to
// touch. If the thing being tested can reach outside its fixture, the decoy
// shows it, and nothing real is lost.
func newDecoyRepository(t *testing.T) string {
	t.Helper()

	decoy := t.TempDir()

	git(t, decoy, "init", "--initial-branch", "decoy")
	git(t, decoy, "config", "user.name", "decoy")
	git(t, decoy, "config", "user.email", "decoy@example.invalid")
	git(t, decoy, "commit", "--allow-empty", "--message", "decoy")

	return decoy
}

// stateOf records what the original incident changed: the branch tip, how many
// commits there are, and the identity commits are made under.
func stateOf(t *testing.T, decoy string) string {
	t.Helper()

	return strings.Join([]string{
		git(t, decoy, "rev-parse", "HEAD"),
		git(t, decoy, "rev-list", "--count", "HEAD"),
		git(t, decoy, "config", "--local", "--get", "user.name"),
	}, " ")
}

// git addresses the decoy explicitly, because by the time the decoy is read
// back the test has deliberately pointed the inherited environment somewhere
// else, and a reader that trusted the environment would report on the wrong
// repository.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()

	command := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir}, args...)...) //nolint:gosec // fixture paths
	command.Env = append(
		withoutGitEnvironment(command.Environ()),
		"GIT_DIR="+filepath.Join(dir, ".git"),
		"GIT_WORK_TREE="+dir,
	)

	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))

	return strings.TrimSpace(string(output))
}

func withoutGitEnvironment(environment []string) []string {
	kept := make([]string, 0, len(environment))

	for _, variable := range environment {
		name, _, _ := strings.Cut(variable, "=")
		if !slices.Contains(gitEnvironment, name) {
			kept = append(kept, variable)
		}
	}

	return kept
}

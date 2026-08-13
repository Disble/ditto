package cmdtestrunner

import (
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/result"
)

// gitEnvironment is what git exports to a hook, and in a linked worktree it
// exports them as absolute paths. Everything spawned below inherits them, so a
// command meant for the sandbox addresses the real checkout instead. It then
// succeeds, which is the whole problem: the damage is silent and the test still
// passes.
//
// Measured twice before this list existed — once replacing a live branch's tree
// with a commit named `fixture`, once setting core.bare on the real repository
// and writing the fixture's identity into user.name. See AGENTS.md.
var gitEnvironment = []string{ //nolint:gochecknoglobals // one fixed list, read only
	"GIT_DIR",
	"GIT_INDEX_FILE",
	"GIT_WORK_TREE",
	"GIT_OBJECT_DIRECTORY",
	"GIT_COMMON_DIR",
}

type CMDTestRunner struct {
	name string
	args []string
}

func New(name string, args ...string) *CMDTestRunner {
	return &CMDTestRunner{
		name: name,
		args: args,
	}
}

func (t *CMDTestRunner) Test(repository ditto.TemporaryRepository) result.Result[string] {
	// noctx wants CommandContext. The configured test command has no
	// cancellation contract today, and giving it one is an API decision rather
	// than a lint fix; recorded in docs/backlog.md.
	command := exec.Command(t.name, t.args...) //nolint:gosec,noctx
	command.Dir = repository.Root()
	command.Env = withoutGitEnvironment(os.Environ())

	output, err := command.CombinedOutput()
	if err != nil {
		return result.Ok(string(output))
	}

	return result.Err[string](string(output))
}

// withoutGitEnvironment removes the inherited git addressing and nothing else.
// The test command still sees every other variable, because configuring a
// project's tests through the environment is ordinary and this is not the place
// to have an opinion about it.
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

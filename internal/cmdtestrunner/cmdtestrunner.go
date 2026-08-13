package cmdtestrunner

import (
	"os"
	"os/exec"

	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/result"
)

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
	command.Env = os.Environ()

	output, err := command.CombinedOutput()
	if err != nil {
		return result.Ok(string(output))
	}

	return result.Err[string](string(output))
}

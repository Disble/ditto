package cmdtestrunner

import (
	"context"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/result"
	"github.com/Disble/ditto/internal/verdict"
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
	name     string
	args     []string
	deadline time.Duration
}

func New(name string, args ...string) *CMDTestRunner {
	return NewWithDeadline(DefaultDeadline, name, args...)
}

// DefaultDeadline bounds one mutant's test command.
//
// Ditto bounded nothing before this. `gobuildrunner` runs a test binary
// directly, where -test.timeout defaults to 0 and a looping mutant never
// returns, and loopcondition, loopbreak and rangebreak are all in the default
// virus set. On the ordinary path the only bound was whatever the user's own
// command carried.
//
// Ten minutes is `go test`'s own default, which is the number a Go suite is
// already written against, so it stops a hang without turning a slow honest
// suite into a false kill. PIT and Stryker derive theirs from the unmutated
// run's duration instead -- a factor plus a constant -- which is better and is
// recorded in docs/backlog.md rather than guessed at here.
const DefaultDeadline = 10 * time.Minute

// NewWithDeadline is New with the bound named. A mutant whose command runs past
// it is stopped and reported as killed BY THE DEADLINE, which PIT, Stryker and
// Infection all treat as a kill carrying its own reason.
func NewWithDeadline(deadline time.Duration, name string, args ...string) *CMDTestRunner {
	return &CMDTestRunner{
		name:     name,
		args:     args,
		deadline: deadline,
	}
}

func (t *CMDTestRunner) Test(repository ditto.TemporaryRepository) result.Result[string] {
	ctx, cancel := context.WithTimeout(context.Background(), t.deadline)
	defer cancel()

	command := exec.CommandContext(ctx, t.name, t.args...) //nolint:gosec // the command is the caller's own
	command.Dir = repository.Root()
	command.Env = withoutGitEnvironment(os.Environ())

	output, err := command.CombinedOutput()

	// The deadline is ditto's own clock, so ditto knows it fired without reading
	// anything. A mutant stopped this way is killed -- a suite that never
	// returns is a difference the tests did notice -- and it says so, because a
	// kill whose reason is a clock is not a kill an assertion earned.
	if ctx.Err() != nil {
		return result.Ok(string(output) + "\n" + deadlineNotice(t.deadline))
	}

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

// deadlineNotice is the line the report and internal/verdict both read. It is a
// sentence rather than a code because it also has to be legible in the captured
// output a reader sees.
func deadlineNotice(deadline time.Duration) string {
	return "ditto: the test command passed its " + deadline.String() +
		" deadline and was stopped; " + verdict.DeadlineMarker
}

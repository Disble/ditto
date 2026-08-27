// Command ditto runs mutation testing from a shell instead of from a test.
//
// Everything it does was already reachable as a library, and for a long time
// that was the only way in: ditto.Release takes a *testing.T, so a caller had to
// be a test, and anything that wanted to drive a release from outside had to
// build a test to get in. That is where the wrappers came from — a file behind a
// build tag, a contract of environment variables, and a `go test` invocation
// standing in for a command line.
//
// This is the command those wrappers were standing in for.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/Disble/ditto"
)

// exitFailure is what a run that reached a verdict and did not like it returns.
// A refusal and a usage error are told apart by their message, not by this.
const exitFailure = 1

func main() {
	if err := command(os.Args[1:], os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitFailure)
	}
}

// errNoSubcommand is what an invocation with nothing to do reports.
var errNoSubcommand = errors.New("ditto needs a subcommand, for example: ditto run")

func command(args []string, help *os.File) error {
	if len(args) == 0 {
		usage(help)

		return errNoSubcommand
	}

	switch args[0] {
	case "run":
		return runCommand(args[1:])
	case "-h", "--help", "help":
		usage(help)

		return nil
	default:
		usage(help)

		return fmt.Errorf("ditto has no subcommand %q", args[0]) //nolint:err113 // the name is the message
	}
}

func usage(out *os.File) {
	fmt.Fprint(out, `ditto — mutation testing for Go

  ditto run [flags]    mutate a repository and report what survived

Run `+"`ditto run -h`"+` for its flags.
`)
}

// excludes collects a flag that may appear more than once, because a repository
// rarely has exactly one thing worth leaving out.
type excludes []string

func (e *excludes) String() string { return strings.Join(*e, ",") }

func (e *excludes) Set(value string) error {
	*e = append(*e, value)

	return nil
}

func runCommand(args []string) error {
	flags := flag.NewFlagSet("ditto run", flag.ContinueOnError)
	root := flags.String("root", ".", "repository root to mutate")
	testCommand := flags.String("test-command", "go test -count=1 ./...", "command that decides whether a mutant died")
	threshold := flags.Float64("threshold", 1.0, "minimum mutation score, from 0 to 1")
	gated := flags.Bool("gated", false, "run a file's mutants from one compilation instead of one each")
	loud := flags.Bool("verbose", false, "print what the run is doing as it does it")

	var exclude excludes

	flags.Var(&exclude, "exclude", "regexp of source paths not to mutate; repeatable")

	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("reading the flags: %w", err)
	}

	if *threshold < 0 || *threshold > 1 {
		return fmt.Errorf("--threshold is %.2f, and a mutation score is between 0 and 1", *threshold) //nolint:err113 // the number is the message
	}

	options, err := optionsFor(*root, *testCommand, float32(*threshold), *gated, *loud, exclude)
	if err != nil {
		return err
	}

	return ditto.Run(options...) //nolint:wrapcheck // this is the top of the program: the message is already the one a reader needs
}

func optionsFor(
	root, testCommand string,
	threshold float32,
	gated, loud bool,
	exclude excludes,
) ([]ditto.Option, error) {
	options := []ditto.Option{
		ditto.WithRepositoryRoot(root),
		ditto.WithTestCommand(testCommand),
		ditto.WithMinimumThreshold(threshold),
	}

	for _, pattern := range exclude {
		if _, err := regexp.Compile(pattern); err != nil {
			return nil, fmt.Errorf("--exclude %q is not a valid regexp: %w", pattern, err)
		}
	}

	if len(exclude) > 0 {
		options = append(options, ditto.IgnoreSourceFiles(exclude...))
	}

	if gated {
		options = append(options, ditto.Gated())
	}

	if loud {
		options = append(options, ditto.Verbose())
	}

	return options, nil
}

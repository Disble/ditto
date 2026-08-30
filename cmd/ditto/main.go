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
	"io"
	"os"
	"regexp"
	"runtime/debug"
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

func command(args []string, help io.Writer) error {
	if len(args) == 0 {
		usage(help)

		return errNoSubcommand
	}

	switch args[0] {
	case "run":
		return runCommand(args[1:])
	case "staged":
		return stagedCommand(args[1:], os.Stdout)
	case "changed":
		return changedCommand(args[1:], os.Stdout)
	case "-h", "--help", "help":
		usage(help)

		return nil
	case "-version", "--version", "version":
		fmt.Fprintln(help, versionLine(buildVersion()))

		return nil
	default:
		usage(help)

		return fmt.Errorf("ditto has no subcommand %q", args[0]) //nolint:err113 // the name is the message
	}
}

// buildVersion is the module version recorded in this binary.
//
// There is no linker-stamped constant to keep in step with the tag, on purpose:
// a constant is a second place the version lives and a second place it goes
// stale. `go install` records the version it resolved and `runtime/debug` reads
// it back, so the only source is the one the toolchain already wrote.
func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}

	return info.Main.Version
}

// versionLine renders what `ditto --version` prints.
//
// A binary built from a checkout rather than installed from a tag has no
// released version to report -- `go build` records `(devel)` and a test binary
// records nothing. Saying so is the point: the answer to whether a mutant that
// never compiled is scored as killed differs between v0.6.0 and v0.7.0, so a
// version this command is unsure of is worse than one it declines to name.
func versionLine(version string) string {
	if version == "" || version == "(devel)" {
		return "ditto (built from source; no released version recorded)"
	}

	return "ditto " + version
}

func usage(out io.Writer) {
	fmt.Fprint(out, `ditto — mutation testing for Go

  ditto run [flags]       mutate a repository and report what survived
  ditto staged [flags]    mutate only what a staged change justifies
  ditto changed [flags]   mutate only what a committed change justifies
  ditto version           the module version this binary was built from

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
	testCommand := flags.String("test-command", "go test -count=1 -json ./...", testCommandHelp)
	threshold := flags.Float64("threshold", 1.0, "minimum mutation score, from 0 to 1")
	gated := flags.Bool("gated", false, "run a file's mutants from one compilation instead of one each")
	confirm := flags.Bool("confirm-kills", false, confirmKillsHelp)
	loud := flags.Bool("verbose", false, "print what the run is doing as it does it")
	sandbox := flags.String("sandbox", "", `how each file reaches the sandbox: "copy" (default), "hardlink" or "link"`)

	var exclude excludes

	flags.Var(&exclude, "exclude", "regexp of source paths not to mutate; repeatable")

	if err := flags.Parse(args); err != nil {
		// Asking for help is not a failure. flag reports it as one under
		// ContinueOnError, and reporting it back made `-h` exit non-zero.
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		return fmt.Errorf("reading the flags: %w", err)
	}

	if *threshold < 0 || *threshold > 1 {
		return fmt.Errorf("--threshold is %.2f, and a mutation score is between 0 and 1", *threshold) //nolint:err113 // the number is the message
	}

	options, err := optionsFor(*root, *testCommand, float32(*threshold), *gated, *confirm, *loud, exclude)
	if err != nil {
		return err
	}

	if *sandbox != "" {
		options = append(options, ditto.WithSandboxStrategy(*sandbox))
	}

	return ditto.Run(options...) //nolint:wrapcheck // this is the top of the program: the message is already the one a reader needs
}

func optionsFor(
	root, testCommand string,
	threshold float32,
	gated, confirm, loud bool,
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

	if confirm {
		options = append(options, ditto.ConfirmKills())
	}

	if loud {
		options = append(options, ditto.Verbose())
	}

	return options, nil
}

func stagedCommand(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("ditto staged", flag.ContinueOnError)
	directory := flags.String("cwd", ".", "a directory inside the repository; its root is resolved from here")
	testCommand := flags.String("test-command", "go test -count=1 -json ./...", testCommandHelp)
	threshold := flags.Float64("threshold", 1.0, "minimum mutation score, from 0 to 1")
	dry := flags.Bool("dry", false, "report what the staged change justifies and run nothing")
	gated := flags.Bool("gated", false, "run a file's mutants from one compilation instead of one each")
	confirm := flags.Bool("confirm-kills", false, confirmKillsHelp)
	loud := flags.Bool("verbose", false, "print what the run is doing as it does it")
	sandbox := flags.String("sandbox", "", `how each file reaches the sandbox: "copy" (default), "hardlink" or "link"`)

	var exclude excludes

	flags.Var(&exclude, "exclude-prefix", "repository-relative prefix never worth mutating; repeatable")

	// There is no flag for .ditto.json, so `-h` is the one place a reader would
	// look and not find it. Named here rather than left to the readme.
	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage of ditto staged:")
		flags.PrintDefaults()
		fmt.Fprint(os.Stderr, stagedConfigHelp)
	}

	if err := flags.Parse(args); err != nil {
		// Asking for help is not a failure. flag reports it as one under
		// ContinueOnError, and reporting it back made `-h` exit non-zero.
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		return fmt.Errorf("reading the flags: %w", err)
	}

	if *threshold < 0 || *threshold > 1 {
		return fmt.Errorf("--threshold is %.2f, and a mutation score is between 0 and 1", *threshold) //nolint:err113 // the number is the message
	}

	if *dry {
		return reportPlan(*directory, exclude, out)
	}

	options := stagedOptions(*testCommand, float32(*threshold), *gated, *confirm, *loud, *sandbox)

	return ditto.RunStaged(*directory, exclude, options...) //nolint:wrapcheck // this is the top of the program: the message is already the one a reader needs
}

// stagedOptions is what the staged flags mean, kept apart from reading them.
//
// Gating is here because it was missing: `run` registered the flag and `staged`
// did not, so every mutant of a staged run paid the 750-950 ms cost of starting
// the test command with no way to decline it. Nothing about one compilation per
// file is specific to a whole-repository release, and the staged path mutates
// fewer files, which is the case where that compilation is easiest to repay.
func stagedOptions(testCommand string, threshold float32, gated, confirm, loud bool, sandbox string) []ditto.Option {
	options := []ditto.Option{
		ditto.WithTestCommand(testCommand),
		ditto.WithMinimumThreshold(threshold),
	}

	if gated {
		options = append(options, ditto.Gated())
	}

	if confirm {
		options = append(options, ditto.ConfirmKills())
	}

	if loud {
		options = append(options, ditto.Verbose())
	}

	if sandbox != "" {
		options = append(options, ditto.WithSandboxStrategy(sandbox))
	}

	return options
}

// changedCommand is `staged` for a change that is already committed.
//
// A gate cannot ask the index anything: on a CI checkout nothing is staged, so a
// run pointed at the staged scope skips and reports a green that measured
// nothing. This asks the same question of `--since ref ... HEAD` instead.
func changedCommand(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("ditto changed", flag.ContinueOnError)
	directory := flags.String("cwd", ".", "a directory inside the repository; its root is resolved from here")
	since := flags.String("since", "", "the `ref` to measure the change against, for example a tag or origin/main")
	testCommand := flags.String("test-command", "go test -count=1 -json ./...", testCommandHelp)
	threshold := flags.Float64("threshold", 1.0, "minimum mutation score, from 0 to 1")
	dry := flags.Bool("dry", false, "report what the change justifies and run nothing")
	gated := flags.Bool("gated", false, "run a file's mutants from one compilation instead of one each")
	confirm := flags.Bool("confirm-kills", false, confirmKillsHelp)
	loud := flags.Bool("verbose", false, "print what the run is doing as it does it")
	sandbox := flags.String("sandbox", "", `how each file reaches the sandbox: "copy" (default), "hardlink" or "link"`)

	var exclude excludes

	flags.Var(&exclude, "exclude-prefix", "repository-relative prefix never worth mutating; repeatable")

	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage of ditto changed:")
		flags.PrintDefaults()
		fmt.Fprint(os.Stderr, changedConfigHelp)
	}

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		return fmt.Errorf("reading the flags: %w", err)
	}

	if *since == "" {
		return errNoBaseRef
	}

	if *threshold < 0 || *threshold > 1 {
		return fmt.Errorf("--threshold is %.2f, and a mutation score is between 0 and 1", *threshold) //nolint:err113 // the number is the message
	}

	if *dry {
		return reportChangedPlan(*directory, *since, exclude, out)
	}

	options := stagedOptions(*testCommand, float32(*threshold), *gated, *confirm, *loud, *sandbox)

	return ditto.RunChanged(*directory, *since, exclude, options...) //nolint:wrapcheck // this is the top of the program: the message is already the one a reader needs
}

// errNoBaseRef refuses to guess. There is no default that is right in a CI
// checkout and in a working tree and on a branch, and a scope guessed wrong is
// either a bill nobody asked for or a green that measured nothing.
var errNoBaseRef = errors.New("ditto changed needs --since, for example: ditto changed --since origin/main")

// changedConfigHelp names what `-h` would otherwise not say.
const changedConfigHelp = `
The scope is ` + "`--since <ref>...HEAD`" + `, the diff against their merge base, so a
base that has moved on does not drag somebody else's commits into the bill.

The checkout must have no uncommitted work in it. A range scope names bytes of
HEAD and the sandbox is written from the index; those are the same tree only
while nothing is modified or staged.

.ditto.json works here exactly as it does for ` + "`staged`" + `.
`

// reportChangedPlan answers what a committed change would cost without paying
// for it.
func reportChangedPlan(directory, baseRef string, exclude excludes, out io.Writer) error {
	plan, err := ditto.PlanChanged(directory, baseRef, exclude)
	if err != nil {
		return fmt.Errorf("reading the change: %w", err)
	}

	if !plan.Mutable() {
		fmt.Fprintf(out, "ditto: nothing changed since %s is worth mutating.\n", baseRef)

		return nil
	}

	fmt.Fprintf(out, "ditto: %d file(s) changed since %s under %s\n", len(plan.Files), baseRef, plan.Root)

	for _, file := range plan.Files {
		fmt.Fprintf(out, "  %s: %s\n", file, describeRanges(plan.Ranges[file]))
	}

	if !plan.Derived {
		fmt.Fprintf(out, "  scope: %s\n", plan.Reason)
	}

	return nil
}

// reportPlan answers what a staged change would cost without paying for it. A
// dry run that materialised a sandbox or started a suite would not be one.
func reportPlan(directory string, exclude excludes, out io.Writer) error {
	plan, err := ditto.PlanStaged(directory, exclude)
	if err != nil {
		return fmt.Errorf("reading the staged change: %w", err)
	}

	if !plan.Mutable() {
		fmt.Fprintln(out, "ditto: nothing staged is worth mutating.")

		return nil
	}

	fmt.Fprintf(out, "ditto: %d staged file(s) under %s\n", len(plan.Files), plan.Root)

	for _, file := range plan.Files {
		fmt.Fprintf(out, "  %s: %s\n", file, describeRanges(plan.Ranges[file]))
	}

	if !plan.Derived {
		fmt.Fprintf(out, "  scope: %s\n", plan.Reason)
	}

	return nil
}

func describeRanges(ranges []ditto.Range) string {
	if len(ranges) == 0 {
		return "whole file"
	}

	spans := make([]string, 0, len(ranges))
	for _, span := range ranges {
		spans = append(spans, fmt.Sprintf("%d-%d", span.Start, span.End))
	}

	return strings.Join(spans, ",")
}

// testCommandHelp is the description of --test-command, and it names the COST
// rather than only the behaviour.
//
// The old wording was accurate and useless. It said what the flag does and what
// its default is, and a reader who understood every word still ran the default
// against `./...` on a repository whose suite takes 27 seconds -- which is that
// suite once per mutant, sequentially, and looks exactly like a hang because
// ditto used to print nothing while it happened. Two repositories wrote the
// missing sentence into their own docs rather than getting it from here.
//
// Naming it in the flag help is the same decision stagedConfigHelp already made
// below, for the same reason: `-h` is where a reader looks at the moment they
// are deciding what to type, and the readme is not.
// The first backquoted word is not decoration: flag.PrintDefaults renders it as
// the flag's VALUE NAME. Naming anything else there is how this line used to
// render as `-test-command -json`, which reads like a second flag.
const testCommandHelp = "the `command` that decides whether a mutant died. It runs ONCE PER MUTANT, " +
	"sequentially, so ./... costs your whole suite times your mutant count -- name the package " +
	"that owns the change instead. -json is what lets ditto say WHY a mutant died; without it a " +
	"mutant that never compiled is counted as killed"

// confirmKillsHelp names the cost as well as the behaviour, for the reason
// testCommandHelp does.
const confirmKillsHelp = "re-run a mutant that died by assertion, once, and believe the second answer " +
	"when it disagrees. Doubles the cost of every assertion kill, and buys nothing on a suite that " +
	"does not flake -- but the baseline check runs once per release, so a suite that goes red at " +
	"mutant 37 otherwise scores a kill no test earned"

// stagedConfigHelp names the one thing about `staged` that is not a flag.
//
// The sandbox is built from the index, so a repository that does not build from
// its index alone needs to say what git is missing. There is nowhere in
// PrintDefaults for that, and a reader checking `-h` would otherwise conclude
// the flags are the whole contract.
const stagedConfigHelp = `
A repository that does not build from its index alone can name the generated
paths git does not carry, in a .ditto.json at its root:

    {"generated": ["frontend/dist", "frontend/wailsjs"]}

They are copied from the working tree after the index is materialised, and each
copy is announced. Naming a path git tracks is refused: the index version is the
one a staged run measures.
`

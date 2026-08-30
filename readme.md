<h1 align="center">
<a href="https://github.com/Disble/ditto">
	<img src=".assets/logo.png" alt="ditto logo" width="147" height="100">
</a>
</h1>

<p align="center">
	<a href="https://pkg.go.dev/github.com/Disble/ditto"><img src="https://pkg.go.dev/badge/github.com/Disble/ditto.svg" alt="Go Reference"></a>
	<a href="https://goreportcard.com/report/github.com/Disble/ditto"><img src="https://goreportcard.com/badge/github.com/Disble/ditto" alt="Go Report Card"></a>
	<a href="https://github.com/Disble/ditto/actions/workflows/ci.yml"><img src="https://github.com/Disble/ditto/actions/workflows/ci.yml/badge.svg" alt="CI Workflow"></a>
	<a href="https://github.com/Disble/ditto/actions/workflows/mutation.yml"><img src="https://github.com/Disble/ditto/actions/workflows/mutation.yml/badge.svg" alt="Mutation Testing Workflow"></a>
</p>

## What this fork is

Ditto is a fork of [gtramontina/ooze](https://github.com/gtramontina/ooze), whose
last release was v0.3.1 in May 2023. All the good ideas here are Guilherme J.
Tramontina's, and the licence and copyright stay with him.

The fork exists to push on two things, in this order:

**Fast.** Mutation testing is usually run in CI, once, on everything. Ditto is
built for the other place: inside a TDD loop, on staged changes, while you are
still writing the code. That means the target is not throughput on a build
server, it is latency on the laptop you are also editing in — and it means
deliberately _not_ buying speed with parallelism that makes the machine
unusable.

**Well made.** A fast answer that is wrong is worse than no answer, because you
act on it. Ditto treats a misleading verdict as a defect, not a caveat. The
prerequisite two sections down — that a failing suite makes every mutant look
killed — is the clearest example: it is documented upstream as something to be
careful about, and it is on this fork's list to refuse outright.

### Performance is the metric

For a library whose reason to exist is being cheap enough to run, "it got
slower" is the same statement as "it stopped working". So the cost is recorded
rather than remembered:

- `perf/baseline.json` holds exact counters — files linked per mutant, parses
  per release, test-command runs per release. Integers, identical on every
  machine, unaffected by whatever else the machine is doing.
- `internal/perfbench` enforces them. A counter that grows fails the build. A
  counter that shrinks also fails the build, until the gain is written down, so
  an improvement cannot be quietly handed back later.
- Wall clock is measured and reported, never gated. On a real development
  machine the same workload has varied here by more than fifty percent between
  runs; a threshold tight enough to catch a regression would fire on the
  weather, and a gate that cries wolf is a gate people learn to ignore.

Every claim about speed in this repository should come with the number that
would disprove it.

## Mutation Testing?

Mutation testing is a technique used to assess the quality and coverage of test suites. It involves introducing controlled changes to the code base, simulating common programming mistakes. These changes are, then, put to test against the test suites. A failing test suite is a good sign. It indicates that the tests are identifying mutations in the code—it "killed the mutant". If all tests pass, we have a surviving mutant. This highlights an area with weak coverage. It is an opportunity for improvement.

There are different types of changes that mutation tests can perform. A common collection usually include:

- Changing an operator;
- Replacing a constant;
- Removing a statement;
- Increasing/decreasing numbers;
- Flipping booleans;

Mutations can also be domain/application-specific. Although, these are up to the maintainers of such application to develop.

It is worth mentioning that mutation tests can be quite expensive to run. Especially on larger code bases. And the reason is that for every mutation, on every source file, the entire suite of tests has to run. One can look at the bright side of this and think as an incentive to keep the test suites fast.

Mutation testing is a great ally in developing a robust code base and a reliable set of test suites.

## Quick Start

### Prerequisites

Make sure the test suite Ditto will run is passing. It no longer has to be taken on trust: Ditto runs the suite once on unmutated code before scoring anything, and refuses the run if it is already red. That guard exists because the alternative was measured — a red baseline used to report **431 of 431 mutants killed in 5.46 seconds**, a perfect score for a run that compiled nothing. It costs one invocation per release, never one per mutant.

When Ditto reports that it found a living mutant, it will print a diff of the changes the virus made to the source file. The mutant source is printed using Go's [`go/format`](https://pkg.go.dev/go/format) package. This means that, if your source code isn't gofmt'd, the diff may contain some formatting changes that are not relevant to the mutation. This isn't a prerequisite per se, but for a better experience, it is recommended that you run `gofmt` on your source files.

### From the command line

Ditto also runs as a command, which is the shorter way in when what you want is
a gate rather than a test:

```shell
go install github.com/Disble/ditto/cmd/ditto@latest

ditto version                             # which build is this? (also -v, --version)
ditto run --threshold 0.8                 # mutate the repository
ditto staged --dry                        # what would a staged change cost?
ditto staged --threshold 0.8              # mutate only what it justifies
ditto changed --since v1.2.0 --dry        # and the same, for a change already committed
```

`ditto staged` reads the change you have staged and nothing else: which files it
touches, which of their bytes, and — the part that is easy to skip — it runs the
suite against a checkout of the **index**, not of your working tree.

That last one is not caution, it is the difference between two answers. Measured
on a fixture built for it: with the release pointed at the worktree instead, and
one tracked file left dirty and unstaged, **seven of eight verdicts moved** — a
score of 0.13 against 1.00 for the identical eight mutants of an identical file.
Checking that the staged files themselves are clean does not cover it, because
the file that moved them was never staged.

Policy stays with you: `--threshold`, `--test-command`, and `--exclude-prefix`
(repeatable) are yours to set, and Ditto has an opinion about none of them
beyond its defaults.

**Name the package that owns the change in `--test-command`.** This is the one
default that will surprise you, so it is here rather than only in `-h`: the test
command runs **once per mutant, sequentially**, so the default `./...` costs your
whole suite times your mutant count. Reported from a repository whose suite takes
27 seconds across 45 packages: a run that printed nothing for over ten minutes,
twice, and was read as a hang. It was not stuck. It was paying that bill.

```shell
ditto staged --test-command "go test -count=1 -json ./internal/thepackage/"
```

Seconds instead. `--exclude-prefix` and a `--threshold` below 1.00 are the other
two levers, and the sandbox strategy is not one — `--sandbox hardlink` buys back
a fixed fifteen seconds on a two-thousand-file repository, and `--sandbox link`
cannot work at all in a repository with a `go:embed` directive, because embed
refuses an irregular file.

Keep `-json`. It is what lets ditto tell a mutant that never compiled from one a
test caught; without it, the first is counted as the second.

### In CI, where nothing is staged

`ditto staged` reads the index, and a CI checkout has nothing in it: the change
is already committed. A gate pointed at the staged scope there skips, reports
success, and measures nothing.

`ditto changed --since <ref>` asks the same question of `<ref>...HEAD` — the diff
against their merge base, so a base that has moved on does not drag somebody
else's commits into the bill.

```shell
ditto changed --since "$(git describe --tags --abbrev=0 HEAD^)" --threshold 0.8
```

It refuses a checkout with **staged** changes in it, and that refusal is the
point rather than fussiness: a range scope names bytes of `HEAD` while the
sandbox is written from the index, and those agree exactly while the index
agrees with HEAD. A worktree-only modification is never written into the sandbox
and is allowed.

There is no default base. On a CI checkout the useful one is the last release, on
a branch it is the trunk, and a base guessed wrong is either a bill nobody asked
for or a scope of nothing reported as green.

This is how **ditto's own gate** runs. The repository-sized question — 783
mutants — died at its thirty minutes having reached about 424, four times over,
and both levers were spent: gating removes 54% of the compilations and does not
close it, and cutting the mutant's suite by 46% moved the gate by 0.5%. The bill
was the wrong size rather than badly paid.

### When the index is not the whole story

A sandbox is built from the index, and that is deliberate. Some repositories do
not build from their index alone: a generated directory the build needs — an
embedded frontend bundle, generated bindings — is on disk and not in git, so the
sandbox arrives without it and the package that needs it cannot compile.

Name those paths, one at a time, in a `.ditto.json` at the repository root:

```json
{
  "generated": ["frontend/dist", "frontend/wailsjs"]
}
```

They are copied from the working tree after the index is materialised, and the
run says which ones — they are the only bytes in the sandbox that did not come
from git, and a reader deciding whether to trust a verdict deserves to know.

This does not widen what is read. **Naming a path git tracks is refused**, not
obeyed: a tracked path has an index version and that is the one a staged run
measures. A path that is not there is refused too, rather than skipped — it was
named because the build needs it.

### Installation as a library

1. Install ditto:

   ```shell
   go get github.com/Disble/ditto
   ```

   This pulls the latest version of Ditto and updates your `go.mod` and `go.sum` to reference this new dependency.

2. Create a `mutation_test.go` file in the root of your repository and add the following:

   ```go
   //go:build mutation

   package main_test

   import (
   	"testing"

   	"github.com/Disble/ditto"
   )

   func TestMutation(t *testing.T) {
   	ditto.Release(t)
   }
   ```

   The build tag is so you can better control _when_ to run these tests (see the next step). This is a test as you'd write any other Go test. What differs is what the test actually does. And this is where it delegates to Ditto, by `Release`ing it.

3. Run with:

   ```shell
   go test -v -tags=mutation
   ```

   This will execute all tests in the current package including the sources tagged with `mutation`. This assumes that the above is the only test file in the root of your project. If you have other tests, you may want to put the mutation tests in a separate package, under `./mutation` for example, and configure Ditto to use `..` as the repository root (see [`WithRepositoryRoot`](#Settings) below).

   If `-v` is enabled, Ditto will also be verbose. To enable Ditto's verbose mode only without the test framework verbosity, use `-ditto.v`.

   > **Note**
   > printing to `stdout` while Go tests are running has its intricacies. Running the tests at a particular package (without specifying which test file or subpackages, like `./...`), allows for Ditto to print progress and reports as they happen. Otherwise, the output is buffered and printed at the end of the test run and, in some cases, only if a test fails. This is a limitation of Go's testing framework.

### Results

**A mutant that never compiled is not in the score.** A failing compile exits
non-zero, which is how ditto recognises a kill, so those used to be counted as
caught by tests that never ran them. They now leave the numerator *and* the
denominator, and the report says how many there were and which mutation operator
produced them:

```
┃ • Killed:       1
┃ ✓ Score:     0.33 (minimum: 0.00)
┃ 2 of the 5 mutants generated never compiled, and are out of the score entirely.
┃   1 from Integer Decrement
┃   1 from Integer Increment
```

Measured on that fixture, the old number was **0.60**. If a threshold was tuned
against a score from before this, expect it to move.

**Read the survivors, not the number.** The score is a summary of your tests, and
a summary is all it is: Google abandoned the ratio in IEEE TSE 2021 as "neither
concrete nor actionable", fewer than 5% of mutants carry the signal, and below a
high score the number is disconnected from fault revelation altogether. The part
of this report anybody acts on is the survivor list, one line per mutant with its
address. See [`docs/metrics.md`](docs/metrics.md) for what ditto measures about
itself instead, and why.

Once all tests on all mutants have run, Ditto will print a report with the results. It will also exit with a non-zero exit code if the mutation score is below the minimum threshold (see [`WithMinimumThreshold`](#Settings) below). This is an example of the report, exactly as ditto prints it — the byte-for-byte
content of [`testdata/golden/release.txt`](testdata/golden/release.txt), which
`TestReleaseGolden` compares a whole release against on both the ordinary and the
gated path. It is text rather than a screenshot for one reason: a screenshot goes
stale in silence, and this cannot — if the report changes, the golden test fails.

```text
┃ Releasing Ditto…
┃ calc/calc.go — 7 mutants
┃   calc/calc.go:9:12 → Arithmetic
┃ baseline: the suite took 796ms on unmutated code, and every mutant runs it again.
┃   calc/calc.go:12:11 → Arithmetic
┃   calc/calc.go:16:41 → Arithmetic
┃   calc/calc.go:4:41 → Comparison
┃   calc/calc.go:8:8 → Comparison
┃   calc/calc.go:4:40 → Comparison Invert
┃   calc/calc.go:8:7 → Comparison Invert
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╍┅
┃ 🧬 Survivors
┠┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄
┃ calc/calc.go:12:11 → Arithmetic (- → +)
┃ calc/calc.go:16:41 → Arithmetic (+ → -)
┃ calc/calc.go:8:8 → Comparison (inserts =)
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╍┅
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╍┅
┃ 🧬 Mutant survived: calc/calc.go:12:11 → Arithmetic
┠┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄
┃ --- calc/calc.go (original)
┃ +++ calc/calc.go (mutated with 'Arithmetic')
┃ @@ -9,7 +9,7 @@
┃  		return a - b
┃  	}
┃  
┃ -	return b - a
┃ +	return b + a
┃  }
┃  
┃  // Uncovered is never called, so every mutant of it lives.
┃ 
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╍┅
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╍┅
┃ 🧬 Mutant survived: calc/calc.go:16:41 → Arithmetic
┠┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄
┃ --- calc/calc.go (original)
┃ +++ calc/calc.go (mutated with 'Arithmetic')
┃ @@ -13,4 +13,4 @@
┃  }
┃  
┃  // Uncovered is never called, so every mutant of it lives.
┃ -func Uncovered(a, b int) int { return a + b }
┃ +func Uncovered(a, b int) int { return a - b }
┃ 
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╍┅
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╍┅
┃ 🧬 Mutant survived: calc/calc.go:8:8 → Comparison
┠┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄
┃ --- calc/calc.go (original)
┃ +++ calc/calc.go (mutated with 'Comparison')
┃ @@ -5,7 +5,7 @@
┃  
┃  // Partly is exercised from one side only, so some of its mutants live.
┃  func Partly(a, b int) int {
┃ -	if a > b {
┃ +	if a >= b {
┃  		return a - b
┃  	}
┃  
┃ 
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╍┅
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃ • Total:        7                    ┃
┃ • Killed:       4                    ┃
┃ • Survived:     3                    ┃
┠┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┨
┃ ✓ Score:     0.57 (minimum: 0.00)    ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
```

Every survivor is listed by address before any diff is rendered, as
`path:line:col → Virus (what it replaced)`, so it can be jumped to rather than
hunted for in the log.

More examples of the results can be found in the [`mutation.yml`](https://github.com/Disble/ditto/actions/workflows/mutation.yml) workflow.

## Settings

Ditto is configured in two places, and they answer different questions.

**Options** say how to run: threshold, test command, viruses, scope. They are
passed in code, or as flags by `cmd/ditto`, and they are described below.

**`.ditto.json`** says what the repository *is*: the generated paths git does not
carry, which a sandbox built from the index would otherwise be missing. It is a
property of the repository rather than of a run, which is why it lives in a file
and not in a flag — see [When the index is not the whole
story](#when-the-index-is-not-the-whole-story).

Ditto's [`Release`](release.go) method takes variadic [`Options`](options.go), like so:

```go
ditto.Release(
	t,
	ditto.WithRepositoryRoot("."),
	ditto.WithTestCommand("make test"),
	ditto.WithMinimumThreshold(0.75),
	ditto.Parallel(),
	ditto.IgnoreSourceFiles("^release\\.go$"),
)
```

The table below presents all available options.

| Option                 | Default                               | Description                                                                                                                                                                                                                                                                    |
| ---------------------- | ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `WithRepositoryRoot`   | `.`                                   | A string that configures which directory is the repository root. This is usually required when your mutation test file lives some other place that is not root itself.                                                                                                         |
| `WithTestCommand`      | `go test -count=1 -json ./...`        | The test command to run, as string. **`-json` is why the default carries it**: the event stream is the only place a failing command says whether the mutant did not compile, and a mutant that never compiled is not one a test caught. A command that does not emit it still works; ditto simply cannot name the reason and says so rather than guessing. You may configure it as you wish, as a `makefile` phony target, for example. Or simply run the standard `go test` command with extra flags, such as `timeout` and `tags`.                                                                  |
| `WithMinimumThreshold` | `1.0`                                 | A float between `0.0` and `1.0`. This represents the minimum mutation test score to consider the execution successful.                                                                                                                                                         |
| `Parallel`             | `false`                               | Indicates whether to run the tests on the mutants in parallel. Given Ditto is executed via Go's testing framework, the level of parallelism can be configured when running the mutation tests from the command line. For example with `go test -v -tags=mutation -parallel 3`. |
| `IgnoreSourceFiles`    | `nil`                                 | Regular expression representing source files to be filtered out and not suffer any mutations.                                                                                                                                                                                  |
| `WithViruses`          | all available ([see below](#Viruses)) | A list of viruses to infect the source files with. You can also implement your own viruses (generic or even application-specific).                                                                                                                                             |
| `ForceColors`          | `false`                               | Forces colors in logs. This is useful when running the mutation tests in a CI environment, for example.                                                                                                                                                                        |
| `WithChangedRanges`    | `nil`                                 | Restricts the release to named byte ranges of named files, keyed by repository-relative path with forward slashes. Every mutant costs a full run of the test command, so mutating a line a change never touched is charged at the same rate as one that matters. A file with no entry is not mutated at all; a file with an empty range list is mutated whole. Keep the ranges beside their file — a byte offset only means something against the file it was measured in, and one flat set makes every file answer to every range. |
| `Gated`                | `false`                               | Runs a file's mutants from one compilation instead of one each: the file is instrumented so every mutant becomes a gate chosen at run time, the package is compiled once with `go test -c`, and each mutant is selected by environment variable. Starting the test command costs 750–950 ms per mutant regardless of what the suite does, and that fixed toll is the dominant cost of a run. Anything it cannot express that way keeps the path ditto has always taken, so no mutant is lost by turning it on. It stays opt-in because the gating rate is a property of the file — measured between 26% and 72% across real files — and a compilation paid for a quarter of the mutants is an option rather than a default. |
| `ConfirmKills`         | `false`                               | Re-runs a mutant that died by assertion, once, and believes the second answer when it disagrees. The baseline check runs once per release, so it refuses a suite that is ALREADY red and cannot see one that goes red at mutant 37 — where a spurious failure becomes a kill no test earned, indistinguishable from a real one in the report. Only assertion kills are re-run: a mutant that never built already leaves the score on both sides, and a deadline is a clock ditto fired itself. A survivor is never re-run, because a flake manufactures failures and cannot make a mutant the tests caught look like it escaped. Off by default: it doubles the price of every assertion kill and buys nothing on a suite that does not flake. |
| `WithSandboxStrategy` | `copy` | How each file reaches a sandbox: `copy`, `hardlink` or `link`. A sandbox is only a sandbox if what it holds is a copy: a symlink is a reference that `go:embed` refuses and that a write follows through to the original, and a hard link shares the inode so a write reaches it too. `link` is what every release before 0.5.0 used and stays reachable for measurement. Whatever the strategy, a **symlink already in the repository** is reproduced as the same link with its raw target, never followed — the tree ditto measures is the tree on disk. |

## Viruses

| Virus                                                                                            | Name                         | Description                                                                                                                                                                                                                     |
| ------------------------------------------------------------------------------------------------ | ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [`arithmetic`](viruses/arithmetic/arithmetic.go)                                                 | Arithmetic                   | Replaces `+` with `-`, `*` with `/`, `%` with `*` and vice versa.                                                                                                                                                               |
| [`arithmeticassignment`](viruses/arithmeticassignment/arithmeticassignment.go)                   | Arithmetic Assignment        | Replaces `+=`, `-=`, `*=`, `/=`, `%=`, `&=`, <code>&#124;=</code>, `^=`, `<<=`, `>>=` and `&^=` with `=`.                                                                                                                       |
| [`arithmeticassignmentinvert`](viruses/arithmeticassignmentinvert/arithmeticassignmentinvert.go) | Arithmetic Assignment Invert | Replaces `+=` with `-=`, `*=` with `/=`, `%=` with `*=` and vice versa.                                                                                                                                                         |
| [`bitwise`](viruses/bitwise/bitwise.go)                                                          | Bitwise                      | Replaces `&` with <code>&#124;</code>, <code>&#124;</code> with `&`, `^` with `&`, `&^` with `&`, `<<` with `>>` and `>>` with `<<`.                                                                                            |
| [`cancelnil`](viruses/cancelnil/cancelnil.go)                                                    | Cancel Nil                   | Changes calls to [`context.CancelCauseFunc`](https://pkg.go.dev/context#CancelCauseFunc) to pass nil.                                                                                                                           |
| [`comparison`](viruses/comparison/comparison.go)                                                 | Comparison                   | Replaces `<` with `<=`, `>` with `>=` and vice versa.                                                                                                                                                                           |
| [`comparisoninvert`](viruses/comparisoninvert/comparisoninvert.go)                               | Comparison Invert            | Replaces `>` with `<=`, `<` with `>=`, `==` with `!=` and vice versa.                                                                                                                                                           |
| [`comparisonreplace`](viruses/comparisonreplace/comparisonreplace.go)                            | Comparison Replace           | Replaces the left and right sides of an `&&` comparison with `true` and the left and right sides of an <code>&#124;&#124;</code> with false. E.g. `1 == 1 && 2 == 2` gets two mutations: `true && 2 == 2` and `1 == 1 && true`. |
| [`floatdecrement`](viruses/floatdecrement/floatdecrement.go)                                     | Float Decrement              | Decrements floating points by `1.0`.                                                                                                                                                                                            |
| [`floatincrement`](viruses/floatincrement/floatincrement.go)                                     | Float Increment              | Increments floating points by `1.0`.                                                                                                                                                                                            |
| [`integerdecrement`](viruses/integerdecrement/integerdecrement.go)                               | Integer Decrement            | Decrements integers by `1`.                                                                                                                                                                                                     |
| [`integerincrement`](viruses/integerincrement/integerincrement.go)                               | Integer Increment            | Increments integers by `1`.                                                                                                                                                                                                     |
| [`loopbreak`](viruses/loopbreak/loopbreak.go)                                                    | Loop Break                   | Replaces loop `break` with `continue` and vice versa.                                                                                                                                                                           |
| [`loopcondition`](viruses/loopcondition/loopcondition.go)                                        | Loop Condition               | Replaces loop condition with an always false value.                                                                                                                                                                             |
| [`rangebreak`](viruses/rangebreak/rangebreak.go)                                                 | Range Break                  | Adds an early break to `range`s.                                                                                                                                                                                                |

### Custom viruses

Ditto's viruses follow the [`viruses.Virus`](viruses/virus.go) interface. All it takes to write a new virus is to have a struct that implements this interface. To get this new virus running, let Ditto know about it by running `Release` with the `WithViruses(…)` option. In order to test it, you may want to use the [dittotesting](dittotesting) package to help out. Take a look at the existing [viruses](viruses) to have an idea.

If your new virus is domain-agnostic, and you find it useful, consider contributing it to this project. You can also write domain-specific viruses. One that looks for a particular struct type and change it in a particular way, for example.

## Tips

1. Ditto runs your test suite for every mutant it creates. Having a fast suite is a good idea. The way Ditto detects that a mutant was killed is by having a failing test. The quicker your suite catches the faster the mutation testing will finish. Go testing framework allows for us to flag it to fail fast with `-failfast`. Although this is better than nothing, this doesn't work across packages (see this [issue](https://github.com/golang/go/issues/33038) for more details). This is where [gotestsum](https://github.com/gotestyourself/gotestsum) comes in. It allows us to fail even faster by configuring it with `--max-fails=1`.
2. Mutation testing usually takes a significant amount of time to run. Especially if you have a large codebase. It may be a good approach to run it on a separate path on your CI pipeline; preferably after you get confirmation that your test suite is passing. This way you can get the results of the mutation testing without slowing down your main pipeline.
3. Ditto runs itself. I recommend exploring this codebase to get a better idea of how to use it.

## Prior Art

Ditto is heavily inspired by [go-mutesting](https://github.com/zimmski/go-mutesting), by [@zimmski](https://github.com/zimmski), and by extra mutations added to a [fork](https://github.com/avito-tech/go-mutesting) by [@avito-tech](https://github.com/avito-tech).

You can find more resources and tools on this subject by browsing through the [mutation testing](https://github.com/topics/mutation-testing) topic on GitHub. The [awesome-mutation-testing](https://github.com/theofidry/awesome-mutation-testing) repository also contains many good resources.

## License

Ditto is open-source software released under the [MIT License](LICENSE).

---

<a href="https://github.com/Disble/ditto">
	<img src=".assets/icon.png" width="24" align="right" alt="ditto icon">
</a>

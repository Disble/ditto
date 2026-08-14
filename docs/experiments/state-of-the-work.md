# State of the work — the gated path

Written to be read by someone with no memory of any of it. Everything here is
measured unless it says otherwise, and where it says otherwise that is the point.

## The one-sentence version

Ditto pays a fixed 750–950 ms to start `go test` for **every** mutant, which is
most of a run; the gated path instruments a file once, compiles once, and selects
each mutant at run time, cutting 135 invocations to 38 on a real file; it is
wired, opt-in, **measured backward-compatible**, and it does not engage at all
under `go test -v`.

## Where everything is

| | |
| --- | --- |
| Branch | `exp/test-invocation`, 42 commits on top of `dfd8ed4`, **not pushed** |
| Worktree | `D:/dev/disble/ditto-worktrees/test-invocation` |
| dharness skill branch | `docs/measurement-skill`, four commits, **not pushed** |
| Experiment notes | 19, in `docs/experiments/` |

The probes live in the repository and are skipped unless asked for by name. See
*How to run any of it again*.

## What is proven

- **The invocation toll is the cost.** 750–950 ms per mutant regardless of what
  the suite does. `test-invocation.md`.
- **Compiling once and selecting at run time reaches the same verdicts** for the
  families it admits. `schemata-feasibility.md`, `gating-non-bool.md`,
  `instrumentation-fidelity.md`.
- **97 of 135 mutants gate on a real file** — 71.9%. `gated-gain-real.md`.
- **The gain is real and unchanged by the fixes**: 135 invocations of the test
  command against **38**, reproduced a day apart by two instruments built
  independently. `gain-after-the-fixes.md`.
- **The branch changed no verdict the ordinary path used to give.** 31 mutants of
  a scratch fixture and 35 of `dharness/internal/setup/writer.go`, compared
  mutant by mutant against `dfd8ed4`: nothing moved column.
  `backward-compatibility.md`.
- **7.3% of the mutants ditto calls killed never ran**, because they do not
  compile and a failing command is how ditto recognises a kill. The bound is
  tight: `go test -c` finds the same 94 and not one more. `false-kills.md`,
  `refusing-false-kills.md`.
- **The gated path fixes that for the class it gates.** On a fixture built to
  contain it, one label differs between the paths and it is the mutant that
  leaves a local unread — survived gated, killed ordinary.
  `disagreement-class.md`.
- **A red baseline used to report a perfect score**, and no longer does.
  `changed-scope.md`.

## What is open

1. **`go test -v` turns the gated path off.** `verboselaboratory` does not
   forward `TestAll`. `backlog.md` entry 11, and the largest of these: `-v` is
   what CI runs.
2. **Nothing reports how much gated.** `Gated()` and `FellBack()` are counters
   the report never prints, so a run that gated nothing looks like one that gated
   everything. Same entry.
3. **The baseline run is counted as a mutant's run** in `GoBuildRunner.Runs`.
   Measured: 3 runs for 2 gated mutants. `changed-scope.md` H3.
4. **The false kill on the ordinary path.** Two mechanisms measured and both
   refuted — the AST reaches 61 of 94 and wrongly refuses 2 that compile; an
   in-process `go/types` check reaches 92 and refuses nothing, missing only a
   package it cannot read. `refusing-false-kills.md`, `backlog.md` entries 6
   and 10.
5. **`Gated()` is opt-in.** Backward compatibility is measured; the default is a
   decision nobody has taken.
6. **A survivor has no address.** `backlog.md` entry 1, and a third of what this
   tool exists for.

## What must be decided, not measured

**The report's vocabulary.** `Diagnostic.IsOk()` is binary and
`total = len(diagnostics)`. A mutant that cannot compile has to leave the
denominator rather than become a kill, and that changes ditto's public report,
which dharness gates on at 0.80.

## The traps that cost the most time

- **A path that was never reached.** Every gated-versus-ordinary comparison
  agreed, which read as agreement and was the gate being off. A `panic` in
  `TestAll` that did not fire found it; a second in `Test` that did fire said
  which of the two possible causes it was.
- **An instrument aimed at the wrong quantity.** A counter inside the mutated
  package counts test-*binary* executions; the gain is quoted in test-*command*
  invocations. Both correct, different questions.
- **A fixture that cannot contain the case.** `comparisonreplace` fires only on
  `&&` and `||`, so a fixture without a logical operator reported the paths
  agreeing because the class was absent, not because they agreed.
- **`cd` before `tar`**, and `tar` reading a leading `C:` as a remote host.
- **A control that fires and gets explained** is worse than no control.
- **Two heredocs in one shell invocation** run their `git add` before the first
  commit.

## How to run any of it again

The probes are skipped unless asked for:

    DITTO_PROBE=1 go test . -run TestChangedScope -v
    DITTO_PROBE=1 go test . -run TestInstrumentationFidelity -v
    DITTO_PROBE=1 go test . -run TestDisagreementClass -v
    DITTO_PROBE=1 go test . -run TestBackwardCompatibility -v

    DITTO_PROBE=1 DITTO_COMPAT_REPO=<repo> \
      go test . -run TestBackwardCompatibilityOnTheRepository -v
    DITTO_PROBE=1 DITTO_COMPAT_REPO=<repo> \
      go test . -run TestGainAfterTheFixes -v -timeout 120m

    DITTO_ADMIT_ROOT=<copy>      go test ./internal/schemata/ -run TestAdmits...
    DITTO_FALSEKILL_ROOT=<copy>  go test ./internal/schemata/ -run TestCounts...

`DITTO_COMPAT_REPO` is copied, never mutated in place. `DITTO_FALSEKILL_ONLY`
narrows the last one and takes a value with no slashes, because the shell
rewrites anything that looks like a path.

**`TestChangedScope` asserts that a defect is present.** The day one of them is
fixed it turns red for having been fixed, which is why it is not in the gate.

The repository's own gate is `CPUS=1 mingw32-make lint test.failfast` on this
machine — `make` is not on PATH, only the native `mingw32-make`, and the MSYS2
build of make loses the environment its recipes need.

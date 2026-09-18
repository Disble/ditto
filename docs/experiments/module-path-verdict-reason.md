# Experiment — what reason does a module-scope kill carry?

Written before the measurement on 2026-09-18, at revision `243f2ad`.

## The research question

**What** reason does `verdict.ReasonOf` report for a mutant the module-scope runner killed, over the two kill kinds the score depends on, in a disposable module at `243f2ad` on 2026-09-18?

| Element | This question's |
| --- | --- |
| Interrogative phrase | What |
| Variable, the counter that moves | The `verdict.Reason` a captured kill produces |
| Population, unit of analysis | Two kills: one by assertion, one a package build failure |
| Space and time | Revision `243f2ad`, a disposable two-package module, 2026-09-18 |

**FINER** — Feasible: the runner returns its captured output and `verdict` is a pure function of it. · Interesting, the decision that turns on it: whether the module path may carry `--confirm-kills` and whether it can report a reason at all. · Novel: nothing has read a reason off the module path; `gobuildrunner` runs a test binary, and no test binary emits `go test -json`. · Ethical: both fixture and tool are throwaway copies. · Relevant: `internal/confirminglaboratory` re-runs a kill only when the reason is `Assertion`, and `internal/consolereporter` removes `BuildFailed` from both sides of the score.

**PICOT** — P: two kills, one per mode. · I: the module-scope runner's own captured output, and the same output passed through `go tool test2json`. · **C, the control**: the ordinary `go test -count=1 -json ./...` over the same fixture and the same mutant. · **O, the exact counter**: the `verdict.Reason` value, one of four strings. · T: one run per mode, bound to the revision above.

## Why this is not a style question

`verdict.ReasonOf` reads the JSON stream that `go test -json` emits. A package test binary started directly has no such stream to emit: `-json` is a flag of the `go test` driver, not of the binary. So the prediction is that every kill on the module path arrives as `Unknown`.

That would matter three ways, in order of severity:

1. `internal/confirminglaboratory` re-runs a kill **only** when the reason is `Assertion`. If the module path reports `Unknown`, `--confirm-kills` becomes a silent no-op exactly where gating is used — a false kill can no longer be caught.
2. A reader loses the distinction between a mutant a test killed and a mutant that never became a program.
3. Any later rule keyed on the reason stops applying to the path.

The package-only gated runner has had the same gap since it was written. It matters more now because the module path is what the default command takes.

## Hypotheses, and what kills each one

**H1 — a module-scope assertion kill arrives as `Unknown`.** The ordinary control reports `Assertion` for the same fixture and the same mutant; the module path reports `Unknown`.
*Falsified if the module path reports anything other than `Unknown`.*

**H2 — a build failure never reaches the reason at all on the module path.** `ModuleScopeRunner.prepare` fails closed on a non-compiling package, so `Built()` is false, no binary runs, and the batch falls back to the ordinary laboratory — which then produces `BuildFailed` from `-json`. The module path therefore reports no reason of its own for it.
*Falsified if a build failure on the module path produces a captured kill with a reason.*

**H3 — passing the binary's output through `go tool test2json` restores the reason.** With `test2json -t -p <package>` in the middle, `ReasonOf` reports `Assertion` again for the assertion kill.
*Falsified if `ReasonOf` still reports `Unknown`, or if `test2json` is not available in the toolchain.*

**What would refute all of them:** the control does not report `Assertion`. That means the instrument is wrong, not that the module path is fine.

## Decision rule, fixed in advance

- H1 corroborated and H3 corroborated → implement `test2json` in the module runner, and add a guard that the module path reports `Assertion` for an assertion kill.
- H1 corroborated and H3 refuted → the module path keeps `Unknown`, and the limit is written where a reader looks: `--confirm-kills` does not apply to it, and the report says so rather than implying a reason it does not have.
- H1 refuted → nothing to fix; record what the path actually reports and close the question.

## Method

The fixture is a disposable two-package module created inside the test. `subject`'s test fails when `DITTO_MUTANT` is `1`, which is the variable the runner already sets, so the same fixture answers all three modes through the one channel the product uses.

The control is the ordinary command, `go test -count=1 -json ./...`, run in the same fixture with the same variable set. The reason is read from the captured output by `verdict.ReasonOf`, which is the product's own function — not a re-implementation of it.

`go tool test2json` is measured as a ceiling, not as the fix: the experiment runs it by hand over the runner's captured output, so the result says what the technique could buy before any production code changes shape.

## Results

Measured 2026-09-18 at revision `243f2ad`, from a complete `.git`-free disposable copy of the tree, Go 1.27, windows/amd64. The experiment test is tagged `experiment` and is not part of the default suite.

| Mode | Captured reason |
| --- | --- |
| Control: ordinary `go test -count=1 -json ./...` | `assertion` |
| Module path: the runner's own captured output | `unknown` |
| Module path passed through `go tool test2json` | `assertion` |
| Module path, package that does not compile | no reason: `Built()` is false and the build diagnostic is the output |

### Controls

**The instrument works — passed.** The ordinary command reported `assertion` for the same fixture and the same active mutant. Had it not, the module results would have described a broken probe rather than a path.

**A kill is actually present — passed.** The module-runner subtest fails rather than reporting a survivor when the selected mutant is not killed, so `unknown` was read from a real kill and not from an empty result.

**The compile case is a different path — passed.** The non-compiling fixture left `Built()` false with the compiler's own diagnostic, and no package binary started. That is a fallback, not a kill carrying a reason.

### H1 corroborated

A module-scope assertion kill arrives as `unknown`. The captured output is plain test-binary text — `--- FAIL: TestKilledByTheMutant (0.00s)` and a `FAIL` line — which carries no stream for `ReasonOf` to read, because `-json` belongs to the `go test` driver and not to the binary it starts.

### H2 corroborated

A package that does not compile never reaches a reason on the module path: `prepare` refuses before any binary starts, and the batch falls back to the ordinary laboratory. The failure is fail-closed, which is the property BACKLOG entry 6 wanted, and it is not the same question as the reason a kill carries.

### H3 corroborated

`go tool test2json -t -p <package>` restores the stream, and `ReasonOf` reports `assertion` again. The technique is available in the toolchain ditto already resolves to an absolute path, so it needs no new dependency and no re-implementation of a tool that exists.

## Verdicts: 3 of 3

## Conclusion

The prediction held. The module path reported a reason of `unknown` for every kill, which means `internal/confirminglaboratory` never re-runs a module-path kill — it re-runs one only when the reason is `Assertion` — so `--confirm-kills` was a silent no-op on exactly the path this change promotes. The decision rule selects the first outcome: put `go tool test2json` in the module runner and add a guard that a module-path assertion kill reports `Assertion`.

## What the pipeline costs, and what is still not covered

`test2json` is one more process per package execution. The package executions themselves do not move, and the guard for that is the counter the experiment already keeps. Its wall-clock cost is measured separately and reported rather than gated.

A **deadline** kill is still not covered, and it is not the same defect. The ordinary path marks one with a sentence its own runner writes when ditto fires the clock; on the module path the clock belongs to the test binary's `-test.timeout`, whose panic text would convert into an `Assertion`. That is a separate question with a separate fixture and is left open deliberately rather than implied to be fixed by this one.

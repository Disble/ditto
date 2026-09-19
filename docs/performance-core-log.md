# Ditto core performance log

Append-only scientific record for the core-performance work requested on 2026-09-18.

This file exists so a later session or a different model can continue from evidence rather than reconstructing intent. It records advances, regressions, refuted assumptions, unresolved questions, and the exact observation that changed each decision.

## Record contract

- Append new entries at the bottom; never rewrite an earlier entry to make the history look cleaner.
- Bind every entry to a date, source revision, and model when known.
- Separate observation, hypothesis, intervention, and conclusion.
- Exact counters decide performance contracts. Wall clock is reported, never used alone as a gate.
- A regression or refuted hypothesis is a result and remains in the record.
- No conclusion is promoted to production until its control has run and its failure path has been observed.
- The repository file is canonical. Engram mirrors it for cross-session recall under topic key `ditto/performance-core-log`.

## Entry template

```markdown
## YYYY-MM-DD — NNN — short title

- Status: advance | regression | correction | blocked | decision
- Revision:
- Model:
- Question:
- Prior hypothesis:
- Intervention:
- Control:
- Exact evidence:
- Wall-clock observation:
- Verdict:
- What changed:
- What remains unknown:
- Next falsifiable step:
- Artifacts:
```

## 2026-09-18 — 001 — Preserve the incumbent baseline

- Status: advance
- Revision: `5f65e3d3c4201689b81b707533b18aa42364ea8b`
- Model: `gpt-5.6-sol`
- Question: What does the current core cost before any architectural change?
- Prior hypothesis: Repeated test-command execution, not parsing or sandbox construction, is the dominant removable cost.
- Intervention: None; this entry records the incumbent before improvement.
- Control: A complete `.git`-free disposable copy reproduced `go test ./internal/perfbench/` with exit code 0. Source and copied `perf/baseline.json` shared SHA-256 `c470245e542046aac1e486482a9338d002314573ff0393e6f5749de442e1a23d`.
- Exact evidence:
  - source parses with three viruses: 4
  - AST walks with three viruses: 12
  - laboratory runs over the whole fixture: 48
  - test-command invocations over the whole fixture: 49
  - files linked per sandbox: 6
  - sandboxes built per sequential release: 1
  - laboratory runs for one changed function: 4
  - laboratory runs for one changed function in each of two files: 8
  - mutants in a full release over this repository: 789
- Wall-clock observation: The baseline gate completed in 0.518 s; this is environmental evidence, not a contract.
- Verdict: Baseline reproduced. The architecture should target repeated command/driver starts; parsing and sandbox creation are already small.
- What changed: Nothing in production.
- What remains unknown: The faithful cost of replacing package-only gating with complete module-scope execution.
- Next falsifiable step: Compare ordinary, package-only, and module-scope execution over the same controlled mutant population.
- Artifacts: `perf/baseline.json`, `internal/perfbench/`, `docs/experiments/module-scope-runner.md`.

## 2026-09-18 — 002 — Package-only gating asks the wrong question

- Status: correction
- Revision: `5f65e3d3c4201689b81b707533b18aa42364ea8b` plus the uncommitted experiment harness
- Model: `gpt-5.6-sol`
- Question: Can the current package-only gated architecture stand in for the configured default `go test -count=1 ./...` scope?
- Prior hypothesis: Compiling and running only the mutated package might preserve the default verdict while removing repeated Go driver starts.
- Intervention: A four-package disposable module placed one sentinel mutation in `pkg0` that only a dependent-package test in `pkg1` could kill.
- Control: Ordinary `./...` execution had to produce twelve mutants, thirteen Go driver starts, 52 package-test executions, six killed, and six survived. The package-only mode had to expose the sentinel disagreement rather than accidentally agree.
- Exact evidence:
  - ordinary A: 13 Go driver starts, 52 package-test executions, 6 killed, 6 survived
  - package-only B: 1 Go driver start, 13 package-test executions, 5 killed, 7 survived
  - module-scope C: 1 Go driver start, 52 package-test executions, 6 killed, 6 survived
  - sentinel `pkg0/values.go:4:16`: A/C killed; B survived
  - every other ordered verdict and address agreed
- Wall-clock observation: Package-only was fastest because it skipped required work; that number is invalid as evidence for configured-scope performance.
- Verdict: The package-only architecture is not scope-equivalent. Its speed partly comes from omitting tests the user configured.
- What changed: The proposal moved from “make the package runner faster” to “make the configured test scope a first-class execution plan.” No production code changed.
- What remains unknown: How production should discover package binaries, handle duplicate package names, and retain verdict-reason fidelity without reintroducing one `go test` invocation per mutant.
- Next falsifiable step: Measure a complete module-scope prebuilt runner with all package binaries executed for every selector.
- Artifacts: `docs/experiments/module-scope-runner.md`, `internal/perfbench/module_scope_experiment_test.go`.

## 2026-09-18 — 003 — Module-scope execution removes the driver toll

- Status: advance
- Revision: `5f65e3d3c4201689b81b707533b18aa42364ea8b` plus the uncommitted experiment harness
- Model: `gpt-5.6-sol`
- Question: Can one module-scope build preserve all configured package executions and verdicts while significantly reducing Go driver starts?
- Prior hypothesis: Module-scope C would match ordinary A exactly, execute all 52 package tests, reduce driver starts from 13 to 1, and keep C/A wall time at or below 0.25 in every measured round.
- Intervention: Instrument the twelve real comparison mutants once, compile four uniquely named package test binaries in one driver start, then run every binary for selector zero and every mutant selector.
- Control: A permanent refusal test supplied 52 executions against an intentionally wrong expectation of 51 and produced `C module-scope package-test executions = 52, want exactly 51`. Per-selection guards required +4 package executions in A/C and +1 in B, including after killed mutants.
- Exact evidence:
  - all three measured rounds: A = 13 driver starts / 52 package tests
  - all three measured rounds: B = 1 / 13
  - all three measured rounds: C = 1 / 52
  - A and C ordered addresses and verdicts: identical
  - driver-start reduction: 12 of 13, or 92.3%
- Wall-clock observation:
  - round 1: A 11.4070712 s, C 2.2254465 s, C/A 0.1951
  - round 2: A 9.6900866 s, C 1.9315122 s, C/A 0.1993
  - round 3: A 10.2873751 s, C 1.8605086 s, C/A 0.1809
- Verdict: All three pre-registered hypotheses were corroborated. The light fixture ran 5.0–5.5 times faster without narrowing scope.
- What changed: The evidence selects module-scope execution as the production direction.
- What remains unknown: Real-repository gain; one-time package discovery cost; custom commands; duplicate package names; packages without tests; Windows output naming; verdict-reason fidelity; and globally unique selectors across multiple source files.
- Next falsifiable step: Agree on the production contract. Recommended first slice: optimize only the exact default Go module scope and preserve custom commands through ordinary fallback.
- Artifacts: `docs/experiments/module-scope-runner.md`, `internal/perfbench/module_scope_experiment_test.go`, `docs/learning-log.md`.

## 2026-09-18 — 004 — Production contract remains open

- Status: decision
- Revision: working tree based on `5f65e3d3c4201689b81b707533b18aa42364ea8b`
- Model: `gpt-5.6-sol`
- Question: Which command forms may use the new module-scope execution plan?
- Prior hypothesis: Restricting optimization to the default Go `./...` scope and falling back for custom commands is the smallest scope-faithful production contract.
- Intervention: None; production implementation is intentionally paused for contract review.
- Control: Not applicable until the contract selects a behavior.
- Exact evidence: The measured architecture covers one source-file batch, twelve comparison mutants, four uniquely named packages, and the default module scope. It does not cover arbitrary commands.
- Wall-clock observation: No production timing exists yet.
- Verdict: Open. No production core files have been modified.
- What changed: A dedicated append-only continuity record now exists before implementation begins.
- What remains unknown: Whether custom commands must be optimized now or retain the ordinary path.
- Next falsifiable step: Select the command contract, then write behavior-first RED tests for scope fidelity and fallback before implementing the runner.
- Artifacts: `odd/tasks/module-scope-core.md`, `docs/performance-core-log.md`.

## 2026-09-18 — 005 — Safe `Gated()` contract accepted

- Status: decision
- Revision: working tree based on `5f65e3d3c4201689b81b707533b18aa42364ea8b`
- Model: `gpt-5.6-sol`
- Question: Which command forms may enter module-scope execution?
- Prior hypothesis: Optimizing only the exact default Go module scope, with ordinary fallback for custom commands, is the smallest scope-faithful contract.
- Intervention: The user accepted the recommended contract.
- Control: Production behavior must be written first as failing tests: default `./...` is eligible; custom commands, unsupported flags, duplicate names, missing binaries, and build/layout failures take the ordinary path.
- Exact evidence: The accepted contract is bounded by the measured population in entry 003; no broader command support is inferred from it.
- Wall-clock observation: None; this is a contract decision, not a measurement.
- Verdict: Proceed with production implementation behind `Gated()` using default-scope admission and fail-closed ordinary fallback.
- What changed: Production implementation is now authorized within this boundary.
- What remains unknown: Production counters, verdict-reason fidelity, Windows behavior, and real-repository gain.
- Next falsifiable step: Write RED tests for the module runner and fallback boundary before changing production behavior.
- Artifacts: `docs/performance-core-log.md`, `odd/tasks/module-scope-core.md`.

## 2026-09-18 — 006 — Production module runner passes its boundary

- Status: advance
- Revision: working tree based on `5f65e3d3c4201689b81b707533b18aa42364ea8b`
- Model: `gpt-5.6-sol` with delegated writer and independent verifier
- Question: Can a production runner represent the complete default `./...` scope, fail closed when it cannot, and preserve every package execution after a failure?
- Prior hypothesis: Structured Go package discovery plus one module build can replace per-mutant driver starts without narrowing scope.
- Intervention: Added `ModuleScopeRunner` in `internal/gobuildrunner`, with structured `go list -json ./...` discovery, one `go test -c -o <directory> ./...` compilation, deterministic package order, and all-package execution per selector.
- Control: Behavior-first tests cover the cross-package sentinel, duplicate binary names, packages without tests, discovery/build diagnostics, successful build with a missing expected binary, red baseline continuation, and POSIX/Windows binary naming. Manual mutations removed layout rejection and continuation after failure; each owning test failed.
- Exact evidence:
  - sentinel: discoveries 1, Go tool starts 2, compilations 1, selections 2, package runs 4
  - duplicate output name: discoveries 1, starts 1, compilations 0, selections 1, package runs 0
  - missing expected binary: discoveries 1, compilations 1, package runs 0
  - red baseline order: `alpha:0`, then `omega:0`; later packages still ran
  - independent `go test`, focused tests, short tests, and `go vet` all exited 0 from `.git`-free disposable copies
- Wall-clock observation: Not measured in this slice; the runner boundary was evaluated by exact work counters and behavior.
- Verdict: The module runner boundary passed independent verification. Actual Windows process execution remains owned by the existing Windows CI matrix; pure naming is covered locally.
- What changed: Production code now has a complete-scope runner, but `Gated()` does not use it yet.
- What remains unknown: Admission of only the default command, decorator/fallback compatibility, verdict-reason behavior through the release stack, and end-to-end performance.
- Next falsifiable step: Wire safe-default command admission into `Gated()` with custom-command fallback and exact gated/fallback counters.
- Artifacts: `internal/gobuildrunner/module_scope.go`, `internal/gobuildrunner/module_scope_test.go`, `internal/gobuildrunner/module_scope_failures_test.go`.

## 2026-09-18 — 007 — `Gated()` admits only the scope it can answer

- Status: advance
- Revision: working tree based on `5f65e3d3c4201689b81b707533b18aa42364ea8b`
- Model: `gpt-5.6-sol`
- Question: Can `Gated()` replace the configured test command without ever answering a smaller question than the caller asked?
- Prior hypothesis: Admitting only the default Go module scope, and delegating everything else to the ordinary configured path, removes no capability while removing the measured toll where it is payable.
- Intervention: Added an unexported `commandScope` to `Options`, classified by a pure `scopeOf` helper that accepts only `go test -count=1 ./...` and its built-in `-json` equivalent in either flag order. `assemble` now selects `NewModuleScope` for an admitted scope and `NewDisabled` otherwise; `NewDisabled` keeps the batched-laboratory shape and counters while routing every mutant to the ordinary laboratory.
- Control: The admission table covers make targets, package-local scopes, omitted `-count=1`, `-race`, `-tags`, absolute and alternate executables, extra spacing, and malformed tokens. A focused test proves disabled gating delegates every mutant with `Gated()==0` and `FellBack()==N`. The golden fixture was moved to the scope `Gated()` is allowed to replace, so its `gated something` assertion still proves the path engages; a second run with `DITTO_GOLDEN_PACKAGE_ONLY=1` proves a package-local command reports `none` with unchanged verdicts.
- Exact evidence:
  - `gofmt -l .` clean; `go vet ./...` exit 0; `go test ./...` green across every package including the repository root
  - `TestReleaseGolden` passed in 21.73 s, exercising both the engaging and the delegating half
  - weakening `scopeOf` to admit everything made the new guard refuse: `a package-local command gated 4 mutants; module scope may only replace the complete Go module scope`
  - admission table, `TestOptions` (test-command value unchanged), `TestGatedLaboratory`, and all seven module-scope runner tests passed
- Wall-clock observation: The golden fixture now runs the module scope twice, so that test costs roughly one extra release. Reported, not gated.
- Verdict: The safe-default `Gated()` contract holds. Custom commands keep exactly the runner they configured, and disengagement is printed as `none` rather than hidden.
- What changed: Production `Gated()` no longer replaces an arbitrary `WithTestCommand`, which is a behavior change for callers who paired the two. It is the change the measured scope defect required: the package-only path was answering a smaller question than `./...`.
- What remains unknown: End-to-end release evidence through the real CLI, verdict-reason fidelity on the module path, and real-repository gain.
- Next falsifiable step: Run the release end-to-end from a disposable copy and measure the module path against the ordinary path through the shipped binary.
- Artifacts: `options.go`, `release.go`, `gated_scope_internal_test.go`, `internal/gatedlaboratory/gatedlaboratory.go`, `release_golden_test.go`, `testdata/goldenproject/mutation_test.go`.

## 2026-09-18 — 008 — The ratchet fired, and the number was written down with its cause

- Status: advance
- Revision: working tree based on `5f65e3d3c4201689b81b707533b18aa42364ea8b`
- Model: `gpt-5.6-sol`
- Question: What does the new core cost in the one counter that measures this repository rather than a fixture?
- Prior hypothesis: New production code adds mutable sites, so `mutantsPerReleaseOnThisRepository` grows and the ratchet refuses until the number is recorded.
- Intervention: Recorded `813` in `perf/baseline.json` with its attribution, as the repository requires: a counter is never adjusted to match a measurement without naming what moved it.
- Control: The attribution was measured per file rather than attributed to the change as a whole. Each named file was counted at the revision before this change and after it, with the default virus set and the gate's own exclusions.
- Exact evidence:
  - `internal/gobuildrunner/module_scope.go`, new file: 23 mutants, all of them new
  - `internal/gatedlaboratory/gatedlaboratory.go`: 36 before, 37 after, +1
  - `options.go`: 5 before, 5 after, +0 — the command-scope table replaced a branchy classifier with the same number of mutable sites
  - sum 24, against the ratchet's own report of 813 where the baseline was 789
  - the full suite passed 497 tests with 10 skipped before the counter fired; only this counter was red
- Wall-clock observation: The gate reached the counter in about 63 s of tests. Reported, not gated.
- Verdict: The ratchet was correct and the cause is known. The counter speaks for how many mutants a scope produces and says nothing about what judging them costs, which is what the module-scope work changes.
- What changed: `perf/baseline.json` now says 813, with the +24 attributed by file.
- What remains unknown: Whether the module-scope path actually lowers the gate's wall clock on this repository. That is the next measurement, and the only one that can say the added mutants were bought back.
- Next falsifiable step: Run the gate's own scope through the module path from a disposable copy and compare driver starts and wall time against the ordinary path.
- Artifacts: `perf/baseline.json`, `internal/perfbench/repository_test.go`.

## 2026-09-18 — 009 — The change landed as four work units

- Status: advance
- Revision: `5f65e3d` plus four commits on `perf/module-scope-core`
- Model: `gpt-5.6-sol`
- Question: Is the work recorded in reviewable units, each one green on the repository's own gate?
- Prior hypothesis: One unit per claim — the measurement, the runner, the admission, the record — so a reviewer can accept or reject a claim without carrying the others.
- Intervention: Four commits, each passed through `.githooks/pre-commit` (golangci-lint, the suite, and the counters):
  - `5d58801` `test(perfbench): measure what one module-scope build replaces` — the experiment note, the tagged experiment, and the learning-log line
  - `3d35fd5` `feat(gobuildrunner): run the complete module scope from one compilation` — the runner and its two internal test files
  - `8a3c6eb` `feat(gated): replace only the configured scope, and say when it does not` — admission, fallback, the golden fixture, and the moved counter
  - `360f794` `docs: keep an append-only record of the core-performance request` — this file
- Control: The gate is the repository's own, and it refused the work three times before any of this landed: once for nine lint findings across the new runner and the two cyclomatic-complexity overages, and twice more for the counter and a missing import. Every refusal is recorded rather than hidden by a suppression, and the only two suppressions added name `go list -json`'s own field names, which is the same reason `internal/verdict` already gives for its event struct.
- Exact evidence:
  - each commit's hook run: `DONE 497 tests, 10 skipped`, counters green
  - `perf/baseline.json` at 813 with the +24 attributed by file
- Wall-clock observation: About 60 s of suite per commit through the hook, cached where nothing changed. Reported, not gated.
- Verdict: The change is committed in reviewable units, each independently green.
- What changed: The working tree is empty except for `odd/`, this session's own task tracker, which is deliberately not part of ditto's history: it is an el Gentleman convention, not a ditto one, and the durable record here is this file plus `docs/experiments/`.
- What remains unknown: Everything entry 008 leaves open — the CLI end-to-end measurement, verdict-reason fidelity on the module path, and whether the module path lowers this repository's own gate time enough to buy back the 24 mutants it added.
- Next falsifiable step: Run the release end-to-end from a disposable copy through the shipped binary, and check whether a killed mutant on the module path still carries a reason other than `Unknown`.
- Artifacts: `git log 5f65e3d..perf/module-scope-core`.

## 2026-09-18 — 010 — A module-path kill carried no reason, and now does

- Status: advance
- Revision: working tree on top of `243f2ad`
- Model: `gpt-5.6-sol`
- Question: What reason does a module-scope kill carry, and does the ordinary path's reason survive the move to prebuilt binaries?
- Prior hypothesis: `verdict.ReasonOf` reads the stream `go test -json` emits, and `-json` belongs to the driver rather than to the binary it starts, so every module-path kill would report `Unknown`.
- Intervention: The module runner now converts a **failing** package's output through `go tool test2json -t -p <package>` before it reaches the reporter. A green selection starts no converter, because a reason is only ever asked of a kill.
- Control: The ordinary `go test -count=1 -json ./...` command over the same fixture and the same active mutant reported `assertion`. The module path reported `unknown` before the change and `assertion` after it. A non-compiling package was measured separately and fails closed before any binary starts.
- Exact evidence:
  - control: `assertion`
  - module path before: `unknown`
  - module path through `test2json`: `assertion`
  - RED captured behaviourally as `expected "assertion", actual "unknown"`, not as a compile error
  - manual mutation: deleting the conversion made the owning test fail with the same pair, and `ConverterStarts` drop from 1 to 0
  - the full gate then reported `mutantsPerReleaseOnThisRepository: 818, baseline 813 (+5)`, attributed to `internal/gobuildrunner/module_scope.go` alone: 23 before, 28 after
- Wall-clock observation: The converter is one process per failing selection, paid only on a kill. Reported, not gated.
- Verdict: The gap was real and is closed. `--confirm-kills` had been a silent no-op on the gated path, because `internal/confirminglaboratory` re-runs a kill only when the reason is `Assertion`.
- What changed: `readable` in the module runner, two guards, and `perf/baseline.json` at 818 with the file attributed.
- What remains unknown: A **deadline** kill is still not covered and is a different defect: the ordinary path writes its own marker when ditto fires the clock, while on the module path the clock belongs to the binary's `-test.timeout`, whose panic text would convert into an `Assertion`. Recorded, not fixed, and not implied to be fixed.
- Next falsifiable step: Measure the gated path through the shipped binary rather than a harness.
- Artifacts: `docs/experiments/module-path-verdict-reason.md`, `internal/gobuildrunner/module_scope.go`.

## 2026-09-18 — 011 — Through the shipped binary, gated is 3.14× on a light suite

- Status: advance
- Revision: working tree on top of the reason fix
- Model: `gpt-5.6-sol`
- Question: What does `ditto run --gated` actually buy end to end, as a person experiences it?
- Prior hypothesis: the same verdicts and survivor addresses, in at most half the wall clock.
- Intervention: None to the product. Both modes were run through the binary built from a disposable copy, against a throwaway three-package module.
- Control: The first fixture produced zero survivors, which would have made the address comparison vacuous, so three uncovered functions were added and six mutants survived. The first run also said `Gated: none of 24`, and the cause was the fixture rather than the product — the generated sources had a trailing blank line, so they were not gofmt-formatted and `schemata.Plan` refused every site. That is a real product property and is recorded as one.
- Exact evidence:
  - ordinary: 23,145 ms, 30 total, 24 killed, 6 survived
  - gated: 7,361 ms, 30 total, 24 killed, 6 survived, `30 of 30 mutants ran from one compilation`
  - ratio 0.318, or 3.14× faster
  - sorted survivor addresses byte-identical between the modes, over six real survivor reports
  - no recorded counter moved, and `perf/baseline.json` stayed where entry 010 left it
- Wall-clock observation: 23.1 s against 7.4 s on the same machine, ordinary run first so a cold toolchain could not favour it. Reported, not gated.
- Verdict: All three hypotheses corroborated. The architecture earns its keep end to end on the case it was built for.
- What changed: The claim moves from "the mechanism is 5× cheaper" to "a run is about a third of the wall clock", which is the smaller and honest number: a release also pays parsing, instrumentation, the sandbox, the progress line, and one converter per kill.
- What remains unknown: A slow suite, where the removable toll is a smaller share of the bill and an earlier measurement said 0.50-0.58 rather than 0.32; and this repository's own gate, which is repository-sized, takes tens of minutes, and has 24 more mutants than before this change.
- Next falsifiable step: Run this repository's own gate scope ordinary against gated from a disposable copy, and answer whether the module path buys back the 24 mutants it added.
- Artifacts: `docs/experiments/gated-through-the-binary.md`.

## 2026-09-18 — 012 — The module path scales with packages, and that is a cliff

- Status: correction
- Revision: `bb5811e`
- Model: `gpt-5.6-sol`
- Question: Does the gated gain survive a repository with more packages than the three it was measured on?
- Prior hypothesis: the toll removed is the `go test` driver start, which is paid once per mutant in both modes, so the gain should be roughly independent of the package count.
- Intervention: None to the product. The same binary was run over a ten-package light fixture instead of a three-package one.
- Control: The three-package fixture was re-measured in the same session with the same binary and reported 0.3149-0.3201 across three rotated rounds. Both fixtures report identical verdicts in both modes, so neither comparison is between different questions.
- Exact evidence:
  - 3 packages, 30 mutants: ordinary 22,959 ms, gated 7,349 ms, ratio **0.3201**
  - 10 packages, 40 mutants: ordinary 64,195 ms, gated 42,686 ms, ratio **0.6649**
  - identical verdicts in both fixtures: 24 killed / 6 survived, and 20 killed / 20 survived
  - gated line in the ten-package run: `40 of 40 mutants ran from one compilation`
- Wall-clock observation: The gain collapsed from 3.14× to 1.50× by going from three packages to ten. Taking the ordinary cost as one driver start per mutant (64,195 / 40 = 1,605 ms) and the gated cost as one test binary per package per mutant (42,686 ms over 400 binary starts plus one compile), the per-start cost is around 98 ms and the crossover sits near sixteen packages. That arithmetic is derived from measured totals rather than measured directly, and it is recorded as an estimate.
- Verdict: The prior hypothesis was wrong. The module path replaces one driver start per mutant with one test binary **per package per mutant**, so its cost is `selections × packages` while the ordinary path is `selections`. The gain is real and the shape is wrong: on a repository with enough packages the path becomes slower than the one it replaced.
- What changed: The next core change is now known instead of guessed, and the ten-package case is a case the design must answer before `--gated` is recommended for anything but a small module.
- What remains unknown: The exact crossover, which is derived rather than measured; whether the per-start cost is stable across larger test binaries; and this repository's own tree, which has far more than sixteen test packages.
- Next falsifiable step: Run only the packages whose test binaries can observe the mutated package, derived from the Go import graph, and measure package executions per selection against the same two fixtures. A package that cannot transitively import the mutated package cannot observe the mutation, so that reduction is provable rather than heuristic — and the cross-package sentinel from `module-scope-runner.md` is the guard that the closure is not too narrow.
- Artifacts: `docs/experiments/gated-through-the-binary.md`, `docs/experiments/module-scope-runner.md`.

## 2026-09-18 — 013 — The observability closure divides the package factor

- Status: advance
- Revision: `2deb438`
- Model: `gpt-5.6-sol`
- Question: Can the `selections × packages` factor be divided by running only the package test binaries whose dependency closure contains the mutated package?
- Prior hypothesis: a binary without the mutated package in its closure contains no code that can refer to the mutation, so the reduction is provable and should cost `selections × |observers(P)|` instead of `selections × N`.
- Intervention: None to the product. The experiment implements the restriction by hand so the number precedes the design, and runs it against the same fixture as the unrestricted mode.
- Control: The shipped `ModuleScopeRunner` started 35 package binaries over the same fixture and the same five runs, matching the experiment's unrestricted count exactly — without that, the numbers would describe the harness. A per-selection execution log written by the packages themselves makes "the island did not run" observed rather than inferred.
- Exact evidence:
  - closure of a mutation in `base`: `base`, `mid`, `top` — three of seven packages; the four islands excluded
  - `top` is in the closure although it never names `base`, reaching it through `mid`
  - unrestricted: 35 executions, verdicts `killed, killed, killed, survived`
  - restricted: 15 executions, identical verdicts, sentinel still killed
  - 20 of 35 executions removed, a 57% reduction
- Wall-clock observation: Not measured; this question is about an exact counter.
- Verdict: All three hypotheses corroborated. The reduction is provable rather than heuristic and it is not the defect that was just fixed: that one ran the mutated package and nothing else, this one runs it plus everything that can reach it.
- What changed: The next core change is selected and its ceiling is known before any production code changes shape.
- What remains unknown: The cost of computing the closure, which is one `go list` per release rather than per mutant and is paid on top of the discovery already done; whether the reduction reproduces through the shipped binary; and a repository whose packages form a single chain, where the closure is every package and the factor does not divide.
- Next falsifiable step: Implement `ScopeTo` on the module runner behind an optional interface, compute the closure from `go list -deps -test -json ./...`, and fall back to the full scope whenever the mutated package cannot be resolved — then re-measure the ten-package fixture that showed the cliff.
- Artifacts: `docs/experiments/dependency-closure.md`, `internal/perfbench/closure_experiment_test.go`.

## 2026-09-18 — 014 — The closure is implemented, and the cliff bends

- Status: advance
- Revision: `cd24b12`
- Model: `gpt-5.6-sol`
- Question: Does the measured ceiling survive contact with the shipped path?
- Prior hypothesis: running only the test binaries whose closure contains the mutated package divides `selections × packages` and bends the ten-package cliff without moving a verdict.
- Intervention: `ScopeTo` on the module runner, called from `GatedLaboratory` through an optional `scopedRunner` interface; the closure comes from `go list -deps -test -json ./...`, filtered to the module's own packages; a scope that nothing resolves runs every package.
- Control: RED captured behaviourally — with the scope stored but not applied, the guard failed with 4 executions against 3 — and seen refusing again with the filter disabled. The previous entry's shipped-runner control already tied the harness to the product.
- Exact evidence:
  - seven-package guard: 3 of 7 packages run for a mutation in the scoped package, sentinel still killed
  - ten-package module, forty mutants, identical verdicts (20 killed / 20 survived):
    - ordinary 64,103 ms
    - gated before 42,686 ms, ratio 0.6649
    - gated after 25,607 ms, ratio **0.3995**
  - the full suite reached 501 tests; lint clean; counters green
  - the ratchet moved 818 → 846 (+28), attributed per file: `module_scope.go` 28 → 54 (+26), `gatedlaboratory.go` 37 → 39 (+2)
- Wall-clock observation: One run per mode, so the ratio is reported rather than relied on; the exact counters are the contract and the guard is what the change is judged on.
- Verdict: The ceiling reproduced through the shipped binary, and the cliff bent from 0.6649 to 0.3995. The change is kept.
- What changed: The module path no longer scales with the package count alone, which was the property that would have made `--gated` a trap on a real repository.
- What remains unknown: This fixture is the best case for the closure — every package independent, so a mutation is observed by one package of ten. A repository whose packages form a chain gets less, and a single chain gets nothing. The closure's own cost is still unmeasured. And the ten-package ratio was taken once per mode rather than across rotated rounds.
- Next falsifiable step: Re-run the ten-package comparison across rotated rounds, and measure the closure's own `go list` cost against the executions it removes on a repository whose layout is a chain rather than a fan.
- Artifacts: `internal/gobuildrunner/module_scope.go`, `internal/gatedlaboratory/gatedlaboratory.go`, `perf/baseline.json`.

## 2026-09-18 — 015 — The ten-package ratio survives rotated rounds

- Status: advance
- Revision: `c4c2d25`
- Model: `gpt-5.6-sol`
- Question: Was the 0.3995 of entry 014 a measurement or a single favourable pair?
- Prior hypothesis: the ratio from entry 014 was taken once per mode, which is not a measurement by this repository's standard and is not a claim it is entitled to make.
- Intervention: None to the product. The same binary and the same ten-package fixture, one discarded warm-up and then three measured rounds with the mode order rotated, each ratio computed within its own round.
- Control: The warm-up ran each mode once so the toolchain and file caches were warm for every recorded round; round 2 ran the gated mode first, so the result is not one mode being favoured by position.
- Exact evidence:
  - round 1, order A B: ordinary 63,619 ms, gated 25,546 ms, ratio **0.4015**
  - round 2, order B A: gated 25,454 ms, ordinary 63,560 ms, ratio **0.4005**
  - round 3, order A B: ordinary 63,414 ms, gated 25,618 ms, ratio **0.4040**
  - identical verdicts in every round: 40 total, 20 killed, 20 survived; `40 of 40 mutants ran from one compilation`
- Wall-clock observation: the three ratios span 0.4005 to 0.4040, a spread of 0.9%. The single pair from entry 014, 0.3995, sat inside that band, which is the useful part of this entry: the earlier number was not lucky, it was a small sample of a stable one.
- Verdict: The number stands and is now a measurement rather than an observation. Entry 014's caveat is closed rather than carried forward.
- What changed: The ten-package gain is 2.49-2.50× with the guarantee the repository asks for.
- What remains unknown: Unchanged in kind and now stated without the sampling caveat. This fixture is still the best case for the closure — every package independent, so a mutation is observed by one package of ten — and a chain-shaped repository will get less while a single chain gets nothing. The closure's own `go list` cost is still unmeasured.
- Next falsifiable step: Measure the closure's own cost against the executions it removes on a chain-shaped repository, which is the one shape where the factor does not divide.
- Artifacts: `docs/performance-core-log.md`.

## 2026-09-18 — 016 — On a chain, the prediction was refuted and the shape did not behave as argued

- Status: correction
- Revision: `5f947d9`
- Model: `gpt-5.6-sol`
- Question: What does the closure do on a chain, the one shape where entry 014 admitted the factor might not divide?
- Prior hypothesis: a mutation at the chain's head is observed by every package so the closure removes nothing, and the fixture would land near the pre-closure worst case; a mutation at the tip is observed by one package and would land near the favourable ratio.
- Intervention: None to the product. Two eight-package chain modules from one generator, differing only in which package holds the mutable comparison sites.
- Control: One discarded warm-up per fixture, three rotated rounds, and the observer counts read from `go list -deps -test` — the same source the runner itself uses, so the prediction and the implementation cannot disagree about what a closure is.
- Exact evidence:
  - observers: **8 of 8** for a mutation in `pkg0`, **1 of 8** for one in `pkg7` — both exactly as predicted
  - chain-head: 0.3492, 0.3528, 0.3597
  - chain-tail: 0.5408, 0.5623, 0.5536
  - every round in both fixtures: 6 total, 2 killed, 4 survived, `6 of 6 mutants ran from one compilation`
- Wall-clock observation: the fixture where the closure removes nothing is about 1.5× **faster** than the one where it removes seven of eight. Each fixture's three ratios sit inside a band narrower than 3%.
- Verdict: H1 and H2 corroborated, **H3 refuted in the reversed direction**. The prediction that the head fixture would be the slow one is dead.
- What changed: the claim that the closure is what decides these two numbers is now in doubt. Something else is deciding them, and the note names its candidates without choosing: the compile, the sandbox, the instrumentation of a file eight packages import against a file none does, and the converter.
- What remains unknown: which candidate it is. The obvious story — that instrumenting a package everything imports forces a wider rebuild — is a conjecture with no measurement behind it, and the repository's own rule is that a cause offered in passing still needs a kill criterion.
- Next falsifiable step: separate the phases. Time the compile, the baseline selection and the mutant selections independently on both fixtures, and read `SkippedPackages` to confirm the closure engaged on the tail fixture at all. If the closure did not engage there, entries 013 and 014 need re-reading.
- Artifacts: `docs/experiments/chain-shaped-module.md`.

## 2026-09-18 — 017 — The closure engaged, and the time still went the other way

- Status: correction
- Revision: `133e040`
- Model: `gpt-5.6-sol`
- Question: Did the closure silently fail to engage on the tail fixture, which would make entry 016 a harness defect rather than a result?
- Prior hypothesis: the comfortable reading of entry 016 was that the closure never applied, and the refutation was therefore an artifact.
- Intervention: None to the product. A disposable copy of the tree was patched to print the runner's own counters, and both chain fixtures were re-run through that binary.
- Control: The same fixtures and the same mutants as entry 016, which had already produced identical verdicts and `6 of 6 mutants ran from one compilation`.
- Exact evidence:
  - `chain-head`: packageRuns 16 at the second kill — 2 selections × 8 observers — skipped 0, compilations 1
  - `chain-tail`: packageRuns 2 at the second kill — 2 selections × 1 observer — skipped 14, compilations 1
  - both fixtures: 6 total, 2 killed, and `6 of 6 mutants ran from one compilation`
- Wall-clock observation: The tail fixture started **eight times fewer** package binaries and skipped fourteen, and was still about 1.9 s slower. The closure's engagement is therefore confirmed and the refutation of entry 016 stands.
- Verdict: The comfortable explanation is dead. It is not the number of package executions that costs the tail fixture its time.
- What changed: The refutation is now a measured result rather than a suspicious one, and the remaining candidates are narrowed to what the two fixtures do not separate: the one compile both pay, the sandbox, instrumenting a file that eight packages import against one that none does, and the two converters both runs start.
- What remains unknown: Which candidate it is. The obvious one — that instrumenting the package everything imports forces a wider rebuild — now has a kill criterion rather than a story: time the compile phase alone on both fixtures, and if the two are within noise, it is dead too.
- Next falsifiable step: Run that phase split. It is the cheapest remaining experiment and it either names the cause or eliminates the last obvious one.
- Artifacts: `docs/experiments/chain-shaped-module.md`.

## 2026-09-18 — 018 — The compile is charged per source file, and that was the whole gap

- Status: advance
- Revision: `828223c`
- Model: `gpt-5.6-sol`
- Question: Which of the four remaining candidates costs the tail fixture its 1.9 seconds?
- Prior hypothesis: the compile was the obvious one — that instrumenting `pkg0`, which eight packages import, forces a wider rebuild than instrumenting `pkg7`, which none does — and it carried an explicit kill criterion.
- Intervention: None to the product. A disposable copy was patched to time discovery, the module compile and the selections separately, and to print the runner's counters on every call.
- Control: The same fixtures and the same mutants as entries 016 and 017, which had already produced identical verdicts and `6 of 6 mutants ran from one compilation`. The head fixture's instrumented phases reconcile with its wall clock: 169 + 1,651 + 1,902 = 3,722 ms against 3,828 measured.
- Exact evidence:
  - `chain-head`: one runner, discover 169 ms, build 1,651 ms, run 1,902 ms, 56 package runs, wall 3,828 ms
  - `chain-tail`: **two runners** — the first unscoped at discover 141 / build 1,676 / 24 package runs, the second scoped at discover 155 / build 1,580 / 5 package runs and 35 skips — wall 5,718 ms
  - mutant distribution: head `pkg0/pkg0.go — 6 mutants`; tail `pkg0/pkg0.go — 2 mutants` and `pkg7/pkg7.go — 4 mutants`
  - compile measured outside the product: 1,306 ms head against 1,347 ms tail
- Wall-clock observation: The tail fixture paid a second module-wide compile of 1,580 ms. That, and not the package executions, is the 1.9 second gap.
- Verdict: The cause is measured. `ditto.Release` batches per source file and each batch builds its own `GatedLaboratory` runner, so **the module-wide compilation is charged once per source file that has mutants, not once per release**. The wider-rebuild conjecture is refuted by both the in-product and out-of-product compiles.
- What changed: The next lever is named with its price. Sharing one compile across batches would remove a 1,580 ms charge on a fixture with two mutable files, and proportionally more on a repository with many.
- What remains unknown: The cost of sharing a compile is not measured because nothing shares one yet. And entries 014/015 need re-reading rather than correcting: the ten-package fixture has mutants in every package, so it paid **ten** compiles and still reached 0.3995 — a lower bound on what sharing one would give, with the size of the gap left as arithmetic rather than claimed as measurement.
- Next falsifiable step: Share one compilation across the source-file batches of a single release and measure compilations per release, which is the counter that says whether it happened, against the same two fixtures and the ten-package one.
- Artifacts: `docs/experiments/chain-shaped-module.md`.

## 2026-09-18 — 019 — Sharing the compile directory is worth 12,379 ms on a ten-package module

- Status: advance
- Revision: `616c842`
- Model: `gpt-5.6-sol`
- Question: What is the compilation charge that entry 018 identified actually worth, before anything is built for it?
- Prior hypothesis: a fresh output directory throws away the toolchain's own up-to-date check, so ten batches cost ten full rebuilds where one directory would cost one and nine near-no-ops.
- Intervention: None to the product. Two loops of ten `go test -c -o <dir> ./...` over one ten-package module, differing only in whether the directory is fresh.
- Control: Both loops run the same command in the same tree with no mutation in flight, so they differ in exactly one thing. The first compile of each loop is the same job and reads 1,405 ms against the fresh loop's 1,405 ms-scale runs.
- Exact evidence:
  - discovery `go list -deps -test -json ./...`: 190, 177, 174 ms — about 180 ms per release, paid once
  - ten fresh directories: **15,466 ms**, or 11.0× the first compile
  - one shared directory: first 1,405 ms, then 288, 168, 197, 189, 171, 166, 169, 167, 167 — **3,087 ms** total
  - saving: **12,379 ms**, against a pre-registered threshold of 8,500 ms
  - for scale: the ten-package gated run of entries 014/015 measured 25,607 ms, so this is 48% of that run
- Wall-clock observation: Every shared compile after the first is between 166 and 288 ms against a first of 1,405 ms, which is backlog entry 14's up-to-date check working as it described.
- Verdict: All three hypotheses corroborated. The prize is real and it is large, and the mechanism is the toolchain's own check rather than anything re-implemented: the change is which directory the binaries are written to.
- What changed: The next core change is sized. A gated ratio of about 0.21 instead of 0.3995 on a ten-package module is the ceiling.
- What remains unknown: **The tree did not change between compiles, and in a real release it does.** Each batch instruments a different file, so this is a ceiling for a fan and an overestimate for a chain, where instrumenting a package everything imports relinks all of them. The honest figure lies between this and zero, and it has to be measured after the change rather than cited before it. Also unmeasured: where the shared directory should live, and who removes it — a directory that outlives its sandbox is a lifetime the current design does not have.
- Next falsifiable step: Give the module runner one compilation directory per release, count compilations per release, and re-measure the ten-package fixture, the chain fixtures and the closure's own cost.
- Artifacts: `docs/experiments/the-compile-is-per-file.md`.

## 2026-09-18 — 020 — The prize did not appear, and the reason is the sandbox

- Status: correction
- Revision: `c049ab7`
- Model: `gpt-5.6-sol`
- Question: Does one compilation directory per release collect the 12,379 ms that entry 019 priced?
- Prior hypothesis: a fresh output directory per batch throws away the toolchain's up-to-date check, so reusing one directory would collect the prize.
- Intervention: `GatedLaboratory` took one compilation directory per release and handed it to every batch's runner; `ModuleScopeRunner.SetCompilationDirectory` accepts it; `CompilationDirectories()` counts it.
- Control: The guard for the sharing was watched refusing — `expected 1, actual 2` with two different paths — when the sharing rule was deliberately broken. The ten-package comparison was run as three rotated rounds against the same fixture and the same binary shape as entries 014/015.
- Exact evidence:
  - **The ratio did not move.** Before: 0.4005, 0.4040, 0.3995. After: 0.4031, 0.3996, 0.4015. Both fixtures: 40 total, 20 killed, 20 survived, `40 of 40 mutants ran from one compilation`.
  - The directory **is** shared: every batch printed `out ditto-module-compile-79694792`.
  - Every batch still spent about 1,750 ms compiling — 1857, 1755, 1744 — against the ~170 ms a reused directory measured in entry 019's loop.
- Wall-clock observation: The saving predicted at 12,379 ms did not appear at all. The change is worth nothing as built.
- Verdict: The prediction is refuted in full. **Each batch links its own sandbox, so package directories have different absolute paths, and Go's build IDs cover those paths.** Nothing is up to date between batches however the output directory is chosen, so the check the priced loop was measuring never fires. Entry 019's 12,379 ms measured a situation that cannot occur.
- What changed: The change was **reverted** — it added three mutable sites and a counter and bought nothing measurable, and this repository's rule is that an unearned cost is written down rather than carried. The finding is kept; the code is not.
- What remains unknown: Whether one sandbox per release would stabilise the paths and let the check fire, which is the change that would actually collect the prize. It is bigger than the one just reverted and it is named rather than attempted.
- Next falsifiable step: One sandbox per release instead of one per batch, and only then a shared compilation directory — in that order, because sharing the directory without stabilising the paths is the thing just measured to pay nothing.
- Artifacts: `docs/experiments/the-compile-is-per-file.md`.

## 2026-09-18 — 021 — One sandbox per release, and the prize was collected

- Status: advance
- Revision: `4e1c6bd`
- Model: `gpt-5.6-sol`
- Question: Was entry 020's zero a property of the change or of the order it was built in?
- Prior hypothesis: sharing a compilation directory pays nothing while every batch links its own sandbox, because Go's build IDs cover the package directories; stabilising the path first would let the toolchain's up-to-date check fire and collect the price entry 019 measured.
- Intervention: `GatedLaboratory` takes one sandbox and one compilation directory per release and reuses both across its batches. Reuse is safe because every batch restores the file it overwrote before returning, so the tree is pristine between batches.
- Control: `CompilationDirectories()` is the integer, and the guard asserts one sandbox and one directory for two batches. The compilation-directory half was seen refusing earlier — `expected 1, actual 2` — when the sharing rule was deliberately broken.
- Exact evidence:
  - ten-package module, forty mutants, three rotated rounds: ordinary 64,583 / 65,964 / 64,006 ms; gated 14,620 / 14,708 / 14,798 ms; ratio **0.2264 / 0.2230 / 0.2312**
  - the same fixture before this change: 0.4005 / 0.4040 / 0.3995
  - identical verdicts every round: 40 total, 20 killed, 20 survived, `40 of 40 mutants ran from one compilation`
  - full suite 502 tests; lint clean; ratchet 846 → 850, attributed per file (`gatedlaboratory.go` 39 → 42, `module_scope.go` 54 → 55)
- Wall-clock observation: the gated run fell from 25.6 s to 14.6 s on the same fixture, an 11 s saving against the 12.4 s entry 019 priced. The prediction was a ceiling and the real figure came in just under it.
- Verdict: Entry 020's zero was the order, not the change. The two halves are one change, and measuring them apart is what established that.
- What changed: The module path now pays one module-wide compile per release instead of one per source file with mutants.
- What remains unknown: Whether the same reuse helps the chain fixtures, which were not re-measured; and the same open items as before — deadline kills on the module path, a heavy suite, and this repository's own gate.
- Next falsifiable step: Re-measure the two chain fixtures through the shipped binary, and time this repository's own gate scope with the module path against the ordinary one.
- Artifacts: `docs/experiments/the-compile-is-per-file.md`, `docs/performance-core-metrics.md`.

## 2026-09-18 — 022 — The collision refusal is fixed: one compile batch per binary name

- Status: advance
- Revision: working tree on `85f2ea7`, with `internal/gobuildrunner/module_scope.go` and its two internal test files uncommitted
- Model: `deepseek-v4-flash`
- Question: How does the module scope compile a repository whose packages produce test binaries with the same name, without giving up the complete `./...` scope or the single-compile win a collision-free module has?
- Prior hypothesis: the defect is the binary name alone — `go test -c -o <dir>` refuses a duplicate basename per argument list — so grouping colliding packages into separate compile batches restores compilation while leaving a collision-free module paying exactly one compile.
- Intervention: `planCompileBatches` assigns every discovered package to the lowest-index compile batch free of its binary name (case-folded on Windows), each batch gets its own `batch-N` directory under the run's output directory, and `prepare` runs one `go test -c -o <batchDir> <that batch's sorted import paths>` per batch. The union of batch arguments equals the discovered scope; binary paths are assigned only to packages with tests; `validateBatches` keeps `errBinaryNameCollision` as an invariant that now only fires on an internal defect; an empty discovery fails closed with `errEmptyModuleScope`. On ditto the plan is three batches.
- Control: the pre-change binary on the same throwaway colliding fixture reported `Gated: none of 21 mutants ran from one compilation; 21 kept their own.` — so the two paths agreeing on totals and survivor addresses is a comparison between the same questions, not two different ones.
- Exact evidence:
  - the defect, measured before the change on an exact `.git`-free copy of `85f2ea7`: the scope's test binaries are named only from `path.Base(importPath)`, so `github.com/Disble/ditto`, `github.com/Disble/ditto/cmd/ditto` and `github.com/Disble/ditto/internal/ditto` all produce `ditto.test.exe` (and `dittotesting` collides twice); the real runner reported `Built=false`, `Discoveries=1`, `Compilations=0`, `PackageRuns=0` and `module test binary name collision`
  - a real `GatedLaboratory` probe on a gateable batch reported `Gated=0`, `FellBack=1`, with the delegate answering the batch; consequence: module-scope gating on ditto gated 0 of 850 mutants even though `schemata` can express 517 of them (60.8%), measured by `internal/perfbench/gating_test.go` in the same copy
  - toolchain facts, all reproduced in throwaway modules and quoted as such: the toolchain refuses a duplicate basename per argument list even when neither package has test files (`cannot write test binary calc.test for multiple packages`, exit 1, no test files anywhere); `go list -deps -test -json ./...` does not type-check, so a package with no tests and a type error passes discovery while `go test ./...` and `go test -c -o <dir> ./...` both exit 1 naming it; two sequential `go test -c -o <same dir>` compiles of same-named packages exit 0 while silently leaving one binary
  - through the shipped binary on a throwaway colliding module (two packages both producing `pay.test.exe`, 21 mutants, `--threshold 0`): ordinary 21 total / 15 killed / 6 survived; gated 21 total / 15 killed / 6 survived, with the line `Gated: 12 of 21 mutants ran from one compilation; 9 kept their own.`
  - the six sorted survivor addresses are byte-identical between the two paths (sha256 `c1957c66a5268e8a2d6d52667abdd353158648d4ce134ea6d74918f4ddb7e8ec`), and the two packages' tests were proven to run from `batch-0/pay.test.exe` and `batch-1/pay.test.exe`
  - the module scope now builds ditto itself, measured by driving the real runner against a `.git`-free copy of this tree: `Built=true`, `Discoveries=1`, `ToolchainStarts=4`, `Compilations=3`, `PackageRuns=45`, `SkippedPackages=0`, no failure text, 94.3 s. Three batches is the plan for this tree's three colliding basenames; the pre-change runner refused the same tree with `Built=false` and `Compilations=0`
  - repository-level evidence on the changed tree: `go test -count=1 ./...` exits 0 (45 packages `ok`, 20 with no test files, 0 failures, 109 s); `gofmt -l .` and `go vet ./...` clean; the ratchet moved 850 → 873 (+23, all of it `internal/gobuildrunner/module_scope.go`). An earlier state of the same change counted 874, and the extraction that brought `prepare`'s cyclomatic complexity inside the gate's `cyclop` limit moved the count by one; the number written down is the one the gate counted on the committed tree
- Wall-clock observation: the full suite took 109 s. A bounded release over `internal/fstemporarydir` — 20 mutants against a suite re-run of about 70 s per mutant — was killed by its own time budget after the baseline line and before any verdict, so no `Gated:` line was read for ditto's own tree. Reported, not gated.
- Verdict: the collision refusal is fixed, and the fix is corroborated twice — end to end on a colliding module through the shipped binary, and at the runner level on ditto's own tree, which now compiles into three batches and runs 45 package tests where it previously refused with `Compilations=0`. The realized gated share on ditto itself is NOT measured: the bounded release over `internal/fstemporarydir` died to its own time budget before any verdict, so no `Gated:` line exists for ditto's own tree. Said plainly rather than implied.
- What changed: a module whose packages collide now compiles and runs instead of refusing, at the cost of one compile invocation per collision batch; a collision-free module still pays exactly one.
- What remains unknown: the realized gated share on ditto (the syntactic ceiling is 517 of 850); the phase split after the shared sandbox change, which was never re-measured; whether compiling three batches per release costs anything measurable on a real tree; the deadline-kill and heavy-suite items the earlier entries left open.
- Next falsifiable step: a bounded release over ditto's own tree with at most three mutants, reading the `Gated:` line and comparing totals and survivor addresses against the ordinary path; then re-measure the phase split.
- Artifacts: `internal/gobuildrunner/module_scope.go`, its two test files, `perf/baseline.json`, this entry.

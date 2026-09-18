# Experiment — can one module-scope build replace per-mutant Go driver starts?

Written before the measurement on 2026-09-18. Predictions and kill criteria are fixed before any prototype or timing run.

## The research question

**To what extent** does compiling the configured `go test -count=1 ./...` scope once into package test binaries reduce Go driver starts without changing verdicts over twelve gateable comparison mutants from one package whose configured scope contains four package test binaries, at revision `5f65e3d3c4201689b81b707533b18aa42364ea8b`, on the throwaway fixture measured on 2026-09-18?

| Element | This question's |
| --- | --- |
| Interrogative phrase | To what extent |
| Variable, the counter that moves | Go driver starts per release; package-test executions are the scope-fidelity counter |
| Population, unit of analysis | Twelve comparison mutants from `pkg0`, judged by four package test binaries |
| Space and time | Revision `5f65e3d`, a disposable four-package module, 2026-09-18 |

**FINER** — Feasible: the existing schemata path already instruments comparison mutants and builds one package binary. · Interesting, the decision that turns on it: whether the core moves from one configured command per mutant to one module execution plan. · Novel: ditto reports per-file gated counts and its runner reports compilation/run counters, but nothing measures a full configured scope from prebuilt binaries. · Ethical: both the tool and fixture are throwaway copies outside every repository holding work. · Relevant: an exact reduction with scope-equivalent verdicts selects the production architecture; any scope disagreement rejects it.

**PICOT** — P: twelve comparison mutants from one package, with four package test binaries in scope. · I: instrument once, run `go test -c -o <dir> ./...` once, then execute every package test binary for the baseline and each selected mutant. · **C, the control**: ordinary `go test -count=1 ./...` once per baseline/mutant, plus a deliberately package-only runner that must miss a cross-package kill. · **O, the exact counters**: Go driver starts, package-test executions, ordered verdicts, and mutant addresses. · T: one discarded warm-up and three rotated measured rounds, bound to the revision/date above.

## Current baseline, preserved before improvement

These are the existing exact counters in `perf/baseline.json` before the experiment. They remain the incumbent contract while the new fixture measures the missing module-scope cost.

| Counter | Current value |
| --- | ---: |
| source parses per release with three viruses | 4 |
| AST walks per release with three viruses | 12 |
| laboratory runs over the whole fixture | 48 |
| test-command invocations over the whole fixture | 49 |
| files linked per sandbox | 6 |
| sandboxes built per sequential release | 1 |
| laboratory runs for one changed function | 4 |
| laboratory runs for one changed function in each of two files | 8 |
| mutants in one full release over this repository | 789 |

The architectural target is the `49` command invocations, not parsing, AST walking, or sandbox construction. The new experiment uses twelve mutants so that complete package-scope execution is directly observable: thirteen baseline/mutant selections across four package test binaries means **52 package-test executions** in both valid modes.

## Method

Create the experiment and its fixture in a clean feature worktree, then copy the complete tool tree to a disposable directory before every execution. The fixture itself is created under the disposable test process's temporary directory. Confirm the tool-copy revision and fixture module identity before recording a number.

The four package names are unique so `go test -c -o <dir> ./...` can produce one binary per package without the known duplicate-name refusal. Package `pkg0` owns one mutation that its own tests do not kill; a test in dependent package `pkg1` does. That mutant is the scope sentinel.

The modes are:

| Mode | Go driver starts | Required package-test executions |
| --- | ---: | ---: |
| A — ordinary configured scope per baseline/mutant | 13 | 52 |
| B — incumbent package-only prebuilt runner | 1 | fewer than 52; intentionally invalid scope control |
| C — module-scope prebuilt runner | **1** | **52** |

First make the new counter check fail deliberately. Then run A and the scope sentinel. B must disagree on that sentinel; if it does not, the harness cannot observe the defect it claims to prevent. C must restore A's ordered verdicts and addresses while preserving all 52 package-test executions.

Discard one warm-up. Run three measured rounds with the order rotated `A-B-C`, `B-C-A`, `C-A-B`. Exact counters and verdicts decide; wall clock is reported separately and compared only as a within-round C/A ratio.

## Hypotheses, and what kills each one

**H1 — the module-scope execution plan removes the Go driver toll without narrowing the configured scope.** Mode C produces the same twelve ordered verdicts and addresses as A, executes exactly 52 package tests, and reduces Go driver starts from 13 to exactly 1 (92.3%).
*Falsified if any verdict/address differs, package-test executions are not exactly 52 in either valid mode, or C starts the Go driver more than once.*

**H2 — the existing package-only architecture is observably narrower than `./...`.** Mode B compiles and runs only `pkg0`; it reports the scope-sentinel mutant as survived while A and C report it killed, and all three modes agree on the remaining eleven mutants.
*Falsified if B does not disagree on exactly the sentinel, or if C disagrees with A anywhere.*

**H3 — removing repeated Go driver starts is significant on the light TDD fixture.** In every measured round C/A wall clock is at most 0.25.
*Falsified if C/A exceeds 0.25 in any measured round.*

**What would refute all of them:** the ordinary control does not produce twelve mutants, thirteen Go driver starts, 52 package-test executions, and the intended mixed verdict population. That outcome means the fixture or instrument is wrong, not that either architecture won.

## Decision rule, fixed in advance

- Any A/C verdict, address, or package-test-execution mismatch → reject the module-scope architecture and do not change production code.
- B fails to expose the sentinel disagreement → reject the experiment as unable to measure scope fidelity; repair the fixture before drawing a conclusion.
- A and C agree exactly, C starts the Go driver once, but H3 is refuted → preserve the measurement but do not call the architecture a significant TDD-loop gain; return to the performance question.
- All three hypotheses corroborated → implement the smallest production slice that recognizes only the exact default Go scope, uses module-scope binaries for currently admitted schemata, and falls back for unsupported commands, duplicate package names, or failed builds.

No result in this experiment authorizes expanding mutation families or accepting arbitrary test commands.

## Results

Independently reproduced from a `.git`-free disposable copy whose complete tree manifest matched the source at SHA-256 `cd884647f1b2fc18fcc8f74ae9b31043f7f6f31b587321e2aa9ff40732ac62c7`. The experiment source hashed to `c5365c0cf00d201bb58ba2ad45623b6eda641d780c63e7fcc60fc16fa47e248c`. The copy was removed after evidence capture.

| Round | Order | A drivers / package tests | B drivers / package tests | C drivers / package tests | A wall | B wall | C wall | C/A |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 1 | A-B-C | 13 / 52 | 1 / 13 | 1 / 52 | 11.4070712 s | 1.1560115 s | 2.2254465 s | 0.1951 |
| 2 | B-C-A | 13 / 52 | 1 / 13 | 1 / 52 | 9.6900866 s | 984.1189 ms | 1.9315122 s | 0.1993 |
| 3 | C-A-B | 13 / 52 | 1 / 13 | 1 / 52 | 10.2873751 s | 960.3685 ms | 1.8605086 s | 0.1809 |

### Controls

**Incumbent baseline — passed before prototype work.** A verifier copied revision `5f65e3d` without `.git` to `/tmp/ditto-perf-baseline.kXLeHL`. Source and copy of `perf/baseline.json` both hashed to `c470245e542046aac1e486482a9338d002314573ff0393e6f5749de442e1a23d`. `(cd /tmp/ditto-perf-baseline.kXLeHL && go test ./internal/perfbench/)` exited 0 in 0.518 s, reproducing all nine counters in the table above. The disposable copy was removed after evidence capture.

**The exact counter can refuse — passed.** `TestExactCounterRefusalRejectsCounterDrift` supplied 52 executions against a deliberately wrong expectation of 51 and produced `C module-scope package-test executions = 52, want exactly 51`.

**The ordinary population exists — passed.** A produced exactly twelve mutants, thirteen Go driver starts, 52 package-test executions, six killed and six survived. Ordered addresses remained stable.

**The scope defect is observable — passed.** Sentinel `pkg0/values.go:4:16` was killed by A and C and survived under package-only B. B agreed with A on every other mutant, so the one disagreement belongs to the omitted dependent-package test rather than general harness drift.

**A killed mutant does not truncate scope — passed.** Every baseline/mutant selection asserts its own execution-log growth: +4 package binaries in A and C, +1 in B. The checks run after failed mutant commands too; all 52 required A/C executions were observed in every measured round.

### H1 corroborated

A and C produced identical ordered addresses and verdicts in all three rounds. C preserved all 52 package-test executions while reducing Go driver starts from 13 to 1, the predicted 92.3% reduction.

### H2 corroborated

Package-only B missed exactly the cross-package sentinel and no other verdict. The current one-package runner is therefore an observably narrower question than the configured `./...` scope.

### H3 corroborated

C/A measured 0.1951, 0.1993 and 0.1809, all below the pre-registered 0.25 kill line. On this light TDD fixture, the full-scope prototype was 5.0-5.5 times faster without skipping package tests.

## Verdicts: 3 of 3

## Conclusion

The module-scope execution plan is corroborated for the measured population. It removes twelve of thirteen Go driver starts, preserves the complete four-package scope, and restores the cross-package kill that the current package-only path misses. The decision rule selects the smallest production slice: recognize only the exact default Go scope, reuse currently admitted schemata, run every package test binary per selection, and fall back whenever the command, build, or binary layout cannot be represented faithfully.

This is evidence for the architecture, not yet for production code. The production slice must be re-measured through the real release path before any baseline moves.

## What this does NOT establish

The experiment does not establish support for custom shell commands, duplicate package names, build tags outside the active toolchain context, arbitrary `go test` flags, non-comparison schemata, multiple instrumented source files or packages with globally unique selectors, or correctness on a real repository. It measures the smallest architecture boundary first: whether a full configured Go scope can be compiled once and executed faithfully per selected mutant.

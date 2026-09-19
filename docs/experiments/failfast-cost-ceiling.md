# Experiment — how much safe performance headroom remains outside the configured tests?

Written before the measurement.

## The research question

**What fraction** of one complete ordinary fail-fast staged mutation run is spent inside the configured test command, over the same ten real mutants from `internal/schemata/instrument.go` at revision `5320273` on 2026-09-18?

| Element | This question's |
| --- | --- |
| Interrogative phrase | What fraction |
| Variable, the counter that moves | Sum of configured-command nanoseconds divided by end-to-end nanoseconds |
| Population, unit of analysis | One green baseline plus ten mutants from the exact task-8 staged scope |
| Space and time | Windows 11, revision `5320273`, disposable copies, 2026-09-18 |

**FINER** — Feasible: `CMDTestRunner.Test` owns the exact process boundary and needs only two monotonic-clock reads plus one append. · Interesting, the decision that turns on it: whether meaningful precision-preserving optimisation remains inside Ditto. · Novel: prior work timed the whole release and changed proxy counters, but did not sum the real configured command boundary. · Ethical: the tool and target are disposable archives; mutation never runs in the source worktree. · Relevant: a ≥90% test-command share closes performance work for now without weakening precision.

**PICOT** — P: eleven configured-command invocations (baseline + ten mutants). · I: timing only the existing `go test -count=1 -json -failfast ./...` process boundary. · **C, the control**: task 8's three uninstrumented ordinary rounds on the identical revision/scope (623, 662, 676 s) plus byte-identical observable output. · **O, the exact counter**: exactly eleven timing records and their summed nanoseconds; wall-clock share is reported, never made a repository gate. · T: one discarded warm-up, then three measured runs.

## Method

Create a `.git`-free tool copy and a separate self-contained scratch Git target from `git archive 5320273`. Stage the same semantics-neutral comments on `internal/schemata/instrument.go` lines 163 and 167. `ditto staged --dry` and the static planner must reproduce ranges `5162-5243,5260-5341`, ten generated mutants, and ten gateable selectors before any measured run.

In the disposable tool copy only, wrap `command.CombinedOutput()` in `internal/cmdtestrunner.CMDTestRunner.Test` with monotonic `time.Now`/`time.Since`, increment an invocation counter, and append `invocation`, `duration_ns`, and outcome class to a file named by `DITTO_PHASE_TIMINGS`. A second observation-only hook in `ConsoleReporter` writes each diagnostic label and reason to `DITTO_PROTOTYPE_REASONS`, so the task-8 reason hash can be compared; it does not alter execution or rendered output. Do not change command arguments, environment, deadline, sandbox, mutation, scheduling, or reporting.

The phase share for each run is `sum(duration_ns) / external_elapsed_ns`. Residual is the end-to-end duration minus that sum. There is only one measured mode, so mode rotation is inapplicable; task 8's three uninstrumented rounds are the already-run control. Run one warm-up and three measured instrumented rounds.

Controls before reading the share:

1. Exact same dry ranges and ten-mutant static shape.
2. Instrumented output matches task 8: 10 total / 8 killed / 2 survived; identical mutant, survivor, and reason hashes.
3. Exactly eleven monotonically numbered timing lines, all positive, and their sum does not exceed external elapsed time.
4. A deliberately wrong expectation of twelve timing lines must refuse before the correct eleven-line assertion is accepted.
5. Source worktree HEAD/status remain unchanged and no source code is written there.

## Hypotheses, and what kills each one

**H1 — the timing instrument observes the intended boundary without moving the answer.** Every measured run has exactly eleven positive timing records numbered 1–11, sum ≤ end-to-end time, and the exact task-8 composition and hashes.
*Falsified by any missing/duplicate/non-positive record, sum > total, red baseline, or observable mismatch.*

**H2 — configured tests consume at least 90% of the end-to-end run.** `sum(configured-command time) / total time >= 0.90` in each of the three measured runs.
*Falsified if any valid measured run is below 0.90.*

**What would refute all of them:** the exact ten-mutant shape cannot be reproduced or the timing file cannot be bound one-to-one to the eleven configured-command invocations. Then the instrument cannot answer the question and no headroom conclusion is allowed.

## Decision rule, fixed in advance

- H1 dies → discard the experiment and fix the instrument; no performance conclusion.
- H1 holds and H2 holds in all three rounds → record that safe Ditto-only headroom is at most 10% on this workload, mark no further precision-preserving range for now, and close the session.
- H1 holds and every share is 0.75–0.90 → meaningful residual remains; profile that residual in a later session without touching test scope.
- H1 holds and any share is below 0.75 → substantial Ditto overhead remains; continue core optimisation.
- Shares crossing decision bands → record H2 refuted but the next action undecidable; do not claim a ceiling.

## Results

Revision `5320273232f5993a06ff96c47887cffd5c8a032f`, tree `dc85e4232f422e1eba41054577ffef2551a22177`, Go 1.27.0 windows/amd64. `ditto staged --dry` selected `instrument.go` ranges `5162-5243,5260-5341`; the static planner reported `TOTAL=10 GATED=10 FALLBACK=0`.

Each run recorded exactly eleven configured-command invocations, numbered 1–11 with no gaps, all positive. Three invocations passed (the green baseline and the two survivors) and eight failed (the eight kills). Overhead measured here is the wall-clock difference between the launcher's own timestamps and the summed durations of the configured command. It is an upper bound on non-test time, not a breakdown: it necessarily also carries process setup and teardown, shell and timestamp reads, planning and staging before the first invocation, reporting and cleanup after the last, and ordinary machine noise.

| Run | Configured-command time | End-to-end | Share in commands | Non-test upper bound |
| --- | ---: | ---: | ---: | ---: |
| warm-up (discarded) | 621.148 s | 623.400 s | 0.996388 | 2.252 s |
| 1 | 618.717 s | 623.011 s | **0.993109** | 4.293 s |
| 2 | 644.116 s | 648.537 s | **0.993183** | 4.421 s |
| 3 | 601.268 s | 605.427 s | **0.993131** | 4.159 s |

Every measured run reported 10 total / 8 killed / 2 survived at score 0.80, and every run reproduced the recorded mutant, survivor and reason hashes of task 8: `19d0375420ff2925715423c320eb041b309e79d04d5b6c3eca27159f0630bf92`, `4d9c3c8bb9b82aea147d1236cc76e07d55fa8761d6dd10f548f2683a5599ce26`, `fbdda8119b0e46eb5c8f943f06af50754f25fa26b3bc233b06028939f3f5b4af` with 8 assertion kills and 2 unknown survivor reasons.

### Controls

- **Same scope and shape — passed.** Dry ranges and the ten-mutant, ten-gateable static plan matched task 8 exactly.
- **Instrument observes the boundary without moving the answer — passed.** Exactly eleven positive records numbered 1–11 appear in every run, and all three hashes reproduced.
- **Wrong-expectation refusal — asserted, weakly evidenced.** A shell comparison refused eleven records against an expected twelve. No artifact was retained for it, and it was a shell comparison rather than the shipped test suite, so it is reported as a check that was run by the author rather than as independently evidenced control.
- **Source worktree — passed.** HEAD unchanged, no source file differs, index empty, `git diff --check` clean. The documentation and task changes in that worktree are the record, not source edits.
- **Independent readback — passed.** A separate verifier recomputed the sum, total, share and residual of all four runs, re-derived all three hashes, confirmed the eleven-record shape and the absence of source changes, and confirmed this note had not been back-filled.

The instrumented tool copy differed from HEAD in exactly three files: the two env-gated observation hooks and the new static-planning test. Command arguments, environment filtering, deadline, sandbox, mutation and rendering were unchanged. No timing record carries a mutant identity; the one-to-one binding to the eleven invocations rests on the monotonic counter, which printed 1–11 once per run.

### H1 corroborated

The instrument met its contract in every measured run: eleven positive ordered records, command time strictly below end-to-end time, a green baseline, and the exact task-8 composition and hashes.

### H2 corroborated

Configured-command share was 0.993109, 0.993183 and 0.993131. No valid measured round fell below 0.90.

## Verdicts: 2 of 2

H1 corroborated. H2 corroborated.

## Conclusion

The registered rule fires: on this real staged scope, **at most about 0.7% of end-to-end time is outside the configured test command**, and the remaining test execution is 99.3% of the bill. Preserving the user's stated invariant — every viable mutant runs the complete configured suite, with verdicts, reasons, survivors and non-viable classification unchanged — leaves **no measured, precision-preserving performance range available now**. That is recorded as the branch's conclusion rather than pursued further.

This does not say the residual is unreachable forever. It says the only way this branch could have delivered a drastic gain was by running fewer tests, and that was explicitly refused because it trades away the accuracy the tool exists to provide.

## What this does NOT establish

One staged scope, ten mutants, one file, one repository, one machine and one operating system. It does not establish the same share for a repository-sized run, for a heavy suite, for another language or platform, or for the whole gate. The residual is an upper bound on non-test time and is not attributed to any Ditto component; nothing here measures where inside those seconds the time went. The binary's provenance rests on its build in the disposable copy, and the wrong-expectation control is author-reported rather than artifact-backed.

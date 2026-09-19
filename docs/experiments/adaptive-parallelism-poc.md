# Experiment — can bounded outer parallelism buy latency without making the laptop unusable?

Written before the measurement.

## The research question

**To what extent** does running the same real staged mutants through two or three concurrent ordinary test-command lanes reduce end-to-end latency over serial execution at revision `5320273`, while preserving every observable answer and retaining enough memory headroom to keep the machine usable; and, only if that static result is worth promoting, can admission react correctly when available memory changes during a run?

| Element | This question's |
| --- | --- |
| Interrogative phrase | To what extent |
| Variables | Maximum simultaneously active configured commands; end-to-end ratio; minimum available physical memory; aggregate child-process peak memory |
| Population, unit of analysis | The exact three-mutant real scope at `internal/schemata/gate.go:148`: one killed, one survived, one non-viable in the incumbent measurement |
| Space and time | Windows 11, revision `5320273`, `.git`-free disposable tool/target copies, 2026-09-19 |

**FINER** — Feasible: Ditto already has `Parallel()`, deferred results, and a sandbox pool; the PoC only needs observation hooks and an external Windows monitor. · Interesting: 99.3% of the measured bill is the configured command, so outer concurrency is the strongest remaining precision-preserving hypothesis. · Novel: existing work rejected parallelism from saturation experience but never measured bounded 1/2/3 outer lanes over the same real mutant population or a memory-aware admission policy. · Ethical: mutation, builds, tests, and prototype edits occur only in disposable copies; concurrency starts at two and three is attempted only if two leaves the registered headroom. · Relevant: a successful result opens one production design; a failed result closes it without weakening test precision.

**PICOT** — P: three real mutants from one exact source range. · I: the incumbent `Parallel()` mechanism at host limits two and three, followed conditionally by an adaptive admission-controller prototype. · **C:** the identical scope at host limit one, with the exact same configured command `go test -count=1 -json -failfast ./...`. · **O:** generated/scored/killed/survived/non-viable composition, sorted diagnostic identity/reason records, non-empty survivor address, observed maximum active configured commands, system-memory low-water and process-tree peak; wall-clock ratios are reported, never repository gates. · T: one discarded warm-up per mode, then three paired rounds with order rotated; long arms run one at a time and remain externally bounded.

## Fixture and instrumentation

Create a `.git`-free disposable copy from `git archive 5320273`. Add one build-tagged experiment test that calls `ditto.Release` over public changed range `4961-5031` of `internal/schemata/gate.go`, with `Parallel()`, threshold zero, and the exact fail-fast command. The inner command does not carry the experiment build tag, so it cannot recursively invoke the experiment.

In the disposable copy only:

1. Add an env-gated `CMDTestRunner` observation hook recording monotonically numbered command start/end events, active count, maximum active count, duration and outcome. It changes no command argument, environment, deadline, sandbox, verdict or output.
2. Add the same env-gated diagnostic hook previously used by the cost-ceiling experiment, recording every label, result class and reason at summary time.
3. Use an external Windows monitor written in Go. `GlobalMemoryStatusEx` samples total/available physical memory; a Job Object groups the measured process tree where the host permits nested jobs and reports `PeakJobMemoryUsed`. If job assignment is unavailable, the run retains system-memory telemetry and explicitly marks aggregate peak unavailable rather than inventing it.
4. Capture source revision/tree, disposable patch hash, Go version, command, exit code and external elapsed time for every arm.

The source worktree is never built, tested or mutated. Each measured arm is a fresh process. Its concurrency is selected by the host test binary's `-parallel=N`; the nested configured command remains byte-for-byte identical.

## Controls before timing conclusions

1. **Exact shape:** the scope produces exactly 3 generated / 2 scored / 1 killed / 1 survived / 1 non-viable, and the survivor comparison is non-empty.
2. **Reach:** mode 1 records maximum active commands exactly 1; mode 2 exactly 2. Mode 3 must record exactly 3 if admitted.
3. **Refusal:** a deliberate expectation that serial reached 2 must fail before accepting its correct maximum of 1.
4. **Fidelity:** every non-serial mode has byte-identical sorted diagnostic identities, survivor address and reason records to its serial control.
5. **Baseline:** the unmutated configured command is green in every arm.
6. **Isolation:** source HEAD, index and tracked source bytes remain unchanged; every mutation target is disposable.
7. **Memory instrument:** total and available memory samples are positive and available never exceeds total. Job peak, when available, is positive and no greater than its configured accounting domain permits.

## Operational safety rule

Three lanes are attempted only if every completed two-lane arm:

- emits no Windows low-memory notification;
- keeps minimum available physical memory at or above both 2 GiB and 15% of total physical memory; and
- exits normally without timeout or orphaned experiment processes.

If any condition fails, mode 3 is skipped as an explicit safety result, not retried. No deliberate memory-pressure process will be created on the development machine.

## Hypotheses and kill lines

**H1 — bounded outer parallelism is real and preserves the answer.** Modes 1, 2, and, if admitted, 3 reach exact maximum active-command counts 1, 2, and 3; every mode preserves the registered composition plus byte-identical identities, non-empty survivor address and reasons.
*Falsified by any requested maximum not being reached, any observable mismatch, red baseline, empty survivor comparison, or instrumentation ambiguity.*

**H2 — two lanes produce a meaningful end-to-end gain.** In each of three valid paired rounds, `T_two / T_serial <= 0.75`.
*Falsified if any valid round exceeds 0.75 after controls pass. Ratios straddling the line make the performance result undecidable rather than successful.*

**H3 — a third lane earns its extra resource cost.** If the safety rule admits mode 3, in each of three valid paired comparisons `T_three / T_two <= 0.90`, with no low-memory notification and the registered headroom retained.
*Falsified if any ratio exceeds 0.90 or any safety condition fails. If mode 3 is not admitted, H3 is recorded blocked-by-safety, never silently omitted.*

**H4 — adaptive admission can react without changing a verdict.** Conditional on H1 and H2 surviving, an injectable controller is driven through capacity `3 → 1 → 3`. Once capacity falls, it starts no replacement until active work drains to the new allowance; it never cancels active work; it later grows again; indexed results remain in input order. A critical-pressure signal returns an infrastructure refusal and produces no mutant result.
*Falsified by oversubscription after the drain boundary, failure to regrow, cancellation of active work, reordered/missing/duplicate results, or representing resource pressure as a killed mutant.*

**What refutes the whole experiment:** the exact three-mutant shape cannot be reproduced, the baseline is red, host `-parallel` does not control observed command overlap, or the instrumentation cannot bind events one-to-one to configured-command invocations. Then no performance or architecture conclusion is allowed.

## Decision rule, fixed in advance

- H1 dies: stop; current parallel machinery is not a trustworthy base.
- H1 holds and H2 dies: stop before adaptive work; dynamic admission cannot rescue concurrency that does not buy meaningful latency at two lanes.
- H1 and H2 hold, but the safety rule blocks three: build H4 with automatic capacity capped at two and record that the laptop, not CPU count, set the ceiling.
- H1/H2/H3 hold: build H4 with hard cap three.
- H4 holds: recommend a production design slice, still requiring re-measurement through the shipped CLI because the PoC runs through the library test entry point.
- H4 dies: do not promote dynamic parallelism; retain only the static evidence.

No production code is promoted by this experiment.

## Results

The first valid serial warm-up and the requested two-lane warm-up reproduced the exact registered composition: 3 generated / 2 scored / 1 killed / 1 survived / 1 non-viable. Their three sorted diagnostic records were byte-identical, sha256 `6c9f12c7958a3bbda54af6c3a1c08fe43f2a5ceb03ac2cd1c06d597de881f874`, including one non-empty survivor and reasons assertion / unknown / build-failed. Both baselines were green and both processes exited zero.

The observation hook recorded exactly four configured commands in each run (one baseline plus three mutants), each with one start and one end. Serial reached maximum active 1, as required. The requested two-lane mode also reached **maximum active 1**, not 2:

| Warm-up | Requested host `-parallel` | Configured-command starts | Observed maximum active | End-to-end | Minimum available | Peak job memory |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| serial | 1 | 4 | 1 | 162.936 s | 15,117,778,944 B | 3,655,020,544 B |
| requested two | 2 | 4 | **1** | 165.474 s | 13,868,216,320 B | 3,643,973,632 B |

Both Windows monitors collected positive samples with zero sample errors, assigned the process tree to a Job Object, and saw no low-memory notification. The deliberate wrong serial expectation `maximum == 2` refused with actual 1 before the correct expectation was accepted.

The cause is directly visible in the output and code path. Outer mutant subtests were registered and paused, but every configured command completed before those subtests continued. Running the host with `-v` makes `release.go` install `VerboseLaboratory`; that type always exposes `TestAll`. `TestingTLaboratory.TestAll` therefore calls the inner batch first, and `VerboseLaboratory.TestAll` serially falls back through every mutant before the reporting subtests are created. This is not a timing inference: the trace contains no overlap and the output orders all work before `=== CONT` for the three mutant subtests.

An earlier serial warm-up was discarded before this result because the observation variables leaked into inner `go test` processes and their own runner tests polluted the trace. The instrument was corrected in the disposable copy by stripping only the two prototype observation variables from the configured command environment; compilation was rechecked before these two valid runs.

### Verdicts: 4 of 4

- **H1 refuted.** The requested two-lane maximum was not reached. Fidelity held, but the hypothesis required both reach and fidelity.
- **H2 undecidable by rule.** There was no two-lane intervention to time; 165.474 / 162.936 compares two serial runs and says nothing about parallel performance.
- **H3 blocked by H1.** Mode 3 was not attempted because mode 2 did not exist in the observed execution.
- **H4 blocked by H1.** The registered decision rule stops before adaptive work when current machinery is not a trustworthy base.

## Conclusion

The current `Parallel()` path cannot serve as the PoC base under Ditto's real verbose mutation invocation: `-parallel=2` still executes one configured command at a time. No claim about whether actual outer concurrency is faster or memory-safe follows from these timings.

This corrects one premise of the pre-registration: Ditto does **not** already have a usable bounded outer scheduler in this path. The next question must prototype an explicit scheduler that owns command admission independently of `testing.T.Parallel` and the verbose decorator. That is a new intervention and receives a new pre-registration before it is built.

## What this experiment does not establish

It does not establish that outer parallelism is slow, fast, safe or unsafe; the intervention never engaged. It is bound to the library test entry point with verbose host output and does not establish behavior of a future CLI scheduler, Linux, the gated path, repository-sized scaling, or an optimal default. The Job Object accounts the measured process tree, not the whole machine; `GlobalMemoryStatusEx` observes the whole machine but is volatile by contract.

# Experiment — does an explicit outer scheduler earn adaptive parallelism?

Written after the incumbent `Parallel()` intervention was refuted and before the explicit scheduler was built.

## The research question

**To what extent** does a disposable scheduler that owns mutant admission independently of `testing.T.Parallel` reduce end-to-end latency for the exact three-mutant real scope at `internal/schemata/gate.go:148`, while preserving every observable answer and registered memory headroom; and, only if two static workers earn promotion, can that scheduler change admission safely while work is active?

| Element | This question's |
| --- | --- |
| Interrogative phrase | To what extent |
| Variables | Scheduler worker limit, maximum active configured commands, end-to-end ratio, minimum available physical memory, aggregate process-tree peak |
| Population | Three real mutants: one assertion kill, one survivor, one non-viable mutant |
| Space and time | Windows 11, revision `5320273`, disposable copy, 2026-09-19 |

**FINER** — Feasible: the prior experiment validated the fixture, telemetry, exact observables and process monitor; only admission was missing. · Interesting: the current public parallel option was measured inert under the real verbose invocation, so a scheduler is the unresolved mechanism rather than an optimization detail. · Novel: no Ditto path has measured explicit bounded command admission over an identical real mutant population. · Ethical: all code and execution remain in the existing disposable copy; no deliberate memory pressure is created. · Relevant: two-worker latency and headroom decide whether adaptive control deserves production design.

**PICOT** — P: the same three mutants and four configured-command starts (baseline plus mutants). · I: a prototype-only `TestingTLaboratory.TestAll` branch submits every ordinary mutant first and admits delegate execution through an explicit worker limit; reporting still consumes indexed futures in input order. · **C:** the same scheduler at limit one, not the old `testing.T.Parallel` route. · **O:** exact composition and diagnostic hash, maximum active command count, start/end balance, elapsed ratio, minimum available physical memory and Job Object peak. · T: one discarded warm-up per mode, then three paired rounds with order rotated.

## Intervention boundary

In the disposable copy only:

- `DITTO_POC_WORKERS=N` activates an explicit scheduler branch before the delegate's batch interface can serialize work.
- It allocates one deferred result per input mutant, starts at most N delegate calls concurrently, and keeps result indexes unchanged.
- It does not call `testing.T.Parallel`; reporting subtests await already-submitted futures.
- `DITTO_POC_WORKERS` and the two observation variables are stripped from the configured child command so nested tests cannot recursively activate or contaminate the prototype.
- The experiment test no longer requests public `Parallel()`; all concurrency must come from this intervention.

No production file in the source worktree is edited. The existing Windows monitor and exact observation hooks from the refuted experiment remain the instrument.

## Controls before timing conclusions

1. **RED reach control:** a focused fake-delegate test requesting two workers must fail against the unmodified scheduler branch by observing maximum active 1, then pass after the branch reaches exactly 2.
2. **Serial control:** worker limit 1 records exactly four starts/ends and maximum active 1.
3. **Two-worker reach:** limit 2 records exactly four starts/ends and maximum active 2.
4. **Fidelity:** every mode reproduces 3 generated / 2 scored / 1 killed / 1 survived / 1 non-viable plus the serial diagnostic hash and non-empty survivor.
5. **Order:** a focused test completes workers out of order and observes returned futures in original input order.
6. **Baseline and isolation:** every unmutated suite is green; source HEAD/index/tracked bytes remain unchanged.
7. **Memory:** monitor samples are valid; no low-memory notification; two-worker minimum remains at least both 2 GiB and 15% of total.

Mode 3 uses the same operational safety rule as the prior note: it is attempted only if every two-worker arm retains the registered headroom, emits no low-memory notification and exits without timeout/orphans.

## Hypotheses and kill lines

**H1 — the explicit scheduler reaches its bound and preserves the answer.** Limits 1 and 2 produce maximum active counts 1 and 2, balanced starts/ends, input-order results, and identical registered observables.
*Falsified by missed/exceeded concurrency, missing/duplicate/reordered results, any observable mismatch, a red baseline or ambiguous trace.*

**H2 — two explicit workers produce a meaningful gain.** In each of three valid paired rounds, `T_two / T_one <= 0.75`.
*Falsified if any valid ratio exceeds 0.75; ratios crossing the line make H2 undecidable.*

**H3 — a third worker earns its resource cost.** If admitted, in each paired comparison `T_three / T_two <= 0.90` while all safety conditions hold.
*Falsified by any ratio above 0.90 or safety failure; if not admitted, record blocked-by-safety.*

**H4 — capacity changes govern admission without corrupting results.** Conditional on H1 and H2, an injectable capacity sequence `3 → 1 → 3` starts no replacement after the drop until active work drains to one, never cancels active work, later grows again, preserves indexed results, and maps critical pressure to infrastructure refusal rather than a mutant result.
*Falsified by post-drain oversubscription, cancellation, failure to regrow, result corruption, or a resource-pressure kill.*

**What refutes the entire experiment:** the validated three-mutant fixture changes shape, the baseline becomes red, or the observation hook cannot bind exactly four outer configured-command invocations. No timing conclusion is then allowed.

## Decision rule

- H1 dies: remove the prototype and close outer scheduling.
- H1 holds but H2 dies: stop before adaptive work; complexity cannot rescue insufficient static value.
- H1/H2 hold and three is blocked: test H4 capped at two.
- H1/H2/H3 hold: test H4 capped at three.
- H4 holds: recommend the smallest production slice, requiring later shipped-CLI remeasurement.
- H4 dies: do not promote adaptive scheduling; retain static evidence only.

## Results

Revision `5320273232f5993a06ff96c47887cffd5c8a032f`, tree `dc85e4232f422e1eba41054577ffef2551a22177`, Go `1.27.0 windows/amd64`. The disposable scheduler and its focused tests were bound by these file hashes:

- scheduler: `f5b2448fc159d65b7b65efe5509cb2ca5ac4976f1d7ae4fced3461b484412fd7`
- focused controls: `9cdbb14e5630516024f3233c68f7dc8de58f092ebdcf782db45f1776e60ff1b3`
- process observation hook: `028f6a7fb68aec636a2fa4536799537f7074f7b6b798a33107bacb3cd7141a39`
- diagnostic hook: `9fcac696683220b910e33a03e119ab149c1ab5e52c8628f1f097aa88d3dc4b33`
- real-scope harness: `594ad50a459471d4f26da925e9a8a468176af3ac1eb5af3f32f59f821acf6132`
- Windows monitor binary: `e79ff2734e3690e01c77af4cc17d6983fbd695ec5a412f7d4f68a28f90eef8a7`

### Mechanism controls

The reach check was RED before implementation: requested workers 2, observed maximum active delegate calls 1. The explicit scheduler made it GREEN at exactly 2. A second control deliberately completed workers in order `second, third, first` while the returned result slice remained `first, second, third`. After green, a manual mutation fixed permit capacity to 1; the reach test failed again with actual 1 / wanted 2. Restoring the capacity returned both focused controls green.

### Discarded warm-ups

Both warm-ups reproduced 3 generated / 2 scored / 1 killed / 1 survived / 1 non-viable, four balanced configured-command starts/ends, green baseline, and diagnostic sha256 `6c9f12c7958a3bbda54af6c3a1c08fe43f2a5ceb03ac2cd1c06d597de881f874`.

| Mode | Maximum active | End-to-end | Minimum available | Peak job memory | Low-memory notification |
| --- | ---: | ---: | ---: | ---: | --- |
| one worker | 1 | 165.707 s | 15,351,324,672 B | 4,377,591,808 B | no |
| two workers | 2 | 156.118 s | 16,199,344,128 B | 3,422,113,792 B | no |

Warm-ups are reported and not used for H2.

### First valid paired round

Order: one worker, then two workers.

| Mode | Maximum active | End-to-end | Minimum available | Peak job memory | Low-memory notification |
| --- | ---: | ---: | ---: | ---: | --- |
| one worker | 1 | 163.748 s | 15,137,050,624 B | 3,655,745,536 B | no |
| two workers | 2 | 154.543 s | 17,354,833,920 B | 3,757,441,024 B | no |

`T_two / T_one = 0.943784`: **5.62% less wall time**, against the registered requirement `<= 0.75` (at least 25% less).

Every registered observable matched. Each mode recorded exactly four starts and four ends; maximum active was exactly 1/2; both exited zero; both had positive telemetry with zero sampling errors and Job Object assignment; both retained far more than 2 GiB and 15% physical-memory headroom; neither emitted a low-memory notification. Sorted diagnostics remained byte-identical at sha256 `6c9f12c7958a3bbda54af6c3a1c08fe43f2a5ceb03ac2cd1c06d597de881f874`, including the non-empty survivor.

The command-duration arithmetic shows why the overlap barely moved the outer clock, without establishing a cause: sorted mutant-command durations were 2.711 / 27.309 / 64.069 s with one worker and 12.727 / 50.439 / 72.611 s with two. Every configured mutant command took longer in the overlapping run. Naming CPU, disk, compiler cache or another shared resource as the cause would require a separate experiment.

H2's kill line was **any** valid paired ratio above 0.75. The first valid ratio was 0.943784, so H2 was refuted immediately. The remaining two planned rounds were not run: repeating a hypothesis after its registered kill condition fired would spend approximately ten more minutes without changing its verdict. Mode 3 and adaptive admission were conditional on H2 and therefore were not built or measured.

### Independent readback available in this session

A separate parsing pass over the raw JSON/CSV/JSONL/output artifacts re-derived every start/end count, maximum active value, composition line, diagnostic hash, memory invariant and the 0.943784 ratio. Package-owned subagent verification was unavailable because this Pi session is rooted in a different Git clone; no claim of independent-agent verification is made. Source HEAD and tracked Go bytes remained unchanged and the index stayed empty.

## Verdicts: 4 of 4

- **H1 corroborated.** The explicit scheduler reached exact bounds 1 and 2, preserved input-order results despite out-of-order completion, and preserved every real-scope observable.
- **H2 refuted.** The first valid paired ratio was 0.943784, above 0.75.
- **H3 blocked by H2.** The third worker was not attempted; the block was insufficient two-worker value, not memory safety.
- **H4 blocked by H2.** Per the registered rule, adaptive admission was not built because the static work it would govern saved only 5.62% on the decision scope.

## Conclusion

An explicit Go scheduler can run two complete mutation commands concurrently without changing the answer or creating memory pressure on this fixture. That feasibility question is answered positively. The performance question is answered negatively for the registered scope: two workers recovered only 5.62%, because the overlapping configured commands themselves became substantially slower.

The registered decision rule closes adaptive admission for now. A memory-aware controller can regulate concurrency, but on this measured workload it would regulate a lever that did not earn enough latency to justify production complexity. No production code is promoted.

## What this experiment does not establish

One three-mutant scope cannot prove that every repository or a much slower, less internally parallel suite gains only 5.62%. It does not identify why individual commands slowed under overlap. It does not measure mode 3, repository-sized scaling, Linux, the shipped CLI or gated-path concurrency. It also does not refute adaptive scheduling as a general technique; it refutes promoting it from this exact result under the pre-registered 25% threshold. Memory snapshots cannot guarantee that unrelated applications will not allocate immediately afterward.

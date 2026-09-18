# Experiment — what the gated path actually buys, through the shipped binary

Written before the measurement on 2026-09-18, at revision `243f2ad` plus the
verdict-reason change.

## The research question

**To what extent** does `ditto run --gated` reduce the wall clock of a release over a light multi-package module, against the same release without it, in a throwaway project on 2026-09-18?

| Element | This question's |
| --- | --- |
| Interrogative phrase | To what extent |
| Variable, the counter that moves | Gate counts and wall clock; verdicts and addresses are the fidelity counter |
| Population, unit of analysis | One release over a three-package module with a light suite |
| Space and time | Revision at the top of `perf/module-scope-core`, 2026-09-18 |

**FINER** — Feasible: the binary is built from a disposable copy and pointed at a throwaway project. · Interesting: whether the architecture earns its keep end to end, which is the request that started this work. · Novel: every earlier measurement ran a fixture harness or a build-tagged experiment, never the shipped command. · Ethical: no repository with work is touched; both the tool copy and the project are disposable. · Relevant: it is the only number that can say the module path pays for the mutants it added.

**PICOT** — P: one release, three packages. · I: `--gated`, which selects the module-scope runner for the default command. · **C**: the same release over the same project without `--gated`. · **O**: the gate counts the report prints, the ordered survivor addresses, and the wall clock. · T: one run per mode, with the ordinary mode run first so a cold toolchain cannot favour it.

## Why the shipped binary and not a harness

The repository's own rule: observable output is verified by running the binary, not by reading the suite. A harness measures the mechanism; this measures what a person gets. The mechanism was already measured at 0.1951, 0.1993 and 0.1809 of ordinary in `docs/experiments/module-scope-runner.md`, so a number far above that here would mean the pipeline around it — instrumentation, sandbox, the `test2json` conversion — is what costs, and that is worth knowing.

## Hypotheses, and what kills each one

**H1 — the gated path preserves every verdict.** The survivor addresses and the totals are identical between the two modes.
*Falsified if any address, total, killed count or survived count differs.*

**H2 — the gated path is substantially faster end to end.** Gated wall clock is at most 0.50 of ordinary.
*Falsified if the ratio exceeds 0.50.*

**H3 — the gated path reports that it gated.** The run prints a gate count above zero, so a run that silently took the ordinary path cannot be mistaken for one that gated.
*Falsified if the count is `none`.*

**What would refute all of them:** the ordinary run does not produce the mutants the fixture was built to contain. That means the project or the binary is wrong, not that either mode won.

## Decision rule, fixed in advance

- H1 refuted → the module path must not be reachable from `--gated`, and the change is reverted.
- H1 and H3 corroborated but H2 refuted → the architecture is recorded as correct but not worth it on a light suite, and the default stays as it is.
- All three corroborated → the gain is ratified, and the recorded counter moves only if a recorded counter moved.

## Method

The tool is built from a complete `.git`-free disposable copy, so no build artefact lands in the tree being measured. The project is a three-package module with a light suite, created fresh for this run, and ditto mutates it in a sandbox rather than in place, so the project is not a copy with work in it.

Wall clock is reported and never gated; the counters the report prints are the contract.

## Results

Measured 2026-09-18 from a complete `.git`-free disposable copy, with the binary built from that copy and pointed at a throwaway three-package module. Toolchain Go 1.27, windows/amd64.

| Mode | Wall clock | Total | Killed | Survived | Gate line |
| --- | ---: | ---: | ---: | ---: | --- |
| Ordinary | 23,145 ms | 30 | 24 | 6 | — |
| `--gated` | 7,361 ms | 30 | 24 | 6 | `30 of 30 mutants ran from one compilation` |

Ratio: **7,361 / 23,145 = 0.318**, or **3.14× faster**.

### Controls

**The fixture contains survivors — passed.** Six mutants survive in both modes, so the address comparison below is over six real survivor reports rather than over an empty list. The first version of this fixture produced zero survivors, which would have made that check vacuous.

**The gated path actually engaged — passed.** The report says `30 of 30`, not `none`. The first run of this experiment said `none of 24`, and the cause was the fixture rather than the product: the generated sources had a trailing blank line, so they were not gofmt-formatted, and `schemata.Plan` refuses a difference that carries formatting. That is a real property of the product — a repository whose sources are not gofmt'd gets no gating at all — and it is reported rather than hidden.

**The fixture is green before mutation — passed.** `go test ./...` in the project passes before either run, so a red baseline cannot be mistaken for a killed mutant.

### H1 corroborated

Totals, killed and survived are identical, and the sorted survivor addresses are byte-identical between the two modes — six survivor reports, each with its line, column, virus and replaced text.

### H2 corroborated

0.318 against a kill line of 0.50. The end-to-end gain is smaller than the 0.18-0.20 the mechanism showed in `module-scope-runner.md`, which is the expected direction: a release also pays for parsing, instrumentation, the sandbox, the progress line, and now one `test2json` conversion per killed selection, none of which the mechanism measurement contained.

### H3 corroborated

The gate line reports `30 of 30`. A run that silently took the ordinary path cannot be mistaken for one that gated, which is the property backlog entry 11 cost three measurements to learn.

## Verdicts: 3 of 3

## Conclusion

The decision rule selects the third outcome: the gain is ratified. On a three-package module with a light suite, `--gated` produced the same verdicts and the same survivor addresses in about a third of the wall clock, through the shipped command rather than a harness.

No recorded counter moved as a result of this experiment: it measures a run, and the counters that gate this repository measure selection and the fixture. `perf/baseline.json` therefore stays where the previous entry left it.

## What this does NOT establish

A three-package module with a light suite is the case gating is for and also the case most favourable to it. A slow suite is not represented: the fixed toll being removed is a smaller share of a bill dominated by real test work, and an earlier measurement put that case at 0.50-0.58 rather than 0.32.

This repository's own gate was not run. It is repository-sized, takes tens of minutes, and 24 of its mutants were added by the change under measurement; whether the module path buys those back is still unmeasured and is the next question rather than an implication of this one.

The experiment also says nothing about `--confirm-kills`, whose module-path behavior the reason change enables but which no run here exercised.

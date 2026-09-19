# Experiment — can shared compilation preserve fail-fast and beat its real cost?

Written before the measurement.

## The research question

**To what extent** does a throwaway module runner that carries fail-fast through admission, package binaries, and observer-package stopping reduce end-to-end latency over the ordinary fail-fast command for ten gateable mutants in `internal/schemata/instrument.go` at revision `5320273`, without changing any observable answer?

| Element | This question's |
| --- | --- |
| Interrogative phrase | To what extent |
| Variable, the counter that moves | Package binaries started per selection; end-to-end ratio is reported beside it |
| Population, unit of analysis | Ten statically gateable mutants from two real staged lines in one Ditto source file |
| Space and time | Windows 11, revision `5320273`, disposable copies, 2026-09-18 |

**FINER** — Feasible: the module runner, counters, staged scope, and shipped binary already exist; only the fail-fast policy is missing. · Interesting, the decision that turns on it: whether this architecture remains the route to Ditto's primary goal of drastic real staged-run latency reduction. · Novel: entry 024 measured fail-fast and shared compilation only in isolation; nobody has combined them. · Ethical: both tool and mutation target are disposable copies outside every repository holding work. · Relevant: a successful result promotes a production TDD slice; a failed result stops module-path optimisation.

**PICOT** — P: ten real gateable mutants in one staged file. · I: exact fail-fast command admitted to a shared module compile, `-test.failfast` passed to package binaries, and later observer binaries stopped after the first killing failure. · **C, the control**: ordinary `go test -count=1 -json -failfast ./...` over the identical staged tree; focused controls separately remove admission, the binary flag, and early stopping. · **O, the exact counter**: generated/scored/killed/survived/non-viable composition, `Gated`/`FellBack`, package runs and stopped packages; wall-clock ratio is reported but never becomes a repository gate. · T: one discarded warm-up per mode, then three measured rounds with order rotated.

## Method

Create two copies from a verified `git archive` of `5320273`: a `.git`-free tool copy patched only for the prototype, and a self-contained scratch Git repository used as the mutation target. Never run mutation, builds, or tests in the source worktree.

Stage semantics-neutral trailing comments on `internal/schemata/instrument.go` lines 163 and 167. Before any mutation run, use `ditto staged --dry` plus a zero-test-command static planner to confirm the exact byte ranges, exactly ten generated mutants, and selector non-zero for all ten. If that shape is not exact, stop and rewrite this note before measuring.

The prototype admits only `go test -count=1 -json -failfast ./...`; near-miss commands remain disabled. It passes `-test.failfast` to package binaries and, only in that new mode, stops later observing package binaries after the first failure. The default module runner remains the deliberate no-early-stop control.

Run and tick these controls before reading performance:

1. **Reach refusal:** remove the one admission mapping; the shipped path must print `Gated: none of 10`.
2. **Binary fail-fast:** a package with a failing first test and a second marker-writing test writes the marker without `-test.failfast` and does not write it with the flag.
3. **Observer stopping:** the fail-fast runner reports `StoppedPackages > 0` for at least one killed selection; disabling the stop restores those package runs.
4. **Default invariance:** existing complete-scope and red-package continuation scenarios still run every configured package under the default constructor.
5. **Fail-closed:** discovery, compile, missing-binary, empty-scope and untested-package type-error controls refuse before package execution.
6. **Counter refusal:** deliberately wrong expected package/stopped counts must make the experiment harness fail.

Discard one warm-up per mode. Then run three measured rounds, alternating `ordinary → combined`, `combined → ordinary`, `ordinary → combined`. Compute each ratio inside its round. Do not average absolutes across rounds.

## Hypotheses, and what kills each one

**H1 — the combined runner preserves the configured answer.** In every measured round, both modes produce identical generated, scored, killed, survived and non-viable counts; identical sorted mutant identity lines; identical non-empty survivor address lines; and identical kill-reason multisets. The combined path reports `Gated: 10 of 10` and zero fallback.
*Falsified if any observable differs, any selector falls back, the survivor comparison is empty, or a baseline is red.*

**H2 — fail-fast removes real package work inside the shared runner.** The combined path records at least one stopped observer package on a killed selection, and disabling early stop increases package runs by exactly the stopped count while leaving verdicts unchanged.
*Falsified if `StoppedPackages == 0`, the package-run identity does not balance, or removing the stop changes a verdict.*

**H3 — the combined path is a drastic end-to-end win against the real denominator.** In each of the three measured rounds, `T_combined / T_ordinary_failfast <= 0.70`.
*Falsified if any round exceeds 0.70 after the warm-ups and all controls are green.*

**What would refute all of them:** the exact ten-mutant staged shape cannot be produced, admission does not engage despite the intentional mapping, or the prototype's baseline is red. That means the instrument is not asking the registered question; no architectural conclusion is allowed.

## Decision rule, fixed in advance

- Any H1 mismatch → kill the direction; speed cannot buy a different answer.
- H1 holds but H2 dies → kill the mechanism; it does not remove the work it claims to remove.
- H1 and H2 hold, H3 holds in all three rounds → promote the combined policy to a production TDD work unit, still re-measured through the real shipped binary before release.
- H1 and H2 hold, H3 dies → stop module-path optimisation as the primary direction and return to the end-to-end question with a new conjecture.
- Ratios straddling 0.70 across rounds → record H3 undecidable and do not promote anything; use a larger real staged scope only if the exact counters show more removable work exists.

## Results

The source identity was `5320273232f5993a06ff96c47887cffd5c8a032f`, tree `dc85e4232f422e1eba41054577ffef2551a22177`, with Go `1.27.0 windows/amd64`. The final throwaway patch has sha256 `8bbd3d628bddf3a76b8ef9afa1281c15bf48a8ea3f9dd81254d448d678549e51`; the prototype binary has sha256 `d2253e90a190a9bfe1762b361a4f7ebe39cfa9cb61f3158d98a9b0481de0d88a`.

`ditto staged --dry` selected `instrument.go` ranges `5162-5243,5260-5341`. The static planner produced exactly ten mutants and ten non-zero selectors. Every measured run reported 10 total / 8 killed / 2 survived. Every combined run reported `Gated: 10 of 10 mutants ran from one compilation; 0 kept their own.`

Warm-ups, discarded as registered: ordinary reach control 622 s, `none of 10`; combined 692 s, `10 of 10`. The measured rounds were:

| Round | Order | Ordinary fail-fast | Combined | Combined / ordinary |
| --- | --- | ---: | ---: | ---: |
| 1 | ordinary → combined | 623 s | 703 s | **1.1284** |
| 2 | combined → ordinary | 662 s | 715 s | **1.0801** |
| 3 | ordinary → combined | 676 s | 705 s | **1.0429** |

Across all six measured runs:

- sorted mutant identities were byte-identical, sha256 `19d0375420ff2925715423c320eb041b309e79d04d5b6c3eca27159f0630bf92`
- the two non-empty survivor lines were byte-identical, sha256 `4d9c3c8bb9b82aea147d1236cc76e07d55fa8761d6dd10f548f2683a5599ce26`
- sorted reason records were byte-identical, sha256 `fbdda8119b0e46eb5c8f943f06af50754f25fa26b3bc233b06028939f3f5b4af`: 8 assertion kills and 2 unknown survivor reasons
- every run exited zero against a green baseline

Every combined round ended on the same exact counters: 11 selections (one unselected baseline plus ten mutants), 1 discovery, 4 toolchain starts, 3 compilations, 221 package runs, 208 packages stopped by fail-fast, 66 packages skipped by observability closure, and 8 converter starts.

### Controls

- **Reach refusal — passed.** The unmodified binary with the same custom command and `--gated` reported `none of 10`; the prototype reported `10 of 10`.
- **Binary fail-fast — passed, after RED.** The first focused run deliberately used the default runner and failed because the later test wrote its marker. The fail-fast constructor suppressed that marker.
- **Observer stopping — passed.** The focused fixture ran 3 package binaries and stopped 1; disabling early stop ran 4 and stopped 0 with the same killed verdict. The real measured scope recorded 208 stopped packages. The exact 221+208 identity was not independently rerun at real-tree scale with stopping disabled; the focused fixture owns that causal control.
- **Default invariance — passed.** The existing complete-scope and red-package continuation scenarios still ran every configured package under the default constructor.
- **Fail-closed — passed.** Discovery diagnostics, compile failure, missing binary, empty scope and an untested package type error all refused before package execution.
- **Counter refusal — passed.** Replacing the expected stopped count 1 with 999 made the focused control fail with actual 1; restoring it returned green.
- **Independent readback — passed.** A separate verifier reproduced the ranges, shape, all six compositions, hashes, reason multiset, counters, ordering and ratios, and confirmed the source worktree had no code changes.

### H1 corroborated

All registered observables were identical in every measured round, the survivor comparison was non-empty, all ten selectors gated, and no fallback occurred.

### H2 corroborated

The combined runner removed observer-package work: 208 package starts were stopped in every real-tree run. The focused disable/enable control balanced one stopped package against one restored run without moving the verdict.

### H3 refuted

The prediction required every ratio to be at most 0.70. The measured ratios were 1.1284, 1.0801 and 1.0429. The combined prototype was slower in every round despite stopping 208 package binaries.

## Verdicts: 3 of 3

H1 corroborated. H2 corroborated. H3 refuted.

## Conclusion

The registered decision rule selects the fourth outcome: **stop module-path optimisation as the primary direction**. Shared compilation plus fail-fast preserved the answer and removed a large exact count of package starts, but it did not reduce end-to-end latency; it increased it in all three rotated rounds. The package-run counter is therefore not a sufficient proxy for the user's latency on this workload, and the stopped work does not rescue this architecture.

No production code is promoted. The next investigation must return to the end-to-end question with a new conjecture rather than profile or refine this module path again.

## What this does NOT establish

This experiment does not identify which remaining phase makes the combined path slower; naming compilation, sequential package execution, conversion, or scheduling as the cause would require a new pre-registered experiment. It does not establish package-local dharness compatibility, repository-wide latency, another operating system, or production maintainability. The exact early-stop balancing control is fixture-sized; the real-tree run independently establishes 208 stopped packages but did not rerun all 429 potential package starts with stopping disabled.

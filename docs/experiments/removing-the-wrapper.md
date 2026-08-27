# Experiment — how much of `tools/mutationstaged` ditto's own API already replaces

Written before the measurement. Nothing below has been run.

## The research question

**To what extent** does ditto v0.3.0's published API reproduce the behaviour of
`tools/mutationstaged` **without the wrapper**, over the 10 wrapper files and the
5-variable environment contract they carry, on dharness `79c9df7` and ditto
`main` (v0.3.0), August 2026?

| Element | This question's |
| --- | --- |
| Interrogative phrase | to what extent the API reproduces the wrapper |
| Variable, the counter that moves | reported mutant addresses (`path:line:col`) and their count; test-command invocations |
| Population, unit of analysis | the mutants of one staged scope of dharness's own tree |
| Space and time | dharness `79c9df7`, ditto `main` v0.3.0, Go 1.26, August 2026 |

**FINER** — *Feasible*: each hypothesis is one release against a throwaway copy,
minutes each. *Interesting*: the decision is whether autoreas-bridge copies ten
files or runs a command. *Novel, and what the tool already reports*: **one item
of the previous breakdown was already shipped and is recorded below as a
correction** — checking this is what found it. *Ethical*: scratch copies built
and destroyed; never a checkout with work in it. *Relevant*: dharness's
`tools/mutationstaged`, and ditto's `backlog.md`.

**PICOT** — **P** the mutants of one staged scope · **I** ditto's API standing
alone · **C** the same scope through the wrapper as it is today, plus a
per-hypothesis control that removes the variable under test · **O** the set of
reported addresses, compared as a set, and invocations of the test command — all
exact integers · **T** one release per variant, ditto pinned by revision.

## Method

Two throwaway copies: dharness's tree and a fixture project, both copied out,
never the checkouts. Fixture identity confirmed by `git rev-parse` before
anything is measured. Every comparison is address-by-address, never by total —
two runs agreeing on a count while disagreeing on which mutants is the failure
mode `backward-compatibility.md` was written to catch.

Each hypothesis carries a control that must MOVE. The trap named in
`state-of-the-work.md` is a path that was never reached: agreement reads as
agreement and is the code being off. A control that cannot be seen to move
disqualifies the agreement it was meant to protect.

## The hypotheses

**H1 — `buildIgnorePattern` is already dead on the derived path.**
`scopedrepository.go:41` drops any file the scope does not name, so the RE2
alternation dharness builds over every tracked `.go` selects nothing that
`WithChangedRanges` had not already excluded.

*Prediction: a release with `WithChangedRanges` alone and one with
`WithChangedRanges` + `IgnoreSourceFiles` report the identical set of addresses —
zero differing on either side.*

*Falsified if any address appears in one set and not the other.*

*Control that must move: the fail-open whole-file path, where `plan.encoded` is
empty and `WithChangedRanges` is never applied. Removing `IgnoreSourceFiles`
there MUST change the count. If it does not, the harness is not reaching the
option and H1's agreement establishes nothing.*

**H2 — ditto `main` already refuses a red baseline, so `baseline.go` is
redundant.** `internal/laboratory/laboratory.go:82` runs the suite once on
unmutated code and refuses. It is **not in any published version**: the fix is
commit `a7259a0`, and the v0.3.0 tag is `a97e04a`, which precedes it. dharness
pins v0.2.0. So this hypothesis cannot be tested against a released ditto, and a
`replace` to a copy of `main` is what it needs.

*Prediction: on a red-baseline fixture, dharness at ditto `main` with
`baseline.go` DELETED reports no score and exits non-zero — the same verdict it
gives with the file present.*

*Falsified if the run without `baseline.go` reports any score at all.*

*Control that must move: the same fixture at ditto v0.2.0, which must show the
false perfect score. If v0.2.0 also refuses, the fixture does not contain the
case and neither result counts.*

**H3 — ditto's baseline costs no more than the wrapper's.** The wrapper refuses
in the sandbox before ditto starts; ditto refuses inside the laboratory, after
materialising. The order may cost a materialisation.

*Prediction: test-command invocations before refusal are equal on both paths,
difference of 0.*

*Falsified if ditto's path runs the suite more than once before refusing, or
materialises a sandbox the wrapper's path never built.*

**H4 — the index sandbox is redundant given `rejectPartiallyStaged`.**
`fsrepository.go:65` walks the worktree, which is why `sandbox.go` runs
`git checkout-index`. But the wrapper already refuses a scoped file with unstaged
edits, so the two guards may cover the same case twice.

**Counter corrected before running.** This hypothesis first named the address set
as its counter. That set *cannot move*: the scoped file is byte-identical under
both roots, so the same mutants are generated either way and an assertion over
them rebuilds its expectation from its own inputs. What can move is each
address's **verdict**, so the counter is the set of surviving addresses.

*Prediction: with the guard in force, the worktree root and the index sandbox
report the identical set of surviving addresses — zero differing.*

*Falsified if any address survives under one root and not the other.*

*Control that must move — inverted, for the same reason: the case here is a
tracked file the scope does NOT name, dirty in the worktree and never staged,
which decides how thorough the suite is. The control is the same run with that
file clean, where both roots read the same bytes and the surviving sets MUST
agree. If they disagree there, the harness is noisy and nothing it says counts.*

**What would refute all four:** the wrapper's behaviour turning out not to be
reproducible through the published API at all — a verdict that moves under every
variant. That sends this back to the question of what ditto's contract should be,
not to which wrapper file to delete, and the previous breakdown's premise dies
with it.

## Decision rule, fixed in advance

- H1 holds and its control moved → delete `buildIgnorePattern`,
  `normalizeTrackedPaths` and the `git ls-files` walk from the derived path; keep
  them on the fail-open path only.
- H1 holds and its control did NOT move → nothing is concluded; the harness is
  repaired and H1 is re-run.
- H2 holds → bump dharness to ditto v0.3.0, delete `baseline.go` and its test.
- H2 dies → `baseline.go` stays, and the note records what ditto's baseline does
  not cover.
- H3 dies → the wrapper's earlier refusal is worth its cost; that is an argument
  for the pipeline living in ditto, and it is recorded as such rather than acted
  on here.
- H4 holds → the choice between the sandbox and the guard is a decision about
  which is cheaper, not a measurement, and it moves to the section below.
- All four die → stop. The wrapper is not an API gap and no ditto change is
  proposed.

## What must be decided, not measured

No counter moves on these. Naming them as experiments would be enumerating
instead of conjecturing.

- **`ditto.DefaultViruses()` exported.** `defaultOptions.Viruses` is package
  private and `WithViruses` replaces the set, so dharness pins all fourteen in
  `viruses.go` with `TestDefaultVirusesPinsDittoMutatorSet` guarding the drift.
  Exporting it deletes a file and a test. It is an API-surface decision.
- **A non-`*testing.T` entry point, and `ditto staged`.** The env-var bridge
  costs one extra `go test` startup per *run* — 750–950 ms once
  (`test-invocation.md`) — against runs measured in minutes. Performance does not
  decide this and quoting it would be inventing a number. What decides it is
  whether ditto owns the staged-scope pipeline or every consumer rewrites
  `changedbytes.go`.
- **The zero-candidate refusal.** `AnalyzeSource` exists because `Release` can
  `t.Fatal` internally, so a guard after it is unreachable. Whether ditto refuses
  an empty scope itself lands on the denominator question `state-of-the-work.md`
  already holds open — a mutant leaving the denominator changes the public report
  dharness gates on at 0.80. It should not be settled apart from that one.

## Two corrections this note owes

**First.** The breakdown before this note listed "run the suite unmutated before
scoring" as something that *should* move into ditto. It is already written —
`a7259a0`, `internal/laboratory/laboratory.go:82`. The observation was read off
ditto **v0.2.0**, which is what dharness pins; the recommendation was stale, not
the reading.

**Second, and it corrects the first.** That fix is **not in v0.3.0**. The tag is
`a97e04a` (2026-08-14 21:32) and `a7259a0` lands after it on `main`
(`9bf3e90`, 23:09). The first correction was written after grepping ditto's
working tree, which was checked out on `docs/report-screenshot` — not `main`,
and not the tag. `git show <ref>:<path>` per ref is what settled it:

| ref | has `verifyBaseline` |
| --- | --- |
| `v0.3.0` | no |
| `main`, `origin/main` | yes |

v0.1.0, v0.2.0 and v0.3.0 are the three versions the module proxy serves
(`go list -m -versions`), so the red-baseline refusal is written, merged and
**unreleased**. H2 is what settles whether the wrapper's copy can go, and it
cannot be answered by a version bump today.

## Results

Fixture: two files in one package, `d3c1eadb92a338417cce9cc85a7ab2248321efd0f051e4631295e9a4a215d01f`
over the concatenated sources, built in a scratch directory outside every
repository. Baseline confirmed green before anything was measured. Tool: ditto
**v0.3.0** from the module proxy. Probe: one throwaway module selecting a variant
by environment variable. Counter: the set of subtest names ditto emits, one per
mutant, each carrying `path:line:col → Mutator`. One warm-up run discarded.

| Variant | Options | Mutants | Killed | Survived | `beta.go` addresses |
| --- | --- | --- | --- | --- | --- |
| A | `WithChangedRanges{alpha.go}` | 10 | 9 | 1 | 0 |
| B | `WithChangedRanges{alpha.go}` + `IgnoreSourceFiles(^beta\.go$)` | 10 | 9 | 1 | 0 |
| C | neither — the fail-open path | 26 | 23 | 3 | 16 |
| D | `IgnoreSourceFiles(^beta\.go$)` alone | 10 | 9 | 1 | 0 |

Three rounds, order rotated `A B C D` / `B C D A` / `C D A B`. Every variant
produced a byte-identical address set in all three rounds, so the counters are
deterministic and no averaging applies. Wall clock is not gated on and is
reported only: variant C took 23.7 s for its 26 mutants.

### Controls

- **The control that had to move: C against D.** 26 addresses against 10, with
  16 present only in C. `IgnoreSourceFiles` is reached, and the harness can see
  the difference it makes. Ticked off against `C-r1.set` and `D-r1.set`.
- **Determinism.** Each variant's three rounds diffed against one another: no
  difference in any of the four. Ticked off against the twelve `.set` files.
- **Fixture identity.** Hashed before the first measured round, unchanged after
  the last.

### H1 — corroborated

A and B report the **identical set of 10 addresses; zero differ on either side**,
which is what H1 predicted and what would have killed it had one moved. Neither
names `beta.go`: the scope alone drops the file, exactly as `scopedrepository.go:41`
says it does. The control moved, so the agreement is not the option being unreached.

What that costs in dharness today, measured on `79c9df7`: **97 tracked `.go`
files**, so a single staged file makes the wrapper walk `git ls-files` and build
an RE2 alternation enumerating the other **96** — to exclude files
`WithChangedRanges` had already dropped.

### H2 — corroborated

Fixture: one package whose test asserts something false, so the suite is red on
unmutated code; confirmed red before measuring, and a green twin confirmed green.
Three rounds, order rotated across `v0.2.0 / v0.3.0 / main`.

| ditto | mutants | killed | survived | score | exit | says why |
| --- | --- | --- | --- | --- | --- | --- |
| v0.2.0 | 8 | 8 | 0 | **1.00** | 0 | — |
| v0.3.0 | 8 | 8 | 0 | **1.00** | 0 | — |
| `main` | **0** | 0 | 0 | none | 1 | `refusing to score against a red baseline` |

`main` reports no score and exits non-zero, which is what H2 predicted. Identical
in all three rounds.

**The control moved, and moved twice over.** v0.2.0 reports a perfect score for a
run that tested nothing, so the fixture contains the case. v0.3.0 reports the
same 1.00 — which is the second correction above, measured rather than argued:
the fix is not in the tag.

### H3 — corroborated

| path | invocations before refusing | evidence |
| --- | --- | --- |
| ditto `main` | **1** | counter file, 3 rounds |
| dharness's wrapper | **1** | `TestRunRefusesToScoreWhenTheBaselineSuiteFails`, run and passing |

Difference **0**, which is the prediction. Nothing runs the suite twice before
refusing.

What it costs when the suite is green, counted the same way on the green twin:

| ditto | mutants | invocations | extra |
| --- | --- | --- | --- |
| v0.3.0 | 8 | 8 | 0 |
| `main` | 8 | **9** | **+1** |

One extra run per release, not per mutant — `sync.Once` in `laboratory.go:83`.
The wrapper pays the same +1 (`TestBaselineRunsOozesOwnCommandInTheSandboxOozeWillMutate`
asserts exactly two processes on a green baseline: the baseline, then ditto). So
upgrading without deleting `baseline.go` pays **2** where 1 answers the question,
at the 750–950 ms `test-invocation.md` measured for starting the command.

### H4 — REFUTED

The prediction was that both roots agree. They do not.

Fixture: a git repository whose `alpha.go` is the whole scope, and whose
`helper.go` — tracked, never staged — supplies the table the test walks. Weakened
in the worktree only.

| Variant | Root | `helper.go` | mutants | survivors |
| --- | --- | --- | --- | --- |
| A | worktree | weakened, unstaged | 8 | **7** |
| B | index sandbox | as staged | 8 | **0** |
| C, the control | worktree | as staged | 8 | **0** |

**7 of 8 verdicts differ between A and B** — a score of 0.13 against 1.00 for the
identical eight mutants of an identical `alpha.go`. The control agrees with B
exactly, so the harness reports agreement when the bytes agree. Deterministic in
all three rounds.

And the guard does not see it. `rejectPartiallyStaged` runs
`git diff --name-only -- <file>` over the **staged** files only. Demonstrated on
the same fixture with `alpha.go` staged and `helper.go` dirty:

    staged                        : alpha.go
    dirty and unstaged            : helper.go
    what the guard checks         : alpha.go -> '' (clean, so it passes)

The sandbox and the guard cover different cases. The sandbox is load-bearing and
stays.

## Verdicts: 4 of 4

H1 corroborated · H2 corroborated · H3 corroborated · **H4 refuted**.

## Conclusion

Three of four survived, so the wrapper is mostly an API gap — but not entirely,
and H4 is where the line falls. What each verdict licenses:

- **H1** → delete `buildIgnorePattern`, `normalizeTrackedPaths` and the
  `git ls-files` walk from the derived path. On `79c9df7` that is an alternation
  over 96 paths built to exclude what the scope already dropped. They stay on the
  fail-open path, where variant D is the evidence they still do the whole job.
- **H2 + H3** → `baseline.go` is redundant against ditto `main`, at identical
  cost, and keeping both after an upgrade pays a second run per release for
  nothing. Neither can be acted on yet: **the fix is unreleased**, and the first
  step is cutting a ditto version that carries `a7259a0`.
- **H4** → `sandbox.go` stays. It is not a duplicate of `rejectPartiallyStaged`;
  the guard reads staged paths and the sandbox decides which bytes the *suite*
  runs against. Deleting it on the reasoning in the previous breakdown would have
  moved 7 of 8 verdicts on the fixture that tested it.

A hypothesis that survived is corroborated, not proven. H1 earns the deletion it
names on the derived path; H2 and H3 earn a release, not an edit.

## An observation this pass turned up, outside the set

`cmdtestrunner.go` in v0.3.0 already strips `GIT_DIR`, `GIT_INDEX_FILE`,
`GIT_WORK_TREE`, `GIT_OBJECT_DIRECTORY` and `GIT_COMMON_DIR` from the test
command's environment, for the same measured reasons `process.go` does. The
previous breakdown listed that as something to move into ditto; for the test
command it is already there. It is **not** a full overlap: dharness strips them
from its own `git` invocations too, and ditto never shells out to git. Read off
the source, not measured, and it belongs to a question nothing here asked.

## What this does NOT establish

Of H1:

- **One pattern shape.** The exclusion measured was `^beta\.go$`, one file.
  dharness builds an alternation over every tracked non-staged path. Same
  mechanism, but the enumerated form was not the one run.
- **One package.** The fixture is two files in a single package. That
  `WithChangedRanges` also drops files in *other* packages follows from
  `ScopedRepository` filtering the whole repository's file list, and follows by
  reading, not by measurement.
- **Not the fail-open path.** Variant D is the evidence that `IgnoreSourceFiles`
  still does the whole job when no scope is passed — 10 addresses against C's 26.
  H1 licenses removing it from the derived path only.
- **One version.** ditto v0.3.0. Nothing here is bound to `main`.

Of H2 and H3:

- **Measured through a probe, not through the wrapper.** H2 was stated as
  "dharness with `baseline.go` deleted"; what ran was ditto itself under the two
  versions. The wrapper's side of H3 comes from its own passing test, which is a
  counter, but nobody ran `tools/mutationstaged` end to end against ditto `main`.
  Until that happens, "delete `baseline.go`" is licensed by two measurements that
  meet rather than by one that covers both.
- **One `sync.Once` per Laboratory**, so the +1 is per release for a single
  laboratory. Whether a run that builds more than one pays more was not measured.

Of H4:

- **One shape of leak.** A tracked file the scope does not name, weakened in the
  worktree. Untracked files, generated files and a dirty *test* file are the same
  family and none of them was run.
- **It says nothing about cost.** That the sandbox is load-bearing does not make
  `git checkout-index` the cheapest way to be right; no alternative was measured.

And of the set as a whole: whether any of it changes a score anyone acts on is
`backlog.md` entry 8, a different question. Nor whether autoreas-bridge's two
repo-specific pieces survive — `stubEmbeddedFrontend` and its exclusion prefixes
are configuration, and no measurement here touches them.

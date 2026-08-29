# Turning gating on, for this repository

Written before measuring.

## The decision this has to answer

Ditto's own mutation gate does not finish. It timed out at `-timeout=30m` on
2026-08-28, and the repository has grown since: 431 mutants when it last passed,
660 when it timed out, **729 now**. At the rate the green run managed — 2.269 s
per mutant — 729 mutants is 27m34s, and at the timed-out run's rate 29m35s.
Before a single hang.

`Gated()` is off in `ditto_mutation_test.go`. It is the only lever measured at
the right size: this repository has already established a **fixed 750–950 ms per
mutant just to start the test command**, paid once per mutant on the ordinary
path, and the gated path compiles once per file instead.

**The decision:** turn gating on for ditto's own gate, or leave it off and find
the time elsewhere.

## Two questions, two metrics, two thresholds

They are not the same question and neither substitutes for the other.

**Cost** — does gating remove enough work? Measured as **exact integers**:
compilations and test-command invocations per release. Not seconds. This
repository's own contract says wall clock is reported and never gated, and the
counters exist already — `GoBuildRunner.Compilations()`, `.Runs()`,
`GatedLaboratory.Gated()`, `.FellBack()` — and nothing reads them outside their
own package.

**Precision** — does gating change the answers? Measured as a **set difference**
over the mutants both paths consider viable. Threshold 0, and the qualifier is
load-bearing: a mutant one path refuses to build is outside the comparison, not a
disagreement in it. See `what-the-field-already-decided.md`.

A cheaper run that answers differently is not a cheaper run. Cost may not be
traded against precision, so precision is checked first and can veto on its own.

## Hypotheses

**H1 — gating removes most of the per-mutant compilations.** The gated path
compiles once per file, the ordinary path once per mutant, so compilations should
fall towards the file count rather than the mutant count. Prediction: with 729
mutants over 78 source files, **compilations per release fall below 200**.
*Kill line:* 400 or more. Then the saving is under half and the fixed toll
survives, so gating cannot be what makes the gate fit.

**H2 — the gating rate is inside the band this repository already measured.**
26–72% of mutants gated, established per-file earlier.
*Kill line:* below 26% over the whole repository. Then most mutants fall back to
the ordinary path anyway, H1's mechanism does not apply at this scale, and the
band was measured on files that were not representative.

**H3 — the two paths agree, over the mutants both run.** Symmetric difference of
the verdict set is **0**, once mutants that only one path can build are removed.
*Kill line:* any disagreement that is not a non-viable mutant. Then gating buys
speed with different answers and stays off whatever H1 says.

## Controls, all of which must hold

1. **The ungated run must be the expensive one.** If both measure the same
   compilations, the harness is not exercising the gated path — which has
   happened here before: `verboselaboratory` does not forward `TestAll`, so a
   probe run with `-test.v` compared the ordinary path with itself
   (`docs/backlog.md` entry 11).
2. **The instrumented file must build and its suite be green with nothing
   selected**, or a fallback is being measured instead of gating.
3. **A known non-viable mutant must be excluded on both sides**, or H3 is
   comparing a category error.

## Decision rule, fixed here

- H3 fails → gating stays off. No cost number rescues it.
- H3 holds and H1 holds → turn it on, and record the new counters.
- H3 holds and H1 fails → gating is not the lever; the note says so and the time
  has to come from scope or from the mutant count itself.

## Results

### H2 — corroborated

    source files with mutants : 50
    mutants                   : 729
    gated                     : 438 (60.1%)

60.1% is inside the 26–72% band this repository had already measured per file,
and the band now has a whole-repository number rather than a range taken from a
handful of files.

### H1 — survives, and the prediction was wrong in my favour

    compilations ungated     : 729
    compilations with gating : 335
    removed                  : 394 (54.0%)

Predicted **under 200**, kill line 400. Measured **335**: the hypothesis stands
and the prediction does not. The arithmetic says why, and it is not subtle — 291
of the 729 mutants are **not** gated, and each still pays for its own
compilation. Gating removes the compilations of the 438 it can express and
nothing else, so the floor is the ungated remainder, not the file count.

Recording the miss rather than the survival: predicting a 93% saving where the
mechanism can only reach 54% was a modelling error, not noise.

### What this number does NOT decide

Compilations are an exact integer and they say the mechanism works. **They do not
say the gate fits in thirty minutes.** That is a wall-clock question, and this
repository reports wall clock and never gates on it, so the counter cannot answer
it and must not be made to look as if it does.

The two are also not proportional. Gating removes compilations; it removes **no
invocations at all** — a gated mutant still runs the binary once. The fixed
750–950 ms toll this repository measured is the cost of starting `go test`, which
compiles *and* runs, so how much of it survives when only the compile is removed
is unmeasured.

### H3 — corroborated, on a scope with survivors in it

Re-run over `internal/schemata/instrument.go`, chosen because it holds 28
survivors. 3574 s, both paths:

    ordinary: total 78, killed 50, survived 28
    gated   : total 78, killed 50, survived 28

    same 78 mutants on both paths
    same verdict for every one of the 78

**This is the run the first attempt could not be.** The earlier comparison held
ten mutants that every path killed, and a set with no survivor in it cannot show
a disagreement about survival. This one has 28, and they agree.

Gating is turned on.

### H3, first attempt — corroborated on a scope too weak to decide it

Two real packages of ditto, both paths, the same test command:

    mutants, ordinary : 10
    mutants, gated    : 10
    symmetric difference of the mutant set : empty
    verdicts: every mutant PASS on both paths
    ordinary: total 10, killed 10, score 1.00
    gated   : total 10, killed 10, score 1.00

**But all ten were killed on both paths, and a set with no survivor in it cannot
show a disagreement about survival.** That is the weakness of this measurement
and no reading of it should be stronger than that sentence.

The scope was reduced to get here. The first attempt covered
`internal/gomutatedfile`, and at roughly two and a half minutes per verdict the
66 runs came to nearly three hours — the measurement paying the exact cost the
hypothesis exists to remove. It was stopped and narrowed rather than left to run,
and the narrowing is why the sample holds no survivor.

What is known beyond this sample: `disagreement-class.md` measured the one class
where the paths were ever seen to differ, and it is the mutant that does not
compile — which now leaves the score on **both** paths, so the only recorded
disagreement is no longer scored by either.

## Decision

The rule fixed in advance says: H3 holds and H1 holds → turn gating on.

Both held, and **the rule fires on evidence thinner than it assumed.** H3 was
written expecting a scope with survivors in it, and what it got was ten mutants
that every path kills. Recording that the rule fired is not the same as recording
that the question is settled.

What would settle it is one run over a scope with survivors — `instrument.go`
alone had 28 — and that run costs about forty minutes. It is the next
measurement, and it is named here rather than skipped.

## What none of this decided

Whether the gate finishes in thirty minutes. Compilations fall 54%; invocations
do not fall at all. The wall-clock question is still open, and this repository
reports wall clock rather than gating on it, so it needs its own run and its own
sentence — not an inference from an integer that was measuring something else.

## What two runs of the gate measured, and it was not ditto

Turning gating on appeared to kill the gate in 6.44 seconds. It did not.

`ditto_mutation_test.go` hardcoded `make`, and this machine is Windows, where GNU
make answers to `mingw32-make`. `.githooks/pre-commit` has probed three names
since the day the hook was unrunnable for exactly this — so the fact was written
down in this repository and used anyway.

The command therefore failed instantly with no output. Ditto reads a non-zero
exit as a killed mutant and an unmutated one as a red baseline, so it refused in
under three seconds. **A test command that cannot start is not a red baseline**,
and nothing here had ever run.

The second run, with gating turned back off, refused the same way in 2.93s. That
is what settled it: the failure was identical without the change blamed for it.

Two consequences, both mine:

- **`Gated()` was turned off on a false diagnosis** and is turned back on. The
  measurement that justified it — 54% of compilations, and identical verdicts
  over 78 mutants with 28 survivors — was never in question.
- **The first failure was real and is separate.** That run reached the point of
  instrumenting `cmd/ditto/main.go` and reading its suite, which is further than
  the other two got, and its message was different: *fails its own suite*, not
  *the test command fails*. The fallback for it stands.

The gate resolves make the way the hook does now, and runs.

## Does the gate finish? No, and two levers are now spent

Three full runs, all to the same end.

| run | configuration | mutants of 727 | result |
| --- | --- | --- | --- |
| 1 | gating on, `make` resolved | 422 | timed out at 1801.057s |
| 2 | + the three nested-release tests skipped | 360 | timed out — **contaminated**, coverage runs were on the same machine |
| 3 | same as 2, nothing else running | **424** | timed out at 1800.885s |

**424 against 422.** The suite skip bought nothing, and the reason is measurable
rather than mysterious: `-failfast` already stops a killed mutant at its first
failing test, so it never reaches the expensive ones. Cutting the suite from
69.3s to 37.2s — 46% — only helps **survivors**, and they are the minority. A 46%
reduction in the suite produced a 0.5% change in the gate.

That is the non-uniform cost of a mutant, measured on this repository rather than
quoted from the literature, and it refutes the arithmetic that justified the
change: 4.27 s/mutant divided by 1.86 predicted 2.30 and would have fit. The
prediction assumed every mutant pays the whole suite. Most pay a fraction.

The skip is reverted. A change with no measured benefit does not ship, and this
one also removed three tests from a mutant's judgment.

### What is left, and it is not a lever — it is the product

Mutating 727 mutants on every push is the wrong shape. Ditto's own answer to that
is `ditto staged`: mutate what the change touched, and pay for that. It is the
feature this tool exists for, dharness uses it, and `WithChangedRanges` is
measured at 4 mutants where the whole fixture charges 48.

The gate asks the repository-sized question on every push and then complains
about the repository-sized bill.

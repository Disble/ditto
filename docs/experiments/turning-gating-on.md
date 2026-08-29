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

### H3

*(measuring)*

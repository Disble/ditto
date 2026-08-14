# Experiment — does the instrumented file say the same thing

Written before the measurement.

## The research question

**To what extent** does the gated path reach the same verdict as the ordinary
path **for each individual mutant**, and does the instrumented file behave as the
original when no mutant is selected — over the mutants of a package built to
exercise every family `schemata` admits and refuses, at `exp/test-invocation` on
`dfd8ed4`, August 2026?

| Element | This question's |
| --- | --- |
| Interrogative phrase | to what extent do the two paths agree, per mutant |
| Variable, the counters that move | mutants whose verdict differs between paths; files whose unselected run differs from the untouched run |
| Population, unit of analysis | the mutants of one scratch package covering every admitted and refused family |
| Space and time | `exp/test-invocation` on `dfd8ed4`, Go 1.25, August 2026 |

Everything measured so far about the two paths compared **totals**. `killed 2,
survived 4` on both sides says the counts match; it does not say the same mutants
are in the same column. `changed-scope.md` says so in its own limits: *this probe
counts, it does not attribute*.

**FINER** — *Feasible*: one scratch module, two releases, seconds each.
*Interesting*: the gated path cannot be turned on by default until this is
answered, and `backlog.md` entry 7 says exactly that. *Novel*: no per-mutant
comparison exists; the golden compares bytes for one fixture of four mutants.
*Ethical*: scratch modules in `t.TempDir()`, destroyed with the test.
*Relevant*: `ditto.Gated()`'s default, and every verdict it would produce.

**PICOT** — **P** the mutants of the scratch package · **I** the gated path ·
**C** the ordinary path over the same mutants, and the untouched file for the
unselected run · **O** disagreeing mutants and disagreeing files, both exact
integers · **T** one release per path.

## The hypotheses

**H1 — the instrumented file is the original when nothing is selected.** With
`DITTO_MUTANT` unset or zero every gate takes its original arm, so the suite must
reach the same verdict as it does over the untouched file.

*Prediction: the unselected run and the untouched run agree, for every file the
planner admits. Falsified by one file where they differ.*

**H2 — the two paths agree mutant by mutant, not merely in total.** The set of
mutants each path reports as survived is the same set, compared by label rather
than by size.

*Prediction: the two survivor sets are identical. Falsified by one label present
in one and absent from the other.*

**H3 — every disagreement is the ordinary path scoring a mutant that never ran.**
`gated-gain-slow.md` claimed this from a single case and the claim has governed
how the gated path is described since.

*Prediction: each label in the symmetric difference of the two sets is a mutant
that fails to compile on its own. Falsified by one that compiles.*

## What would refute all three

A disagreement neither the baseline nor compilation explains — the gate changing
evaluation order, a site admitted that should not have been, an id numbered so
that two mutants select each other's arm. That outcome sends the question into
`instrument.go` and `gate.go` rather than into the paths around them, and none of
these three reaches there.

If H2 holds, H3 has nothing to range over and is recorded as **not decidable
here** rather than as held: an empty symmetric difference satisfies it
vacuously, and a hypothesis satisfied by having nothing to test is not
corroborated.

## Decision rule, fixed in advance

- H1 and H2 hold → the gated path has the evidence it needs for a default, and
  `backlog.md` entry 7 is answered rather than repeated.
- H1 fails → instrumentation changes the program; the gated path is withdrawn,
  not fixed, until the mechanism is understood.
- H2 fails and H3 holds → the disagreements are the false kill and the gated path
  is right; the population of that defect is what it reaches.
- H2 fails and H3 fails → the question moves into `schemata`, and the gated path
  stays opt-in.

Nothing here belongs inside a hypothesis.

## Controls

1. **The comparison is shown to be able to disagree.** Two identical sets prove
   nothing until one path is deliberately broken and the sets are seen to differ.
   Run with a mutant's verdict inverted; the difference must appear.
2. **The fixture exercises both sides of the admission rule.** It must produce
   mutants the planner gates and mutants it refuses, confirmed by the fallback
   count being neither zero nor everything.
3. **The suite is green before anything is mutated**, so no verdict below is
   inherited from a red baseline — which `changed-scope.md` has just shown is
   invisible in the report.

## Fixture

One scratch Go module in `t.TempDir()`, with a package holding: comparisons on
integers, integer literals in and out of index position, string concatenation,
a switch over literal cases, and a loop with a break. Tests cover some of it and
not all, so both columns are populated.

## Results

### Controls

1. **The comparison was shown to be able to disagree.** The same fixture with one
   test added reported **11 survivors** against **14** without it. Two identical
   sets would otherwise have proven nothing.
2. **Both sides of the admission rule are exercised.** The planner gates **13 of
   19** mutants — neither zero nor everything.
3. **The suite is green before anything is mutated**, confirmed by running it.

### H1 holds

The untouched file and the instrumented file with nothing selected reach the same
verdict: both green. Every gate takes its original arm and the program behaves as
it did.

### H2 is refuted, and its first result was an artefact

**Corrected.** This note first reported H2 holding, on these numbers:

    gated   : total 19, killed 5, survived 14    <- WRONG, the gate never ran
    ordinary: total 19, killed 5, survived 14

The probe launched the release with `-test.v`, `release.go` reads
`testing.Verbose()` and wraps the gated laboratory in `verboselaboratory`, and
that decorator does not forward `TestAll`. Both columns were the ordinary path
compared with itself, which is why they matched. `backlog.md` entry 11.

Re-measured with the gate engaged:

    gated   : total 19, killed 4, survived 15
    ordinary: total 19, killed 5, survived 14

    survived ONLY on the gated path: calc/calc.go → Integer Decrement

**H2 is refuted**, by one label. The paths do not agree mutant by mutant.

### H3 is decidable after all, and is answered elsewhere

Written in advance: *if H2 holds, H3 has nothing to range over.* H2 does not
hold, so the symmetric difference is non-empty and H3 has its case. It is
measured deliberately, with a fixture built for it, in
`disagreement-class.md` — where both it and its H1 hold.

### Verdicts: H1 holds, H2 refuted, H3 answered in a note built for it

## What this does NOT establish

- **Nothing about the class that produced the original disagreement.** The
  fixture contains no `declared and not used` case. Until one is measured,
  agreement is agreement on the cases that agree.
- **H1 compares a verdict, not a suite.** Green against green. Two suites can
  both pass while different tests pass; nothing here would see that.
- **One fixture, 19 mutants.** Not a repository, and not a second shape.
- **The false kills in this fixture are agreed on by both paths.** `Label`'s
  literal cases collide under increment and decrement, so some of the 5 killed
  never ran. Both paths report them the same way, which corroborates
  `changed-scope.md`'s H2 rather than adding to it.

## How to run it again

Both probes are skipped unless asked for by name:

    DITTO_PROBE=1 go test . -run TestInstrumentationFidelity -v
    DITTO_PROBE=1 go test . -run TestChangedScope -v

`TestChangedScope` is behind that switch for a second reason, and it is the one
that matters: its assertions record that a defect is **present**. The day the
baseline is read and the run refuses, it turns red for having been fixed. A
measurement frozen as a regression test fails in the wrong direction, and the
gate is not where it belongs.

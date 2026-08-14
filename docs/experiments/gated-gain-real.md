# Experiment — the gated path on a real package, with the real virus set

Written before the measurement. It exists because the fixture in
`gated-gain.md` was designed by someone expecting a result, and it only ever
carried the five viruses this path can gate.

## What is different here, and why it matters

Two things the synthetic fixture could not show.

**All fourteen viruses run.** Arithmetic, statements, assignments — the nine this
path refuses — are present, and each of their mutants keeps the ordinary path. The
gated fraction on real code is therefore lower than the 10 of 18 measured before,
and that fraction is what the saving is made of.

**The suite does its own work.** The claim made from the synthetic result was that
a suite doing real work keeps that work and the ratio falls towards one. That is a
prediction, and this is where it can die.

## H1 — the two paths agree

Same total, killed, survived, score.

*Falsified by any difference.*

## H2 — the gated fraction is at least half

Of the mutants produced for one real file, at least 50% run from the shared
compilation.

*Falsified below 50%*, which would mean the nine refused viruses and the nesting
rule together leave too little for the gate to be worth its complexity here.

## H3 — the ratio falls below the synthetic fixture's

Under 1.6×, against the 1.88–1.97× measured on a fixture built for it.

*Falsified at 1.6× or above.* If the ratio does not fall, the explanation offered
for the synthetic number — invocation toll dominating a trivial suite — was
wrong, and the earlier note has to be corrected rather than defended.

## Controls, each to be ticked off against its evidence

1. **The counter counts.** The ordinary run's invocations must equal the mutant
   total. A counter that reports the same number whatever happens measures nothing.
2. **The comparison refuses.** The gated path is deliberately made to select
   nothing, and the verdicts must separate. Two numbers agreeing say nothing until
   something shows they could have disagreed — this control was declared and
   skipped once already, and running it is the whole reason the rule exists.
3. **The baseline is green** before anything is attributed to the change.

## Fixture

A throwaway copy of dharness, scoped to one file with `WithChangedRanges`, which
is the mechanism dharness's own wrapper uses. One warm-up discarded, three
measured rounds, the order of the two paths rotated.

## Results

Run 2026-08-13 on a throwaway copy of dharness, scoped to
`internal/jsconfig/jsconfig.go`, all fourteen viruses. Warm-up discarded, three
rounds, the two paths rotated.

| Round | Path | total | killed | survived | invocations | wall |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | ordinary | 135 | 127 | 8 | 135 | 112429 ms |
| 1 | gated | 135 | 127 | 8 | **38** | 40530 ms |
| 2 | gated | 135 | 127 | 8 | **38** | 39815 ms |
| 2 | ordinary | 135 | 127 | 8 | 135 | 107732 ms |
| 3 | ordinary | 135 | 127 | 8 | 135 | 115468 ms |
| 3 | gated | 135 | 127 | 8 | **38** | 42424 ms |

### Controls, each against its evidence

1. **The counter counts.** 135 mutants, 135 invocations on the ordinary path.
2. **The comparison refuses.** With the gated path deliberately made to select
   nothing, the same run reported **33 killed and 102 survived** against the
   ordinary 127 and 8. Run before any result below was looked at.
3. **The baseline is green.** `ok internal/jsconfig 0.188s`.

### H1 held

135 mutants, 127 killed, 8 survived, identical in all six runs, on a real file
with every virus running.

### H2 held, above its line

97 of 135 mutants gated — **71.9%**, against a line of 50%. Higher than the 56%
the synthetic fixture managed, because real code has comparisons and literals
that are not all threshold comparisons with the literal nested inside.

### H3 is dead, and it is dead the wrong way round

Predicted **under 1.6×**, falsified at 1.6× or above. Measured **2.77×, 2.71×,
2.72×** — 112 s to 40 s.

Not only above the line: *above the synthetic fixture's 1.9×*, when the whole
point of the prediction was that a real package would be lower.

**Why the prediction was wrong.** It was made from "real package" and should have
been made from how long the suite takes. `internal/jsconfig`'s suite runs in
**188 ms** — faster than the synthetic fixture's. The mechanism behind the
prediction is not wrong: a suite that does real work keeps that work and dilutes
the ratio. It was simply never exercised, because the package chosen to exercise
it does less work per run than the toy did.

The variable is suite duration against the 750–950 ms invocation toll, not
whether the code is real. Conflating those two is what produced a prediction that
missed by the width of the thing it claimed to test.

### What this corrects

`gated-gain.md` closes by saying the clock does not travel and the counter does.
That stands, and this strengthens it — the clock moved from 1.9× to 2.7× between
two packages while the counter behaved exactly as described in both. What does
not stand is the reading offered alongside it, that a real package would show a
lower ratio. On this one it showed a higher one.

## What this does NOT establish

- **The dilution claim is still untested.** No package measured here has a slow
  suite. Testing it needs one where a single run costs multiples of a second, and
  until that is run, "the ratio falls towards one" is a mechanism with no
  measurement behind it.
- **One file, one package.** 135 mutants scoped with `WithChangedRanges`, which
  is the shape of a staged change, not of a whole-repository run.
- **Nothing about the nine refused viruses beyond their cost being paid.** They
  are 28.1% of the mutants here and they took the ordinary path in full.

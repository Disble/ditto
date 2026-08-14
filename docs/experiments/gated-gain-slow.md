# Experiment — a slow suite, and the primitive the ratio is made of

Written before the measurement.

## Why this is being measured

Two notes now report a ratio: 1.9× on a fixture built for the result, 2.7× on a
real package. A prediction died between them, and it died because the ratio is a
**derived** quantity. It depends on how long the suite takes, and both packages
measured so far have fast suites — the claim that a slow one dilutes the saving
has never been exercised.

So this measures two things instead of one: the ratio on a genuinely slow suite,
and the quantity the ratio is made of.

## The decomposition, measured beforehand

A single invocation is the suite's own work plus a fixed toll for starting it.

| package | wall per invocation | suite reports | derived toll |
| --- | --- | --- | --- |
| `internal/jsconfig` | 980 ms | 243 ms | 737 ms |
| `internal/setup` | 2002 ms | 1155 ms | 847 ms |

The gated path removes the toll for the mutants it takes and keeps the suite.
That is the whole mechanism, and it predicts both numbers rather than either.

## H1 — the two paths agree

Same total, killed, survived.

*Falsified by any difference.*

## H2 — the ratio falls on a slow suite

`internal/setup/writer.go`, 35 mutants, against `internal/jsconfig`'s 2.7×.

Predicted **below 2.0×**, and more precisely **between 1.3× and 1.7×**: with a
gated fraction near 70%, the arithmetic of 2002 ms against 1155 ms lands at
about 1.44×.

*Falsified at 2.0× or above.* That would kill the dilution mechanism outright,
not just this number.

## H3 — the toll is the constant, and the ratio is not

The quantity the ratio is made of: milliseconds saved per invocation removed,
computed inside each round as `(ordinary wall − gated wall) ÷ (invocations
removed)`.

On `internal/jsconfig` that came to 741, 700 and 753 ms. The first experiment of
all measured 750–950 ms for the same thing by a different method.

Predicted **between 600 and 1000 ms** on a package whose suite takes five times
longer.

*Falsified outside that band.* This is the hypothesis worth having: if it holds
across two packages whose ratios differ by 2×, the ratio is explained rather than
merely reported.

## Metrics recorded per round

Mutants, gated, fell back, gated fraction, invocations on each path, wall on each
path, the ratio, wall per mutant on each path, and the toll per removed
invocation. Counters are exact; the wall figures are reported and gated on never.

## Controls, each to be ticked off against its evidence

1. **The counter counts** — ordinary invocations equal the mutant total.
2. **The comparison refuses** — the gated path is deliberately made to select
   nothing and the verdicts must separate.
3. **The baseline is green.**

## Repetitions

Five measured rounds rather than three, order rotated, one warm-up discarded. The
toll is a difference of two large numbers divided by a count, so it carries the
noise of both; three rounds would show a band and not its spread.

## Results

Run 2026-08-13, `internal/setup/writer.go`, 35 mutants, five rounds, order
rotated, warm-up discarded.

| Round | ordinary (k/s, inv, wall) | gated (k/s, inv, wall) | ratio | ord/mut | gat/mut | toll per removed |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | 33/2, 35, 91540 ms | **32/3**, 12, 59803 ms | 1.53 | 2615 ms | 1709 ms | 1380 ms |
| 2 | 33/2, 35, 87750 ms | **32/3**, 12, 57289 ms | 1.53 | 2507 ms | 1637 ms | 1324 ms |
| 3 | 33/2, 35, 85405 ms | **32/3**, 12, 56117 ms | 1.52 | 2440 ms | 1603 ms | 1273 ms |
| 4 | 33/2, 35, 74907 ms | **32/3**, 12, 55303 ms | 1.35 | 2140 ms | 1580 ms | 852 ms |
| 5 | 33/2, 35, 64849 ms | **32/3**, 12, 42036 ms | 1.54 | 1853 ms | 1201 ms | 992 ms |

Controls: baseline green (`ok internal/setup 1.263s`); the counter counted 35 for
35; the comparison refused, reporting 10 killed and 25 survived with the gated
path deliberately selecting nothing, against 33 and 2.

### H1 is dead, and it is the finding

**The two paths disagree.** Ordinary: 33 killed, 2 survived. Gated: 32 killed, 3
survived. In all five rounds, the same one mutant.

The extra survivor is a Comparison Replace on:

    if after, err := os.ReadFile(taken.path); err == nil && bytes.Equal(after, taken.data) {

with `err == nil` replaced by `true`.

**The gated path is right and the shipped tool is wrong.** Written the ordinary
way, that mutation leaves `err` declared and never read, and the file does not
compile:

    internal/setup/writer.go:110:13: declared and not used: err

The build fails, the test command exits non-zero, and ditto reads that as a test
catching the mutation. It is a kill no test earned. Under the gate the original
expression stays in the file — it is the unselected arm — so `err` is still read,
the package compiles, the mutant actually runs, and it survives, which is the
truth about it.

So the equivalence this work has been asserting is false, and false in the
direction nobody looks for: the fast path exposes a defect in the slow one. The
honest score for this file is 32 of 35, not 33.

**What it costs elsewhere.** The golden test asserts that the two paths print the
same bytes. That assertion is only sound while no fixture contains a
non-compiling mutant — on one that does, it would demand the gated path reproduce
a bug. The defect has to be fixed at its source: ditto must tell "the mutant did
not build" apart from "a test failed", instead of scoring the first as the second.

### H2 held

Predicted below 2.0×, and between 1.3× and 1.7×. Measured **1.53, 1.53, 1.52,
1.35, 1.54** — the dilution mechanism finally exercised, on a suite of 1155 ms
against `internal/jsconfig`'s 243 ms, and the ratio fell from 2.7× to 1.5×.

It is a comparison between paths that do not agree, which is worth remembering
before quoting it.

### H3 is dead, and the model was too simple

Predicted 600–1000 ms per removed invocation. Measured **1380, 1324, 1273, 852,
992** — three of five above the band, median 1273.

The quantity is not what it was called. `(ordinary − gated) ÷ removed` attributes
the whole difference to the invocation toll, and the ordinary path also
**recompiles the package for every mutant**, because every mutant changes the
source. The gated path compiles once. So the number contains the toll *and* the
per-mutant compilation, and the second grows with the package: `internal/setup`
is far larger than `internal/jsconfig`, which is exactly where the extra 500 ms
came from.

The primitive is therefore **toll + package compile**, not the toll alone. That
also explains the earlier numbers rather than explaining them away: jsconfig's
700–753 ms is a small package's compile plus the toll, and the first experiment's
750–950 ms was measured on a trivial package where the compile is nearly free.

## What this does NOT establish

- **Nothing about how common the false kill is.** One mutant in 35 here. Whether
  it is rare or everywhere is unmeasured, and it decides how much of every
  mutation score ditto has ever reported is inflated.
- **The corrected model has not been tested.** "Toll plus package compile" is a
  better fit for four measurements; it has made no prediction yet.
- **One file, one package, five rounds.**

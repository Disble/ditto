# Experiment — how ditto could tell a dead mutant from one that never ran

Written before the measurement.

## The defect being fixed

Ditto recognises a killed mutant by the test command exiting non-zero. A mutation
that does not compile makes it exit non-zero too, so it is scored as caught by a
test that never ran. Measured at **7.3%** of mutants over dharness's `internal/`,
from four ordinary causes.

The constraint on any fix is the one this whole line of work is under: it must
not cost a compilation per mutant, because removing exactly that is the point.

## H1 — the exit code already tells them apart

If `go test` returns one code for a package that fails to build and another for a
package whose test fails, ditto can distinguish without reading prose, and
`AGENTS.md`'s rule that a verdict comes from exit codes and never from output
stays intact.

*Prediction: the two codes differ. Falsified if both are the same*, which would
mean the signal is only in the text, and reading it would make ditto depend on
one tool's output format.

Cheapest to test and it decides the shape of everything after it, so it goes
first.

## H2 — building before running costs about what a run costs

If H1 dies, the obvious fix is to build the mutant before running its tests and
report "did not compile" separately. That is a second compilation per mutant.

*Prediction: it makes a run at least 50% slower. Falsified below 20%*, which
would make it affordable after all and end the search.

## H3 — the gated path already cannot produce this false kill for what it gates

The gate keeps the original expression in the file as the unselected arm, so a
variable the mutation stops reading is still read. That is how the one mutant in
`writer.go` surfaced.

*Prediction: of the mutants that fail to build for `declared and not used`, none
survive as build failures once instrumented — the class goes to zero. Falsified
by any remaining.*

This is not a fix for the ordinary path, and saying so matters: it would mean the
defect is repaired only where the gate reaches, which on dharness is 71.9% of
mutants and on the refused viruses is none.

## Controls

1. **Both failure kinds are really produced** — one package with a failing test
   and one with a broken build, each confirmed by hand before any code is read.
2. **The measurement of H3 uses the same probe that produced the 7.3%**, so its
   numbers are comparable rather than merely similar.

## Results

### H1 is dead

Controls first: a package with a deliberately failing test, and the same package
with an unused variable so it cannot build. Both confirmed by hand.

    failing test   : go test exit=1
    broken build   : go build exit=1, go test exit=1

**The same code.** The exit status carries nothing, so the only signal is the
text — and reading `[build failed]` out of the output would tie ditto to one
tool's phrasing, which is the thing `AGENTS.md` forbids in as many words.

### H2's prediction is dead, and the hypothesis was badly formed

Same mutation in both arms, five rounds, order rotated, warm-up discarded.

| Round | test only | build + test | overcost |
| --- | --- | --- | --- |
| 1 | 924 ms | 1084 ms | 17% |
| 2 | 873 ms | 1094 ms | 25% |
| 3 | 889 ms | 1063 ms | 20% |
| 4 | 837 ms | 1060 ms | 27% |
| 5 | 892 ms | 1149 ms | 29% |

Predicted at least 50%; measured a median of **25%**. The prediction is dead.

The kill line was "falsified below 20%", and 25% is neither the prediction nor
below the line. **The hypothesis left a gap between the two**, and a result that
lands in it is neither confirmed nor refuted — which is a defect in how it was
written, not in the measurement. A prediction and its kill line have to meet.

The first attempt at this was worse and is recorded rather than discarded: it
mutated a different line in each arm, so the two were not doing the same work at
all.

**The number that stands:** building before running costs about a quarter of a
run, and buys a verdict that does not depend on any tool's output format.

### What is worth measuring next, and why it may be much cheaper

The extra build is a **subprocess**, so it pays the 750–950 ms start this whole
line of work exists to remove. The same question — does this mutation compile —
can be asked in process with `go/types`, no process at all.

**H4 — an in-process type check classifies a non-viable mutant for a fraction of
a test run.** *Prediction: under 50 ms per mutant. Falsified above 200 ms*, which
would leave the 25% subprocess as the cheapest honest fix.

That prediction and its kill line meet, which the last one did not.

## What this does NOT establish

- **H3 is untested.** Whether the gated path removes the `declared and not used`
  class entirely is still a claim resting on one observed mutant.
- **One package for H2.** The 25% is `internal/jsconfig`'s; a package whose suite
  dominates would show less, and one with a large compile would show more.

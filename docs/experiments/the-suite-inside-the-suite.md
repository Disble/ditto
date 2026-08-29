# The suite inside the suite

Written before measuring.

## What was found

Ditto's gate runs `make test.failfast` once per mutant — 729 times. That suite
takes **79.8 s**, and **seven of its tests are 37 s of it**. They are the tests
that run a whole ditto release inside a test: `TestReleaseGolden`,
`TestRunSaysWhatReleaseSays`, `TestSurvivorsAreDistinguishable`, and the four
that spawn real `go test` processes.

Skipping those seven: **42.8 s. A 46% cut, paid 729 times.**

So the gate spends nearly half its budget running ditto inside ditto, once per
mutant, and the two systematic reviews of cost metrics say nobody models this
because nobody separates a mutant's cost by what it did.

## The question this must not answer carelessly

Removing tests from the command a mutant is judged by **changes what the score
means**. A mutant that only those seven catch becomes a survivor. That is buying
performance with accuracy, which this repository forbids doing blind.

So the question is not "is it faster" — it is measured at 46% and not in doubt.
The question is **what it costs in verdicts**.

## Hypothesis

**H1 — the seven tests kill nothing that the rest of the suite does not.** Over a
real scope, the set of killed mutants is identical with and without them.

*Kill line:* one mutant killed only by the seven. Then they are load-bearing for
the verdict, the 46% is not free, and the answer is to make them cheaper or move
them rather than drop them.

## Control

The scope must contain survivors, or the comparison cannot see a mutant moving
from killed to survived. A scope where everything dies proves nothing — that
mistake was made once already in `turning-gating-on.md` and is not repeated.

## Decision rule, fixed here

- H1 holds → the mutant command excludes them, and the gate keeps the same
  verdicts for half the price.
- H1 fails → they stay, and the time comes from somewhere else. The note records
  which mutants they were load-bearing for, because that is the interesting part.

## Results

*(measuring)*

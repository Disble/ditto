# Experiment — what does forwarding the batch through the verbose decorator change?

Written before the measurement.

## The research question

**To what extent** does making `verboselaboratory` forward `TestAll` change how
many mutants run from one shared compilation under `go test -v`, over the seven
mutants of the golden fixture, at d609a05 and the working tree above it,
2026-08-14?

| Element | This question's |
| --- | --- |
| Interrogative phrase | to what extent |
| Variable, the counter that moves | mutants that ran from a shared compilation, and mutants that kept their own |
| Population, unit of analysis | one mutant. The seven the default virus set produces over `testdata/goldenproject/calc/calc.go` |
| Space and time | throwaway copies of d609a05 and of the working tree, 2026-08-14 |

**FINER** — Feasible: the fixture, the binary and the two run modes already
exist; the measurement is one extra invocation of the same binary with
`-test.v`. · Interesting, the decision that turns on it: whether `ditto.Gated()`
is a real option under the flag CI actually runs, and therefore whether the
consuming harness gets anything at all from a gated release — dharness runs
`go test -v` in `tools/mutationstaged/main.go`. · Novel, and what the tool
already reports: **nothing**. `Gated()` and `FellBack()` are exact counters the
report has never printed, which is the second half of backlog entry 11 and the
reason the first half went unnoticed. · Ethical, the fixture that can be
destroyed: the golden fixture copied to a temporary directory, as the golden test
already does. · Relevant, what acts on the answer: whether the forward ships, and
whether the golden gains a run under `-v`.

**PICOT** — P: the seven mutants of the golden fixture, per run. · I:
`verboselaboratory` forwarding `TestAll` to a delegate that can batch. · **C, the
control**: the same fixture and the same binary run **without** `-v`, where the
batch reaches the gated laboratory today — plus the same two modes before the
forward exists. Four runs, one variable. · **O, the exact counter**: the integers
`Gated` and `FellBack`, and the killed and survived counts. Never wall clock. ·
T: one run per mode, d609a05 for *before* and the working tree for *after*.

## Method

The instrument comes first and stays constant. The report cannot say how many
mutants were gated today, so *before* cannot be measured with the code as it
stands. The counting is therefore added on its own, to a copy of d609a05, with
the forward **not** applied: that change alters what is printed and not which
path runs, so it can be present on both sides of the comparison. Only then is the
forward applied and the four runs compared.

Both copies are throwaway. The fixture is copied to a temporary directory and
mutated there, never where it sits.

The control is the run without `-v`. If it does not gate either, the harness is
broken rather than the decorator, and nothing about the decorator can be
concluded from this fixture.

## Hypotheses, and what kills each one

**H1 — under `-v`, and before the forward, no mutant runs from a shared
compilation.** Predicted: gated 0, fell back 7.
*Falsified if the gated count is above zero before the forward is applied.* That
would mean backlog entry 11's diagnosis is wrong.

**H2 — after the forward, the verbose run gates exactly what the non-verbose run
gates.** Predicted: gated under `-v` equals gated without it, and both are above
zero.
*Falsified if the two numbers differ, or if either is zero.*

**H3 — forwarding the batch moves no verdict.** Predicted: killed and survived
identical across all four runs, and the three survivors keep the same addresses.
*Falsified if any count or any address moves.*

**What would refute all of them:** the verbose run gating a different non-zero
number from the non-verbose one *and* verdicts moving with it — which would mean
the verbose decorator changes what runs rather than what is logged, and the fix
is not a forward at all.

## Decision rule, fixed in advance

- H1, H2 and H3 all corroborated → the forward ships, and the golden gains a run
  under `-v` that asserts the gate count is above zero, so the entry cannot
  reopen silently.
- H1 refuted → no code changes. The entry is rewritten against this evidence
  first, because its diagnosis would be the thing that is wrong.
- H2 refuted → the decorator does more than log, and the forward is not the fix.
  Back to the question with the decorator itself as the subject.
- H3 refuted → the forward is reverted the same day. A mutation tester that
  changes its verdicts to go faster has broken the only thing it is for, and no
  gain buys that back.

## Results

Eight runs of the same fixture binary, four before the forward and four after,
same fixture, same toolchain, one variable. The counting was added first and was
present on both sides, so the instrument is constant across the comparison.

| Run | Gated | Fell back | Killed | Survived |
| --- | --- | --- | --- | --- |
| **before** — gated, no `-v` | 4 of 7 | 3 | 4 | 3 |
| **before** — gated, `-v` | **none of 7** | 7 | 4 | 3 |
| **before** — ungated, no `-v` | no gate line | — | 4 | 3 |
| **before** — ungated, `-v` | no gate line | — | 4 | 3 |
| **after** — gated, no `-v` | 4 of 7 | 3 | 4 | 3 |
| **after** — gated, `-v` | **4 of 7** | 3 | 4 | 3 |
| **after** — ungated, no `-v` | no gate line | — | 4 | 3 |
| **after** — ungated, `-v` | no gate line | — | 4 | 3 |

No wall clock is reported and none was gated on. What moved here is an integer.

### Controls

**The run without `-v`** is the control, and it fired: 4 of 7 gated before the
forward existed. Had it also gated nothing, the fixture or the harness would have
been the fault and nothing about the decorator could have been concluded.

**The ungated runs** are the second control. They print no gate line at all,
which is what says the reporter is wired to the gated laboratory rather than to
the run — a gate count appearing on an ungated run would mean the counter counts
something else.

**The instrument was proved able to refuse.** With the forward removed and
everything else in place, the golden fails with `the gated release gated 0
mutants under -v and 4 without it`. Put back, it passes. The unit that asserts
the counter's zero wording was likewise made to fail on purpose and restored.

### H1 corroborated

Predicted gated 0 under `-v` before the forward; measured **none of 7**, with 7
falling back. Backlog entry 11's diagnosis was right, and this is the first time
the number existed: the counters were there, and nothing printed them.

### H2 corroborated

Predicted the verbose run gating exactly what the non-verbose one gates, both
above zero; measured **4 of 7 against 4 of 7**.

### H3 corroborated

Predicted no verdict moving; measured killed 4 and survived 3 in all eight runs.
The three survivors' addresses were compared directly between the gated `-v` run
and the ungated run and are identical, character for character:

    ┃ calc/calc.go:12:11 → Arithmetic (- → +)
    ┃ calc/calc.go:16:41 → Arithmetic (+ → -)
    ┃ calc/calc.go:8:8 → Comparison (inserts =)

That comparison is only possible because of the addresses added for backlog
entry 1. Before them, "no verdict moved" could be checked at the level of two
totals and no finer.

## Verdicts: 3 of 3

H1 corroborated. H2 corroborated. H3 corroborated. Corroborated, not proven:
three hypotheses surviving one fixture earns the guard that keeps them honest,
which is the run the golden now makes under `-v`.

## Conclusion

The forward ships, per the decision rule fixed before the runs. `ditto.Gated()`
was a no-op under the flag CI runs, and the reason it stayed one for three
measurements is that the two counters that could have said so were never printed.
Both halves of backlog entry 11 are closed by the same evidence.

## What this does NOT establish

- **Nothing about the size of the gain.** This measured whether the gated path
  engages under `-v`, not what it saves. The invocation counts, 135 against 38,
  belong to `docs/experiments/gain-after-the-fixes.md` and were not re-run here.
- **One fixture, seven mutants, one package.** The golden fixture is small and
  built to hold mixed verdicts, not to be representative. That 4 of 7 are gatable
  is a property of that file.
- **Nothing about `-ditto.v`.** `verbose()` is `*dittoVerbose || testing.Verbose()`
  and only the second was exercised. They reach the same branch, which is an
  argument from reading the code and not a measurement.
- **Nothing about the other decorators.** `verboserepository`,
  `verbosetemporarydir` and `verbosetestrunner` were not examined for the same
  class of defect. Each wraps a different interface, and this result says nothing
  about whether any of them drops a capability the way this one dropped a batch.

---
name: falsifiable-measurement
description: "Trigger: measure, benchmark, hypothesis, performance claim, experiment, profiling, is X faster, why is this slow, prove it. Run measurements that are allowed to fail, with a control."
license: Apache-2.0
metadata:
  author: "disble"
  version: "2.1"
---

## Activation Contract

Load before any measurement, benchmark, or performance claim, and before any causal
explanation of why something is slow, broken, or better. Load whenever a stated
result would change a design decision.

## Hard Rules

- Write the numeric prediction AND the result that kills it into
  `docs/experiments/<name>.md` BEFORE running anything. A hypothesis that cannot
  fail is a belief.
- **Make the check fail on purpose before trusting it.** A new check that passes
  has proven nothing until it has been seen to refuse: break what it watches,
  watch it fail, put it back. A green that cannot go red is measuring nothing.
- **A control written in the note is not a control run.** Declaring one reads
  like doing one, and the note is where that substitution happens. Tick each
  control off against its evidence before reporting the result it protects.
- **Confirm the edit landed before measuring its effect.** A scripted
  substitution that silently matched nothing looks exactly like a change that had
  no effect. Grep for what you just wrote.
- Every causal claim carries a kill criterion, including a suspicion offered in
  passing. "Probably the cache" is an excuse wearing an explanation's clothes.
- **Re-read the closing sentence before sending.** Discipline holds through the
  experiment and collapses in the line that says what is next and why: a
  priority, a ranking, a cause. It needs a kill criterion or it goes.
- Run a control that removes the variable under test. Without it, a broken
  harness cannot be told apart from a broken design.
- Gate on exact integer counters. Measure and report wall clock; never gate on it.
- Compare within one interleaved round, discard a warm-up, rotate mode order
  across rounds. Do not average across rounds.
- Never measure against a checkout with work in it. Build a throwaway project.
- Verify the baseline is green before attributing anything to the change.
- Record what the experiment does NOT establish, in the same note.
- When a prediction dies, name it and correct the earlier claim with the evidence.

## Decision Gates

| Situation | Action |
|---|---|
| A step is unexpectedly slow | Time every step separately before theorising about any of them |
| The result looks like the design failing | Re-run with the change removed; suspect the harness first |
| A check passes on the first try | Suspect it is not reached; make it fail before believing it |
| An assertion rebuilds its expectation from its own inputs | It cannot fail — delete it and assert the property directly |
| A prototype's result is about to be carried into shipped code | Re-measure through the real thing; the prototype differs in at least one way that matters |
| The deliverable is output a human reads | Run the binary and read it; a green suite proves only that the suite agrees with itself |
| Two candidate causes | State them as mutually exclusive hypotheses; the experiment must kill at least one |
| The work is long or costly | Run the cheapest decisive experiment first, even if it only kills one option |
| A doc states a safety property | Find the test that refuses without it, or write it; a sentence is not a guard |

## Execution Steps

1. Write the note first: question, each hypothesis, its numeric prediction, its
   kill line, the control, and the fixture.
2. Build the fixture as a throwaway project outside every repository.
3. Confirm the baseline passes, and confirm the harness can report a failure.
4. Run one discarded warm-up, then at least three measured rounds with the mode
   order rotated.
5. Append the raw per-round numbers, including any that contradict you.
6. Append the verdict per hypothesis and the limits section.

## Output Contract

Return, and commit inside the note:

- Every hypothesis with its prediction, kill line, and verdict: held or falsified.
- Raw per-round numbers, with exact counters kept separate from wall clock.
- For **each** control the note declares, the evidence that it ran and what it
  showed. A control with no evidence beside it is a hole, not a tick.
- Any earlier claim this measurement corrects, stated plainly.
- What the result does not establish.

## References

- `references/case-studies.md` — measured cases from these repositories, most of
  which killed a claim the author had already made out loud.

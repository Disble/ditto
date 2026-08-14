---
name: falsifiable-measurement
description: "Trigger: measure, benchmark, hypothesis, falsifiability, Popper, experiment, profiling, is X faster, why is this slow, prove it. Run measurements that are allowed to fail, with a control."
license: Apache-2.0
metadata:
  author: "disble"
  version: "2.2"
---

## Activation Contract

Load before any measurement, benchmark, or performance claim, and before any causal
explanation of why something is slow, broken, or better. Load whenever a stated
result would change a design decision.

This is the scientific method adapted to software, on Popper's premise: a
hypothesis is a tentative answer to a research question, and it earns the name
only if some observation could refute it.

## What programming changes about the method

- **Experiments are cheap and repeatable**, so a single round and a missing
  control have no excuse.
- **The instrument is code you wrote**, so suspect it before suspecting the
  design.
- **Observing changes the system** — runs write files, install packages, spawn
  processes — so everything runs on copies.
- **Results expire.** A dependency bump invalidates one, which is why a result
  names the revision and date it is bound to.

## The research question

Every hypothesis answers one. Write it before conjecturing anything: with nothing
to answer, any statement qualifies as a hypothesis.

| Element | The check |
|---|---|
| Interrogative phrase | begins with *what*, *how*, *to what extent* |
| Variable | the counter that moves is named |
| Population, unit of analysis | which units — mutants, files, requests, commits — and how many |
| Space and time | which revision, which fixture, which date |

**FINER** admits the question: Feasible (the harness exists or is cheap),
Interesting (a decision turns on the answer), Novel (nobody has measured it —
and the tool may already report it, so look before building a proxy), Ethical
(the run can destroy nothing anyone depends on), Relevant (name what acts on the
answer: a gate, a default, a design decision).

**PICOT** names the experiment's parts: **P** the units and how many; **I** the
change under test; **C** those same units without it, which is the control;
**O** the exact integer counter that decides, never wall clock; **T** the
observation window, and the revision and toolchain the result is bound to.

## Hard Rules

### Before measuring

- Write the numeric prediction AND the result that kills it into
  `docs/experiments/<name>.md` BEFORE running anything. A hypothesis that cannot
  fail is a belief. The two must meet: a gap between them admits results that
  neither confirm nor refute.
- A hypothesis is a tentative answer to the question. A statement that answers
  something else is a task, however worth knowing it is.
- A hypothesis and its negation are one hypothesis. The negation is the kill line.
- The set must leave room for an answer that is not on it. Name an outcome no
  hypothesis predicts; if none can be named, the set enumerates instead of
  conjecturing.
- What will be done with each outcome is a decision rule. Fix it in advance, in
  its own section, never inside a hypothesis.
- Every causal claim carries a kill criterion, including a suspicion offered in
  passing. "Probably the cache" is an excuse wearing an explanation's clothes.

### While measuring

- **Make the check fail on purpose before trusting it.** A new check that passes
  has proven nothing until it has been seen to refuse: break what it watches,
  watch it fail, put it back. A green that cannot go red is measuring nothing.
- **A control written in the note is not a control run.** Declaring one reads
  like doing one, and the note is where that substitution happens. Tick each
  control off against its evidence before reporting the result it protects.
- **Confirm the edit landed before measuring its effect.** A scripted
  substitution that silently matched nothing looks exactly like a change that had
  no effect. Grep for what you just wrote.
- Run a control that removes the variable under test. Without it, a broken
  harness cannot be told apart from a broken design.
- Gate on exact integer counters. Measure and report wall clock; never gate on it.
- Compare within one interleaved round, discard a warm-up, rotate mode order
  across rounds. Do not average across rounds.
- Never measure against a checkout with work in it — neither the fixture nor the
  tool. Both run from throwaway copies, and fixture identity is confirmed before
  anything is measured.
- Verify the baseline is green before attributing anything to the change.
- A hypothesis formed after seeing the data belongs to a new question, with its
  own fixture.
- While the set is incomplete, nothing is concluded, changed, or committed as
  settled.

### Before concluding

- Count the verdicts. There is no conclusion until every hypothesis has one; one
  that became undecidable gets a recorded verdict, never a silent skip.
- A refuted hypothesis says nothing about the rest of the set.
- A hypothesis that survived is corroborated, not proven. It earns the next
  experiment, not a line of code.
- Every hypothesis refuted is a result. Report it and return to the question.
- Record what the experiment does NOT establish, in the same note.
- When a prediction dies, name it and correct the earlier claim with the evidence.
- **Re-read the closing sentence before sending.** Discipline holds through the
  experiment and collapses in the line that says what is next and why: a
  priority, a ranking, a cause. It needs a kill criterion or it goes.

## Decision Gates

| Situation | Action |
|---|---|
| A step is unexpectedly slow | Time every step separately before theorising about any of them |
| The result looks like the design failing | Re-run with the change removed; suspect the harness first |
| A check passes on the first try | Suspect it is not reached; make it fail before believing it |
| An assertion rebuilds its expectation from its own inputs | It cannot fail — delete it and assert the property directly |
| Two candidate causes | Write each as a claim that could be wrong even if the other is. One that is the negation of the other is not a second hypothesis |
| One hypothesis just died | The others are unaffected. Measure them |
| A result suggests a new hypothesis | It belongs to a new question, not to this set |
| Every hypothesis was refuted | That is the result. Report it and return to the question |
| A prototype's result is about to be carried into shipped code | Re-measure through the real thing; the prototype differs in at least one way that matters |
| The deliverable is output a human reads | Run the binary and read it; a green suite proves only that the suite agrees with itself |
| The work is long or costly | Run the cheapest decisive experiment first — that orders which one runs, it never licenses a conclusion from the first |
| A doc states a safety property | Find the test that refuses without it, or write it; a sentence is not a guard |

## Execution Steps

1. Write the research question and admit it against the four elements, FINER and
   PICOT.
2. Write the note: every hypothesis with its prediction and kill line, the
   outcome that would refute all of them, the decision rule per outcome, the
   control, and the fixture.
3. Build the fixture and the tool as throwaway projects outside every repository.
4. Confirm the baseline passes, and confirm the harness can report a failure.
5. Run one discarded warm-up, then at least three measured rounds with the mode
   order rotated.
6. Append the raw per-round numbers, including any that contradict you.
7. Repeat 3–6 until every hypothesis has a verdict. Only then append the
   conclusion and the limits section.

## Output Contract

Return, and commit inside the note:

- The research question, with its four elements visible.
- Every hypothesis with its prediction, kill line, and verdict: refuted or
  corroborated. The verdict count equals the hypothesis count.
- Raw per-round numbers, with exact counters kept separate from wall clock.
- For **each** control the note declares, the evidence that it ran and what it
  showed. A control with no evidence beside it is a hole, not a tick.
- Any earlier claim this measurement corrects, stated plainly.
- What the result does not establish.

## References

- `references/case-studies.md` — numbered cases from these repositories, most of
  which killed a claim the author had already made out loud.
- `assets/experiment-note.md` — template to copy into `docs/experiments/`.

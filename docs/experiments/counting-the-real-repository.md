# Counting the real repository

Written before the counter exists, so the predictions can be wrong.

## Where it came from

The mutation gate went from 977.968s (green, 2026-08-15) to the full
`-timeout=30m` (red, 2026-08-28). Comparing the two logs by hand:

| | green | red |
| --- | --- | --- |
| mutants | 431 | 617 completed, more pending |
| seconds per mutant | 2.269 | 2.435 |

Cost per mutant moved 7%. The mutant **count** moved 43%, and every one of the
new mutants belongs to a file that did not exist in August: `internal/staged`
(133), `cmd/ditto/main.go` (59), `staged.go` (21), `internal/filecopy` (9). No
old file grew.

**Nothing recorded that.** `perf/baseline.json` holds eight exact counters and
every one is measured against the synthetic fixture in `internal/perfbench` —
six files, 48 mutants, 49 test-command invocations. The fixture did not change,
so the ratchet stayed green while the repository it is meant to protect grew by
half.

`internal/perfbench/doc.go` already names this counter in prose: *"how many
mutants a scope produces — these are integers, identical on every machine, and a
change in one is always meaningful."* It is the one number in that sentence
nothing measures.

## The question

Is "mutants per release on this repository" an exact counter, in the sense this
project means — a machine-independent integer whose every movement is
meaningful? Or is it wall clock in disguise?

## Hypotheses

**H1 — it is deterministic.** Two runs of the count on the same tree produce the
same integer.
*Kill line:* they differ by one. Then it is not a counter and cannot ratchet.

**H2 — it agrees with what CI actually ran.** The count is at least 617, the
number of mutants the red run finished before its timeout, and the difference is
accounted for by files the run had not reached.
*Kill line:* below 617. Then the count cannot see mutants that really ran, and
recording it would understate the cost it exists to gate.

**H3 — counting is not running.** Producing the number costs **0** test-command
invocations and **0** sandboxes, and writes nothing into the tree.
*Kill line:* either is above zero. A counter that pays the cost it measures is
not a counter, and one that writes to the repository is the defect `AGENTS.md`
forbids.

## Decision rule, fixed here

Record the counter in `perf/baseline.json` only if all three hold. If H1 dies
there is nothing to record. If H2 dies the number is the wrong one. If H3 dies
the mechanism is wrong even when the number is right.

## Results

**H1 — corroborated.** Four runs, four identical integers: **660, 660, 660,
660**. It is a counter.

**H2 — corroborated.** 660 against the 617 the red run finished. The 43 it never
reached are the files the timeout cut it off before: `viruses/*`, which the green
run of 2026-08-15 did reach. The count sees everything CI ran and 43 more.

**H3 — corroborated.** `countingLaboratory` answers every mutant without running
anything, so producing 660 costs **0 test-command invocations** and **0
sandboxes**. The test walks the tree before and after and compares the entry
count, so a run that wrote into the repository would fail rather than pass
quietly.

All three hold, so the decision rule says record it. `perf/baseline.json` now
carries `mutantsPerReleaseOnThisRepository: 660`.

**The ratchet was watched biting, in both directions**, because a gate that has
never failed is not known to work:

    baseline 659 -> REGRESSION mutantsPerReleaseOnThisRepository: 660, baseline 659 (+1)
    baseline 661 -> IMPROVED   mutantsPerReleaseOnThisRepository: 660, baseline 661 (-1)

## What it would have caught

431 on 2026-08-15, 660 on 2026-08-28. **+229, and not one of them from a file
that already existed** — `internal/staged` (133), `cmd/ditto/main.go` (59),
`staged.go` (21), `internal/filecopy` (9). The seven largest files of the green
run carry the identical count today: `instrument.go` 78, `gatedlaboratory` 38,
`gate.go` 27, `gomutatedfile` 27, `gosourcefile` 23, `gobuildrunner` 22,
`fstemporarydir` 20.

The eight recorded counters stayed green through all of it, correctly: the
fixture they measure did not change. That is the whole finding. **A ratchet that
only watches the fixture reports on the fixture**, and the cost that took the
gate past its timeout was never in it.

## What it does not catch

The count is one of two axes. The red run also spent **300 seconds inside a
single mutant** whose suite hung until its own timeout — a sixth of the budget,
invisible to any count of mutants. That one needs its own number, and it is not
this one.

## Correction, 2026-08-28

**H1 was tested against the wrong thing.** The four identical runs of 660 above
were all made on a tree at rest. The counter is not noisy — that much held — but
the question that decided its fate was never asked: is it stable under the
mutation the gate itself applies? It is not. Untagged, it read the mutated tree
inside every mutant's sandbox and turned surviving mutants into kills, up to a
tenth of the run.

A determinism check that does not run in the environment the thing will live in
is a check of something else. See `a-counter-that-answers-for-itself.md`.

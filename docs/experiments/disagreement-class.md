# Experiment — the one class where the two paths were seen to disagree

Written before the measurement.

## The research question

**Does** the gated path reach a different verdict from the ordinary path on a
mutant that leaves a local variable unread, and **is that difference the gated
path being right** — over the mutants of a scratch package built to contain that
class, at `exp/test-invocation` on `dfd8ed4`, August 2026?

| Element | This question's |
| --- | --- |
| Interrogative phrase | does the verdict differ, and is the difference the gated path being right |
| Variable, the counters that move | labels in the symmetric difference of the two survivor sets |
| Population, unit of analysis | the mutants of one scratch package carrying an unread local |
| Space and time | `exp/test-invocation` on `dfd8ed4`, Go 1.25, August 2026 |

`instrumentation-fidelity.md` compared the paths mutant by mutant and found them
identical — over a fixture that contained **no** case of this class, because its
integer arrived as a parameter and Go does not refuse an unused parameter. Its
own limits say so. This is the fixture that note should have had.

`gated-gain-slow.md` claimed the disagreement is the gated path being right, from
one observed case, and every description of the gated path since has rested on
that one case.

**FINER** — *Feasible*: one scratch module, two releases. *Interesting*: it is the
only known disagreement between the paths and the whole claim about the gated
path rests on it. *Novel*: never reproduced deliberately. *Ethical*: scratch
module in `t.TempDir()`. *Relevant*: `backlog.md` entries 6 and 7.

**PICOT** — **P** the mutants of the scratch package · **I** the gated path ·
**C** the ordinary path over the same mutants · **O** labels in the symmetric
difference, an exact integer · **T** one release per path.

## The hypotheses

**H1 — the paths disagree, and only about that class.** Replacing `found > 0`
with a constant leaves `found` unread and the file does not compile; the gate
keeps the original expression as its unselected arm, so it does.

*Prediction: the symmetric difference is not empty, and every label in it belongs
to the function carrying the unread local. Falsified if it is empty, or if any
label from elsewhere appears in it.*

**H2 — the gated path is the one that is right.** Where they differ, the ordinary
path reports killed and the gated path reports survived, because the ordinary
path's mutant never ran.

*Prediction: every differing label is survived on the gated path and killed on
the ordinary path, and each one fails to build when written on its own.
Falsified by one that differs the other way, or one that builds.*

**What would refute both:** the paths agreeing here too. That would mean
`gated-gain-slow.md`'s single observation was something else — a fixture
accident, or a difference that has since been changed away — and the description
of the gated path that rests on it has to be withdrawn rather than narrowed.

## Decision rule, fixed in advance

- Both hold → `backlog.md` entry 6 keeps its claim, scoped to this class, and the
  gated path's advantage is exactly this class and no wider.
- H1 holds and H2 fails → the paths differ and the gated path is *not* reliably
  right; the gated path is withdrawn from any default until that is understood.
- H1 fails → the claim about the gated path is withdrawn, not narrowed.

## Controls

1. **The fault is confirmed present.** The mutation is written by hand —
   `found > 0` to `true` — and must fail to build. Confirmed before anything is
   attributed to it.
2. **The comparison is shown to be able to disagree**, already established in
   `instrumentation-fidelity.md` by adding a test and watching survivors move
   from 14 to 11. The same extraction and comparison code is used here.
3. **The suite is green before anything is mutated.**

## Fixture

A scratch module whose package holds a local assigned once and read only inside a
comparison, with no test covering the function, so the surviving mutant has
nowhere else to die.

## Results

### The harness was broken first, and the first answer it gave was wrong

Run once, this note reported the paths agreeing and H1 falsified. That answer was
an artefact.

The probe launched the fixture's release with `-test.v`. `release.go` reads
`testing.Verbose()`, wraps the gated laboratory in `verboselaboratory`, and
**`verboselaboratory` does not forward `TestAll`** — so the batch never reaches
the gate and every mutant takes the ordinary path. The probe was comparing the
ordinary path with itself.

Found by putting a `panic` inside `GatedLaboratory.TestAll` and watching the run
finish without it. A second `panic`, in `GatedLaboratory.Test`, did fire, which
separated *the laboratory is not in the stack* from *the batch is not forwarded*.
Removing `-test.v` made the first panic fire.

That is a defect in ditto, not only in this probe. It is `backlog.md` entry 11.

### Controls

1. **The fault is confirmed present, and generated.** The mutation was written by
   hand — `found > 0 &&` to `true &&` — and had to fail to build. Then, the one
   the first version of this note was missing: **ditto must produce that
   mutation**. It generates 14 mutants of this fixture, of which exactly 1 leaves
   a local unread, and the planner gates it (mutant 7 of 14).

   The first fixture failed this control. Comparison Replace fires only on `&&`
   and `||`, replacing an operand rather than a whole comparison, so a fixture
   whose comparison stood alone could not contain the class at all. A
   hand-written mutation that fails to compile says nothing about the population.
2. **The comparison can disagree**, established in `instrumentation-fidelity.md`
   and using the same extraction code.
3. **The fixture is green before anything is mutated**, and the instrumented file
   builds and is green too, so the file does not fall back for another reason.

### H1 holds

    gated   : total 14, killed 1, survived 13
    ordinary: total 14, killed 2, survived 12

    survived ONLY on the gated path: calc/calc.go → Comparison Replace

The symmetric difference has exactly one label, and it is the mutant that leaves
the local unread. Nothing else differs.

### H2 holds

The differing label survives on the gated path and is killed on the ordinary
path, which is the direction predicted. Control 1 already established that the
mutation fails to build when written on its own, so the ordinary path's kill is a
mutant that never ran.

### Verdicts: 2 of 2

`gated-gain-slow.md`'s claim, made from one observed case, reproduces
deliberately.

## What this does NOT establish

- **One class, one shape.** The gate is right here because it keeps the original
  expression as its unselected arm, so the local stays read. Nothing here says
  what it does for the other four causes of a non-compiling mutant.
- **Nothing about how often this class occurs.** `false-kills.md` counted 24 of
  94 on dharness; that frequency is dharness's.
- **The measurement depends on the gated path being reached at all**, which was
  false for the first run of this probe and is the reason entry 11 exists.

# What the field measures about cost

Two systematic reviews, run independently on 2026-08-29 against the 2021–2026
literature, on one question: **what is a correct performance metric for a
mutation run?**

They were commissioned because this repository had four of them with thresholds
and none had ever changed a decision.

## They converge, and the answer is that there is nothing to build

**1. There is no accepted performance metric for a mutation run.**

The canonical enumeration is Pizzoleto, Ferrari, Offutt, Fernandes & Ribeiro,
*JSS* 2019 — 175 studies, 21 techniques, **18 cost metrics**. Fourteen are counts
or speedup ratios derived from counts. The two most used are "number of mutants
to be executed" (66 studies) and "mutant execution speedup" (36).

**Not one of the eighteen is the absolute cost of a run.** The survey names the
consequence itself: *"measurements vary even for studies that use the same
techniques."*

**2. The mutant count as a cost surrogate is a threat to validity, not a metric.**

Guizzo, Sarro & Harman, ESEC/FSE 2020:

> "Mutation testing research has often used the number of mutants as a surrogate
> measure for the true execution cost of generating and executing mutants. This
> poses a potential threat to the validity of the scientific findings reported in
> the literature. Out of 75 works surveyed in this paper, we found that 54 (72%)
> are vulnerable to this threat."

Measured: the count differs from true execution time by **44% on average, range
19–91%**, and the error **grows** with program size (ρ=0.74) and with the mutant
count itself (ρ=0.76). **37% of strategy rankings flip** when time is measured
instead of counted.

That is `mutantsPerReleaseOnThisRepository`, exactly, on a repository that went
from 431 mutants to 729 — the direction in which the error grows.

**3. Nobody gates a build on the cost of a mutation run.**

Every gate in every tool surveyed is a mutation-*score* gate. The industrial
answer to a run that became too expensive is to move it off the blocking path,
never to fail the build on its cost.

**4. Nothing predicts a run's duration from anything cheaper than running it.**

Predictive Mutation Testing predicts a mutant's *outcome*, not its *cost* — a
different problem nobody has attacked. PIT's author, 2025: *"The ways these
factors will interact can be hard to predict. Although generally larger code
bases with slower tests will have longer analysis times, this is not always the
case."*

## The finding that decides the design

There is **exactly one** published analytical cost model that predicts a run's
duration in advance: Vercammen et al., *STVR* 2022, Formulas 1–4.

**It assumes uniform per-mutant cost. Its own measurements refute the
assumption** — on one of its six subjects, timed-out mutants consume **93% of the
analysis time**.

So the non-uniform cost of a mutant — killed early under `-failfast`, survivor
paying the whole suite, one stopped by a deadline paying the deadline — is
**implemented everywhere and modelled nowhere**. It is an open gap in the field,
not an oversight here.

## What this repository does about it

**Cost is reported, decomposed by outcome class, and never gated.** There is no
metric to build, and building one anyway would be inventing a number the field
has established cannot be had.

The counters stay as **diagnosis without thresholds**: they say where the time
went once you know there is too much of it. See `docs/metrics.md`.

And one thing falls out that was not designed for: **`internal/verdict` is the
instrument the cost question needs.** It was built for accuracy — so a kill
carries why it died — and the reason it records, `Assertion` / `BuildFailed` /
`Deadline`, is exactly the partition the field says nobody models the cost by.
Almost no tool can decompose its own cost that way because almost no tool knows
the reason. This one does.

That is a measurement worth making and it has not been made: the share of a
run's time taken by each class. On the one number available here — a single
deadlocked mutant was 300 of 1800 seconds, 16.7% — and Vercammen's 93% says the
tail can be much worse.

# Experiment — why does ditto's own gate report a perfect score in five seconds?

Written before the measurement.

## What was observed, and is not yet explained

Run 31860409386, `main`, 2026-08-15: **431 mutants, 431 killed, 0 survived, score
1.00 against a minimum of 0.50, in 5.46 seconds.** Twelve milliseconds per
mutant, while the configured test command is `make test.failfast`.

The log carries, once per mutant:

    make[1]: *** [makefile:13: /.hooks.log] Error 128
    laboratory result for 'dittotesting/test.go': Ok[string](…)

`Ok` is how ditto reports that the command failed, and a failed command is how it
recognises a killed mutant.

That is an observation and a suspicion, not a finding. The suspicion has a kill
line below.

## The research question

**To what extent** does the configured test command failing before it runs any
test account for ditto's own mutation gate reporting 431 killed of 431, at `main`
efd8541, 2026-08-15?

| Element | This question's |
| --- | --- |
| Interrogative phrase | to what extent |
| Variable, the counter that moves | exit status of the test command in a mutant's sandbox; mutants reported as survived |
| Population, unit of analysis | one mutant. The 431 the default virus set produces over ditto's own tree |
| Space and time | `main` at efd8541, throwaway copies, 2026-08-15 |

**FINER** — Feasible: the first hypothesis needs one `make` invocation in a
directory, not a mutation run. · Interesting, the decision that turns on it:
whether the makefile changes, and whether the ordinary path gains a guard the
gated path already has. · Novel, and what the tool already reports: the gated
path already refuses this exact situation — `gatedlaboratory` panics with
*refusing to score against a red baseline* — and the ordinary path does not, so
the tool half-reports it already. · Ethical: throwaway copies; nothing is mutated
in a checkout with work in it. · Relevant, what acts on the answer: `makefile`,
`internal/laboratory`, and `perf/baseline.json` if the guard costs a run.

**PICOT** — P: one mutant sandbox for H1, the 431 mutants for H2, one fixture for
H3. · I: the makefile no longer requiring a git directory; then the ordinary path
refusing a red baseline. · **C, the control**: the same command in the same
directory before each change, and — for H1 — the same command run inside a real
checkout, where it is known to work. · **O, the exact counters**: exit status
(integer), and mutants reported survived (integer). Wall clock is reported and
never gated on. · T: efd8541.

## Method

H1 needs no mutation run: a copy of the tree with `.git` removed is exactly what
a mutant's sandbox is, and `make test.failfast` either works there or it does
not. H2 needs the real gate, run through CI, because that is where the 431 came
from. H3 is a unit test with a test command built to fail.

Every hypothesis is measured against the same tree, efd8541, and each change is
applied only after the hypothesis that justifies it has a verdict.

## Hypotheses, and what kills each one

**H1 — `make test.failfast` cannot run where a mutant is tested.** Predicted exit
status in a copy of the tree with no `.git`: non-zero, with zero tests run.
*Falsified if it exits zero there.* The control is the same command in a real
checkout, which must exit zero — if it fails in both, the makefile is not the
variable and the fault is elsewhere.

**H2 — the perfect score is that failure, counted.** Predicted survivors once the
command runs: **more than zero**, over the same 431 mutants.
*Falsified if the gate still reports 431 killed and 0 survived after the command
works.* That outcome would mean the makefile was never the cause and the number
was right all along — which the 5.46 seconds makes hard to believe, and belief is
not the standard here.

**H3 — nothing in the ordinary path refuses a red baseline.** Predicted result of
a release whose test command fails on unmutated code: score 1.00, no failure, no
diagnostic. *Falsified if it already refuses, or already says anything about the
baseline.*

**What would refute all of them:** `make` working in the sandbox, the score
staying at 1.00, and the ordinary path already refusing — which would mean the
5.46 seconds has a cause nobody has looked at, and the whole line of work starts
again from the log.

## Decision rule, fixed in advance

- H1 corroborated → the makefile stops requiring a git directory. H1 refuted →
  the makefile is not touched at all, whatever else is true.
- H2 corroborated → the 431 were false kills, and every earlier claim that this
  gate passed is corrected with the evidence, including the one made out loud
  during this session.
- H2 refuted → nothing about the gate is claimed until the 5.46 seconds is
  explained. A green that cannot be explained is not a green.
- H3 corroborated → the ordinary path gains the guard the gated path has. Its
  cost is one laboratory run per release, which `perf/baseline.json` ratchets in
  both directions, so the new number is written into the baseline deliberately
  rather than absorbed.
- H3 refuted → no guard; find what already refuses and why it did not fire.

## Results

### H1 corroborated

| Where | Command | Exit | Tests run |
| --- | --- | --- | --- |
| copy of the tree with no `.git` | `make test.failfast` | **2** | **0** |
| the same copy, after the change | `make test.failfast` | **0** | **337** |
| **control** — a real checkout | `make test.failfast` | **0** | **337** |

The failure is verbatim the one in the CI log:

    fatal: not a git repository (or any of the parent directories): .git
    mingw32-make: *** [makefile:13: /.hooks.log] Error 128

`git-dir := $(shell git rev-parse --git-dir)` returns empty where there is no
repository, so the target becomes `/.hooks.log` and make dies before reaching a
test. A mutant's sandbox is precisely a copy of the tree with no `.git`.

**The control fired**, which is what makes this mean anything: the same command
in a real checkout exits zero and runs 337 tests, before and after. The variable
is the git directory, not the command.

### H3 corroborated

`TestLaboratoryRefusesARedBaseline` was written before the guard and failed with
`func (assert.PanicTestFunc) should panic` — nothing in the ordinary path
objected to a test command that reports failure on every single call. The gated
path has refused this since `gatedlaboratory` learned to read its own unselected
run; the ordinary path had no such run and no such check.

Two fixtures had to be corrected by the guard, and both were describing a red
suite without meaning to: `observingRunner` in `internal/laboratory` and
`silentRunner` in `internal/perfbench` each answered *the command failed* on
every call, which is what a suite that is red before anything is mutated looks
like. That two of ditto's own fixtures encoded the bad state is the clearest
evidence that nothing was watching for it.

### The cost, measured rather than assumed

The guard runs the suite once per release, and no recorded counter moved when it
was added. That is not a free lunch, it is a blind spot: every
`laboratoryRuns*` counter substitutes `countingLaboratory` for the real one, so
each of them measures how many times `Release` *asks* for a run and none can see
a run the laboratory makes on its own.

`testCommandInvocationsPerReleaseWholeFixture` now measures it through the
shipped laboratory: **49** — the fixture's 48 mutants and one baseline. Recorded
in `perf/baseline.json`, ratcheting in both directions like every other counter
there. A cost nobody records is one that grows unnoticed, which is the mirror of
the unrecorded gain this project already refuses.

### H2 corroborated

Run 31861407819, same tree, same 431 mutants, one variable — the makefile.

| Test command | Total | Killed | Survived | Score |
| --- | --- | --- | --- | --- |
| dies before compiling | 431 | **431** | **0** | 1.00 |
| runs the suite | 431 | **329** | **102** | **0.76** |

Predicted survivors above zero; measured **102**. The total is identical on both
sides, so it is the same population being counted differently: **102 of the 431
kills were mutants no test ever executed**, 23.7% of the reported kills, and the
score moved from a perfect 1.00 to 0.76.

Every earlier claim that this gate passed is wrong, including the one made out
loud during this session, and it is corrected here with the number that corrects
it.

The counter is what settles this. An earlier draft of this note closed on the
run's *duration* — 5.46 seconds against ten minutes — which is wall clock, one
observation per side, and exactly what this project refuses to gate on. That
sentence is withdrawn. It said something true for a reason that could not carry
it.

### What was known before H2 closed, and did not settle it

H2 predicted survivors above zero once the command runs. The gate was dispatched
on this branch with the makefile fixed (run 31861407819) and **was still running
after ten minutes**, against 5.46 seconds before. It is very likely to reach the
30-minute timeout `make test.mutation` sets, because 431 mutants at roughly
twenty seconds of suite each is hours rather than minutes. If it does, H2 is
undecidable from this run and gets that verdict, not a silent skip.

What that duration does settle is narrower than H2 and worth stating on its own:
**5.46 seconds was not the cost of testing 431 mutants.** It is the only number
that has changed, and nothing was touched except the makefile.

Evidence that already exists, and was not collected for this question — which is
why it is reported here rather than counted as H2's verdict:
`gated-by-default.md` ran ditto's own tree over three files earlier the same day
with a working command, `go test` invoked directly, one invocation per mutant
confirmed by a counter. It reported **69 mutants, 58 killed, 11 survived**.
Ditto's own code produces survivors under a command that runs. A gate reporting
0 survivors over 431 mutants and that measurement cannot both be describing the
same repository.

That is a strong indication and it is not a verdict. H2 stays open.

## Verdicts: 3 of 3

H1 corroborated. H2 corroborated. H3 corroborated.

---

# Round 2 — does any of it reproduce?

Written before the second measurement.

Three hypotheses were answered from **one observation per condition**. Every
counter above is meant to be a deterministic function of the tree and the
mutants, so each of them should reproduce exactly, and "should" is not a result.
A number that moves between identical runs is not a measurement of anything, and
this is the round that finds out.

## The research question

**To what extent** do the counters above reproduce across repeated runs of the
same tree with the same commands, at `fix/mutation-baseline` 119da41,
2026-08-15?

| Element | This question's |
| --- | --- |
| Interrogative phrase | to what extent |
| Variable, the counter that moves | killed and survived, per round |
| Population, unit of analysis | one mutant. The 431 of ditto's own tree, and the scoped subset below |
| Space and time | 119da41 for the guarded code, efd8541 for the code before it, 2026-08-15 |

**PICOT** — P: the same mutants each round. · I: the test command that runs. ·
**C, the control**: the same mutants with a command that dies before compiling,
on the revision before the guard, since the guard now refuses that state
outright. · **O, the exact counters**: killed and survived, integers, per round.
Wall clock is not a counter and decides nothing here. · T: 2026-08-15.

## Hypotheses, and what kills each one

**H4 — the full gate reports the same three integers on a second run.** Predicted
431 total, 329 killed, 102 survived, again.
*Falsified if any of the three differs by one.*

**H5 — a command that dies before compiling reports every mutant killed, in every
round.** Predicted survivors 0 and killed equal to total, in each of three rounds
on the scoped subset, on the revision before the guard.
*Falsified if any round reports a survivor.*

**H6 — a command that runs reports the same survivors in every round.** Predicted
identical killed and survived across three rounds on the same scoped subset.
*Falsified if any round differs from another.* This is the one most likely to
die, and the interesting one: a flaky test in ditto's own suite would show up
here as a mutant that changes column between rounds, and nothing so far would
have caught that.

**What would refute all of them:** counters that wander between identical runs,
which would mean every number in this note is an anecdote and the whole file has
to be rewritten against a design that controls for whatever is moving.

## Decision rule, fixed in advance

- H4, H5 and H6 all corroborated → the numbers above stand as measurements and
  the note is complete.
- H4 refuted → the full-gate figures are reported as a range, never as three
  integers, and the cause of the spread becomes the next question.
- H5 refuted → the false-kill mechanism is not what this note says it is, and the
  makefile fix keeps its own verdict but loses its explanation.
- H6 refuted → ditto's own suite has a flaky test, which is a defect of its own
  and outranks everything else here; it gets a backlog entry the same day.

## Results — round 2

### H4 corroborated

Two independent runs of the full gate, dispatched separately, on the same tree.

| Run | Total | Killed | Survived | Score |
| --- | --- | --- | --- | --- |
| 31861407819 | 431 | 329 | 102 | 0.76 |
| 31862250831 | 431 | 329 | 102 | 0.76 |

Three integers, twice, identical.

### H5 and H6 corroborated

Same tree at efd8541 — the revision before the guard, so the guard cannot refuse
condition A and change what is being measured. Same scope, same mutants. The only
variable is the test command. A warm-up was discarded and the order was rotated
between rounds.

| Round | Order | A, command dies | B, command runs |
| --- | --- | --- | --- |
| 1 | A then B | 69 / **69** / **0** | 69 / **58** / **11** |
| 2 | B then A | 69 / **69** / **0** | 69 / **58** / **11** |
| 3 | A then B | 69 / **69** / **0** | 69 / **58** / **11** |

*(total / killed / survived)*

H5 predicted every mutant killed and no survivor under a command that dies:
measured 69 of 69 killed, three times. H6 predicted identical counters under a
command that runs: measured 58 killed and 11 survived, three times. **H6 was the
one most likely to die** — a flaky test in ditto's own suite would appear here as
a mutant changing column between rounds, and none did.

### The control nobody planned

Condition B reproduces **69 / 58 / 11** exactly, and that is the same triple
`gated-by-default.md` measured earlier the same day — a different harness, a
different scope mechanism, a different revision, and a run made before this
question existed. Three rounds back to back test the instrument; a match across
two independent sessions tests something closer to the claim.

## Verdicts: 3 of 3

H4 corroborated. H5 corroborated. H6 corroborated.

## Conclusion

The counters reproduce. Every number in this note is a measurement rather than an
observation, and the headline stands: **102 of 431 reported kills were mutants no
test ever executed**, 23.7%, and the score ditto's own gate printed was 1.00
where it should have been 0.76.

What earned the correction was not the size of the gap. It was that the first
attempt to close this note leaned on wall clock and a single observation per
side, which is the same class of evidence the tool itself refuses when it reads a
failing command as a dead mutant.

## What this does NOT establish

- **Nothing about a flaky test that fires less often than one run in three.** H6
  saw three rounds. It bounds the flakiness it would have caught, not flakiness
  in general.
- **Nothing about the 102 survivors being worth fixing.** They are mutants that
  survived a suite; how many are meaningful and how many are noise — mutating a
  `token.Pos` field changes nothing observable — is a different question.
- **Nothing about other repositories.** The 23.7% is ditto's own tree with ditto's
  own suite.

## What this does NOT establish

- **Nothing about the true score of ditto's own gate.** Whether it is 0.9 or 0.6
  is exactly what H2 is waiting on.
- **Nothing about other consumers.** The makefile defect is ditto's own test
  command. Any project whose command works in a bare directory was never
  affected, and any project whose command needs a git repository has the same
  defect and does not know it — which is an argument for the guard, not evidence
  about anyone else.
- **Nothing about the gated path.** It already refused this, and none of these
  runs exercised it.

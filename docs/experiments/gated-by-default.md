# Experiment — should the gated path be the default?

Written before the measurement.

## Why this one is measurable

`state-of-the-work.md` lists the default under *what is open* and calls it "a
decision nobody has taken". Backlog entry 7 says what would inform it: the gated
path "is proven on two fixtures and a golden, **not on a repository**", and
"what still blocks a default is entry 11, not the verdicts". Entry 11 is closed
(`forwarding-the-batch.md`), so the remaining gap is the one entry 7 names, and
it is an experiment rather than a preference.

What is already measured, and is therefore not re-measured here: the gating rate
on a real **file**, 97 of 135 (`gated-gain-real.md`); the gain, 135 invocations
against 38 (`gain-after-the-fixes.md`); path-against-path verdicts on
**fixtures** (`instrumentation-fidelity.md`, `disagreement-class.md`); and
revision-against-revision on the ordinary path (`backward-compatibility.md`).
None of those is both paths, end to end, over a repository.

## The research question

**To what extent** does turning the gated path on change the verdict of any
mutant, and the number of test-command invocations, over every mutant of two
packages of a real Go repository, at the working tree above d609a05, 2026-08-14?

| Element | This question's |
| --- | --- |
| Interrogative phrase | to what extent |
| Variable, the counter that moves | mutants whose verdict differs between the paths; invocations of the test command |
| Population, unit of analysis | one mutant. Every mutant the default virus set produces over two packages of a throwaway copy of ditto itself |
| Space and time | scratch copy of the working tree, 2026-08-14 |

**FINER** — Feasible: both paths are one option apart and the fixture is a copy
that already exists. · Interesting, the decision that turns on it: whether
`Gated()` ships on by default in the next release, which changes what that
release is. · Novel, and what the tool already reports: the gating counts, now
that entry 11 is closed, and the survivor addresses, now that entry 1 is — the
comparison below is only exact because of them. · Ethical, the fixture that can
be destroyed: a copy of the tree under the scratch directory; the real checkout is
never mutated. · Relevant, what acts on the answer: `defaultOptions.Gated` and
the CHANGELOG entry for it.

**PICOT** — P: every mutant of `internal/schemata` and `internal/gomutatedfile`
in the copy. · I: `ditto.Gated()`. · **C, the control**: the same mutants, same
copy, same command, with the option off. · **O, the exact counter**: mutants
whose reported line differs between the two runs; killed and survived totals; the
gated and fell-back counts. Integers, and the addresses are compared as text. ·
T: one run per path, working tree above d609a05.

## Method

Ditto is pointed at a throwaway copy of its own tree, which is a real repository
with real tests and no work in it. Two packages rather than all of them, because
the ordinary path pays the invocation toll once per mutant and the whole tree
would cost hours to say the same thing.

The two runs differ by one option and nothing else: same copy, same test command,
same virus set, back to back.

The comparison is mutant by mutant, not total by total. Every survivor now
carries `path:line:col` and the text it replaced, so two runs can be diffed as
text — before entry 1 this experiment could only have compared two integers, and
two runs with the same totals and different survivors would have looked identical.

## Hypotheses, and what kills each one

**H1 — the gated path reports the same survivors as the ordinary one.** Predicted
survivors differing between the paths: 0.
*Falsified if one or more survivors appear in one run and not the other.* No
exception, no class, no reading. This hypothesis is allowed to die on any
difference at all.

An earlier draft of this note wrote H1 with the disagreement class of
`disagreement-class.md` attached as an exemption — differences inside it would
not be counted. That is not a hypothesis. Any difference can be argued into a
class after it has been seen, so a prediction of zero with an exemption attached
cannot be refused by any observation, which is the gap between the prediction and
the kill line that this method exists to close. The class is a claim of its own,
and it gets its own line below, decided by a command rather than by a reading.

**H2 — every survivor that differs between the paths is a mutant that does not
compile.** Predicted: differing mutants that compile, 0.
*Falsified if one or more differing mutants compile.* Decided mechanically:
each differing mutant is rebuilt from its address and its change, written into a
copy, and put through `go test -c` — the same instrument that established the
population of 94 in `refusing-false-kills.md`. It exits zero or it does not.
Undecidable if H1 is corroborated, and it gets that verdict recorded rather than
a silent skip.

**H3 — most mutants of real code can be gated.** Predicted: more than half of the
mutants run from a shared compilation.
*Falsified if half or fewer are gated.* Below that the compilation is being paid
for a minority.

**H4 — the gated run costs fewer test-command invocations than the ordinary
one.** Predicted: strictly fewer.
*Falsified if it is equal or greater.*

**What would refute all of them:** survivors differing, those differences
compiling, a low gating rate and no reduction in invocations — the gated path
being riskier and cheaper by too little to matter. That outcome does not leave it
opt-in; it sends it back to being a conjecture.

## Decision rule, fixed in advance

- H1 corroborated, H3 and H4 corroborated → `Gated()` becomes the default in the
  next release, and the CHANGELOG says so as a behaviour change rather than a
  feature. H2 is recorded undecidable.
- H1 refuted **and** H2 corroborated → the differences are all mutants that do
  not compile, which is the class the gated path is right about. The default
  still ships, and the CHANGELOG says which verdicts move and why, because a
  release that silently changes a score people gate on is worse than one that
  changes it loudly.
- H1 refuted **and** H2 refuted → stays opt-in, whatever H3 and H4 said. A path
  that changes a verdict on a mutant that compiles cannot be the one people get
  without asking for it.
- H3 refuted → stays opt-in. A compilation paid for a minority of mutants is an
  option, not a default.
- H4 refuted → stays opt-in, and `gain-after-the-fixes.md` is corrected, because
  it would mean the gain does not survive contact with a repository.

Nothing here is contingent on how long anything took. Wall clock is reported
below if it is reported at all, never gated on.

## Results

Two runs over the same throwaway copy of the working tree, three files scoped
with `WithChangedRanges`, the whole default virus set, one option apart.

| Path | Mutants | Killed | Survived | Test-command invocations | Gated | Wall clock |
| --- | --- | --- | --- | --- | --- | --- |
| ordinary | 69 | 58 | 11 | **69** | — | 116 s |
| gated | 69 | 58 | 11 | **51** | **18 of 69** | 86 s |

Wall clock is reported and nothing is gated on it.

### Controls

**The counter counts.** The ordinary path made exactly 69 invocations for 69
mutants — one each, which is what `laboratory.Test` does, so the instrument is
not inventing or dropping runs.

**The option is the only difference.** Same copy, same binary, same command, same
virus set, back to back. The mutant total is 69 on both sides.

### H1 corroborated

Predicted 0 survivors differing; measured **0 of 11**. The two runs' survivor
lists are identical character for character:

    ┃ internal/gomutatedfile/gomutatedfile.go:108:23 → Integer Decrement (8 → 7)
    ┃ internal/gomutatedfile/gomutatedfile.go:118:45 → Comparison (inserts =)
    ┃ internal/schemata/gate.go:148:22 → Comparison Replace (found != nil → true)
    …

This is the first time the two paths have been compared mutant by mutant on a
repository rather than a fixture, which is what backlog entry 7 asked for. It is
also a comparison that could not have been made a day ago: without the addresses
from entry 1, two runs with 11 survivors each and *different* survivors would
have looked identical.

### H2 undecidable, and recorded rather than skipped

H2 asked what the differing mutants would be. H1 produced none, so there is
nothing to put through `go test -c`. It gets this verdict, not a silent omission.

### H3 refuted

Predicted more than half gated; measured **18 of 69, 26.1%**. Not close to the
line — it is half of it.

This contradicts a number already written down, and the measurement wins.
`state-of-the-work.md` lists under *what is proven*: "**97 of 135 mutants gate on
a real file** — 71.9%". That measurement is sound and its own note is careful:
`gated-gain-real.md` names its limits as "**One file, one package**", and the
file is `internal/jsconfig/jsconfig.go`. What travelled into the index was the
number without the qualifier.

So the honest statement is not 71.9% and not 26%: **the gating rate is a property
of the file, and it ranges from 26% to 72% across the real files measured.**
`Expand` admits comparisons and integer literals and refuses everything else, so
a file whose logic is comparisons gates well and a file of string handling,
slices and statements does not. Nothing here predicts what an unmeasured file
will do, and that is the point.

### H4 corroborated

Predicted strictly fewer invocations; measured **51 against 69**, a reduction of
18 — exactly the 18 mutants that gated, which is the arithmetic the design
implies and a small confirmation that the counter and the gate counts describe
the same run.

## Verdicts: 4 of 4

H1 corroborated. H2 undecidable. H3 refuted. H4 corroborated.

## Conclusion

**`Gated()` stays opt-in**, by the decision rule fixed before the runs: H3
refuted sends it there whatever the others said, and a compilation paid for a
quarter of the mutants is an option rather than a default.

The two hypotheses that survived are worth saying plainly, because they are what
makes it a good option: on a real repository the gated path changed nothing —
same 69 mutants, same 58 kills, same 11 survivors at the same addresses — and it
did cost fewer invocations. Backlog entry 7's "unproven at scale" is narrower now
than it was. It is the *default* that this refuses, not the path.

What would change the answer is not a re-run. It is `Expand` admitting more
shapes, which is what would move the gating rate, and that is a different
question with a different note.

## What this does NOT establish

- **Nothing about the whole tree.** Three files of one repository, 69 mutants.
  The rate on any other file is unmeasured, which is precisely the finding.
- **Nothing about wall clock as a claim.** 116 s against 86 s is one pair of
  numbers on a machine that was also running an editor, not an interleaved
  measurement, and no decision here rests on it.
- **Nothing about the 7.3% false kills.** Both paths scored 58 killed; how many
  of those never compiled was not measured here and belongs to
  `refusing-false-kills.md`.
- **Nothing about `-v`.** These runs were not verbose. That the gated path
  engages under `-v` is `forwarding-the-batch.md`.

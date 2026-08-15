# Experiment — where does a mutant's address come from?

Written before the measurement.

## The research question

**To what extent** does reading a mutant's address off the byte difference
between the original file and the mutated one place that mutant on the line a
reader would call the mutation site, and distinguish it from every other mutant,
over every mutant of two gofmt'd Go files at `exp/test-invocation` HEAD d609a05,
2026-08-14?

| Element | This question's |
| --- | --- |
| Interrogative phrase | to what extent |
| Variable, the counter that moves | mutants whose derived line differs from the AST node's line; mutants sharing an identical address |
| Population, unit of analysis | one mutant. Every mutant the default virus set produces over `testdata/goldenproject/calc/calc.go` and `internal/schemata/gate.go` |
| Space and time | worktree `exp/test-invocation`, HEAD d609a05, sandbox copy, 2026-08-14 |

**FINER** — Feasible: both mechanisms already exist in the tree, the probe is
one test file. · Interesting, the decision that turns on it: which mechanism
ships, and whether `viruses.NewInfection` — public API — has to change. · Novel,
and what the tool already reports: nothing reports a position today,
`Label()` is `path + " → " + infectionName` and carries none; but the tool does
already compute `schemata.Difference`, which is why it is the candidate rather
than a new byte scanner. · Ethical, the fixture that can be destroyed: a copy of
the worktree under the scratch directory; nothing is mutated in place and no
process is spawned — the probe only parses. · Relevant, what acts on the answer:
`GoMutatedFile.Label()`, the console report, and the golden.

**PICOT** — P: every mutant of the two files named above. · I: address derived
from `schemata.Difference(source, mutated).Start`, converted to line:col by
counting newlines in the original. · **C, the control**: the same mutants
addressed as they are today — path plus infection name, no position — plus the
same mutants addressed from `fileSet.Position(node.Pos())` captured where
`gosourcefile.Incubate` already holds both the node and the file set. · **O, the
exact counter**: (a) mutants where mechanism A and mechanism B name a different
line; (b) distinct mutants sharing an identical address; (c) mutants whose
difference span contains a newline. Integers. · T: one run per fixture, HEAD
d609a05, Go toolchain as pinned by the module.

## Method

The tool is a copy of the worktree taken to the scratch directory, not the
worktree itself; identity is confirmed by `git rev-parse HEAD` in the copy
matching d609a05 and by `git status` being clean before the copy. The probe is a
test file that exists only in the copy: it parses each fixture file through
`gosourcefile.Incubate`, and for every infection records the three addresses side
by side in one pass, so the mechanisms see identical mutants in identical order.

The control removes the variable under test twice over. Today's addressing is
run on the same mutants: if it shows no collisions, the fixture is too small to
measure anything and the fixture is wrong, not the design. The node-position
mechanism is run on the same mutants in the same pass, so a disagreement cannot
be a difference of population.

No rounds and no wall clock: every counter here is a deterministic function of
the bytes, so a second round would reproduce the first by construction. That is
also the reason nothing in this note may be gated on timing.

## Hypotheses, and what kills each one

**H1 — on gofmt'd source the two mechanisms name the same line for every
mutant.** Predicted disagreements: 0 of the mutants measured.
*Falsified if one or more mutants get a different line from A than from B.*

**H2 — the pair (address, mutated text) that mechanism A yields is unique per
mutant.** Predicted collisions: 0.
*Falsified if two distinct mutants share both the same address and the same
mutated text.*

**H3 — a mutant's difference span stays inside one line.** Predicted spans
containing a newline: 0.
*Falsified if one or more spans contain a newline.* Named suspicion with its own
kill line: `rangebreak` inserts a statement into a block, so its span should
cross a line boundary; if H3 survives with `rangebreak` present in the
population, the probe is not reaching that virus and the instrument is wrong
before the design is.

**What would refute all of them:** difference spans that are large and start
nowhere near the mutation — which is what a file that is not gofmt'd produces,
since `Mutate` renders through `format.Node` while the original is the file's own
bytes. If that appears on input believed to be gofmt'd, the whole byte-difference
route is the wrong conjecture and the next one is to render the baseline through
`format.Node` too, and address against that.

## Decision rule, fixed in advance

- H1 corroborated and H2 corroborated → ship mechanism A. No virus is touched,
  `viruses.NewInfection` keeps its signature, and the address plus the mutated
  text both come out of the one `Replacement` the tool already computes.
- H1 refuted → ship mechanism B, whatever H2 said. A line that is wrong is worse
  than a line that is coarse, and B is immune to rendering.
- H1 corroborated and H2 refuted → ship A for the address, and report the
  colliding mutants as what they are: duplicates produced by two viruses at one
  site. That is a backlog entry, not a reason to change mechanism.
- H3 refuted → the label carries the span's start only, never a range. This
  changes what is printed, not which mechanism produces it.
- All three refuted → back to the question, with the `format.Node` baseline as
  the next conjecture.

## Results — round 1

Sandbox: `git archive HEAD` of d609a05 extracted to a scratch directory, with no
`.git` inside it, so no command run there can address a real checkout. Confirmed:
the worktree was clean apart from this note, and the extracted tree carries no
`.git` entry.

135 mutants over four files, all gofmt'd.

| Fixture | Mutants | (a) line disagreements | (b) collisions under A | (c) spans crossing a line | Control: labels colliding today | Control: (node, label) colliding |
| --- | --- | --- | --- | --- | --- | --- |
| `testdata/goldenproject/calc/calc.go` | 7 | 0 | 0 | 0 | **7** | 0 |
| `internal/schemata/gate.go` | 27 | 0 | 0 | 0 | **26** | 6 |
| `internal/schemata/instrument.go` | 78 | 8 | 0 | 1 | **75** | 11 |
| `internal/gosourcefile/gosourcefile.go` | 23 | 3 | 2 | 0 | **21** | 6 |
| **Total** | **135** | **11** | **2** | **1** | **129** | **23** |

### Controls

**Today's addressing, run on the same mutants.** 129 of 135 mutants share their
printed label with at least one other mutant — 7 of 7 in the golden fixture, 26
of 27, 75 of 78, 21 of 23. The fixture is not too small to show the effect, and
backlog entry 1's complaint is now a number rather than an impression.

**The node-position mechanism, run on the same mutants in the same pass.** 23 of
135 collide when addressed by node position plus infection name. The populations
cannot differ: `observe` records all three addresses for the same mutant in one
loop, and the run aborts if the two walks disagree on how many mutants exist.

**The instrument was required to move before being believed.** Every counter was
observed off zero on real data during this round — (a) at 8 and 3, (b) at 2, (c)
at 1, both controls at 129 and 23 — so none of them is a green that cannot go
red. The deliberate-skew switch built into the probe was therefore not needed and
was not exercised.

**A first run of this probe reported (c) = 0 with `rangebreak` absent from the
population.** The note had already named that outcome as an instrument failure
rather than a corroboration, so two fixtures carrying `range` and plain loops
were added before anything was concluded. Without that line written in advance,
H3 would have been recorded as surviving on a population that could not test it.

### H1 refuted

Predicted 0 disagreements; measured **11 of 135**. Every one is `rangebreak`, and
the disagreement is systematic rather than noisy: the node is the `for … range`
statement, the byte difference is the `break` inserted as the body's first
statement, so B names the loop's line and A names the line below it. Neither is
noise and neither is obviously wrong — they answer different questions — but H1
claimed they agree, and they do not.

### H2 refuted

Predicted 0 collisions; measured **2 of 135**, both in
`internal/gosourcefile/gosourcefile.go` at 56:5 with mutated text `false`, both
from `comparisonreplace`. The cause is exact: `a || b || c` parses as
`(a || b) || c`, and the virus replaces the left operand of both the outer and
the inner expression. Both replacements start at the same byte and write the same
text; they differ in **how much they replace** — `len(f.ranges) == 0 || node ==
nil` against `len(f.ranges) == 0`. The key H2 was written against looked at the
start and the new text and never at the end of the replaced span.

### H3 refuted

Predicted 0 spans containing a newline; measured **1 of 135**, a `rangebreak`
insertion in `instrument.go` at 106:4 whose replacement carries four lines of the
loop body along with the `break`.

## Verdicts: 3 of 3

H1 refuted. H2 refuted. H3 refuted.

## Conclusion — round 1

Every hypothesis was refuted. That is the result, and it returns the work to the
question rather than to code.

What it does **not** do is send the work down the branch this note's decision rule
named for that outcome. That branch predicted large spans starting nowhere near
the mutation, and prescribed rendering the baseline through `format.Node` as the
next conjecture. The measurement shows the opposite: every span starts exactly at
the mutation, and the largest is four lines. The three deaths have three specific
causes and none of them is rendering, so the prescribed next conjecture answers a
problem that did not occur. The decision rule was written for an outcome that did
not happen, which is the set failing to leave room for the answer that came back.

The mechanism question is therefore still open, and round 2 asks it properly.

---

# Round 2 — what makes a mutant uniquely addressable?

Written before the second measurement, after round 1's three refutations.

## The research question

**What** address distinguishes every mutant from every other and points a reader
at the construct that was mutated, over the same 135 mutants at d609a05?

| Element | This question's |
| --- | --- |
| Interrogative phrase | what |
| Variable, the counter that moves | mutants sharing the tuple the report would print; mutants whose address falls outside the mutated node |
| Population, unit of analysis | one mutant. The same 135 as round 1, same four files |
| Space and time | same sandbox, d609a05, 2026-08-14 |

**PICOT** — P: the same 135 mutants. · I: the tuple the report would actually
print — start `line:col`, the original text, the mutated text — all three read off
the one `schemata.Difference` result. · **C, the control**: round 1's own keys on
the same mutants, already measured: 129 colliding under today's label, 23 under
node position plus name, 2 under start plus mutated text. · **O, the exact
counter**: (d) mutants sharing the printed tuple; (e) mutants whose byte address
falls outside `[node.Pos(), node.End())`. · T: one run, d609a05.

## Hypotheses, and what kills each one

**H4 — the tuple the report prints is unique per mutant.** Predicted collisions:
0 of 135.
*Falsified if two distinct mutants share start line:col, original text and
mutated text.*

**H5 — the byte address never sends a reader outside the construct that was
mutated.** Predicted mutants whose difference start falls outside the offered
node's own span: 0 of 135. This is the question underneath H1's death: if the
`rangebreak` address at line 54 sits inside the range statement that begins on
line 53, then A and B disagreeing is not A being wrong.
*Falsified if one or more mutants have a difference start outside
`[node.Pos(), node.End())`.*

**What would refute both:** collisions that survive the full tuple **and** an
address landing outside its node — which would mean the byte difference is not a
description of the mutation at all, and the address has to be built from the tree
with the viruses reporting their own site.

## Decision rule, fixed in advance

- H4 and H5 both corroborated → ship mechanism A alone. The report prints
  `path:line:col` with `original → mutated`, no virus is touched, and
  `viruses.NewInfection` keeps its signature.
- H4 refuted → the surviving collisions are examined one by one. If they are
  byte-identical mutations at one site they are duplicate mutants, which is a
  finding about the denominator and a backlog entry, not a mechanism failure.
- H5 refuted → the address comes from the node position and the text from the
  difference: both mechanisms ship, and `goinfectedfile` carries the position
  down.
- Both refuted → the viruses report their own site, which is the expensive branch:
  15 call sites and a public API change.

## Results — round 2

Same sandbox, same 135 mutants, same single pass.

| Fixture | Mutants | (d) colliding printed tuples | (e) address outside its node |
| --- | --- | --- | --- |
| `testdata/goldenproject/calc/calc.go` | 7 | 0 | 0 |
| `internal/schemata/gate.go` | 27 | 0 | 0 |
| `internal/schemata/instrument.go` | 78 | 0 | 0 |
| `internal/gosourcefile/gosourcefile.go` | 23 | 0 | 0 |
| **Total** | **135** | **0** | **0** |

### Controls

**Round 1's keys on the same mutants** are the control, and they are what makes
these zeros mean anything: 129 colliding under today's label, 23 under node
position plus infection name, 2 under start plus mutated text, 0 under the full
printed tuple. Same mutants, same pass, four keys.

**Both counters were made to fail before being believed.** They came back zero on
the first run, which the method treats as a reason to suspect the check rather
than the design. Keying (d) on the line alone and shifting (e)'s address by 1000
bytes drove them to 4/20/62/19 and to 7/27/78/23 — every mutant, in (e)'s case.
Reverting both returned every counter to zero. A green that has been seen red.

### H4 corroborated

Predicted 0 collisions; measured 0 of 135. The two mutants that killed H2 are
separated by the original text: `len(f.ranges) == 0 || node == nil → false`
against `len(f.ranges) == 0 → false`, at the same address.

### H5 corroborated

Predicted 0 addresses outside the mutated node; measured 0 of 135. This settles
what H1's death meant. `rangebreak`'s address is one line below the node's, and
that line is the first statement of the loop's own body — inside the construct,
not away from it. A and B disagree because they point at different ends of the
same mutation, not because A is lost.

## Verdicts: 2 of 2

H4 corroborated. H5 corroborated. Both are corroborated, not proven: they earn
the shipped implementation and the re-measurement through it, not a promotion to
fact.

## Conclusion

Ship mechanism A alone, per the decision rule fixed before the round: the address
and both texts come from the one `schemata.Difference` result the tool already
computes. No virus is touched and `viruses.NewInfection` keeps its signature —
the expensive branch, 15 call sites and a public API change, is not taken.

The number that justifies the work is the control, not the mechanism: **129 of
135 mutants cannot be told apart from another mutant by what ditto prints today**.

## What this does NOT establish

- **Nothing about non-gofmt'd source.** Every fixture here is gofmt'd. The
  failure mode named in round 1 — `format.Node` rendering a file that was not
  formatted, so the difference carries layout as well as mutation — was never
  produced, and is therefore untested rather than ruled out.
- **Nothing about the gated path.** These 135 mutants were addressed from the
  bytes `Incubate` and `Mutate` produce. Whether the gated path reports the same
  address for the same mutant is a different question, and it belongs to the
  golden, which compares both paths byte for byte.
- **Mechanism B was measured through a replica.** Nothing carries a node position
  out of `Incubate` today, so the control walked the tree a second time. The
  replica matched mutant for mutant — the run aborts otherwise — but it is a
  replica, and B is not the mechanism being shipped.
- **Nothing about `cancelnil`.** It needs type information, `Incubate` passes
  `nil`, and it is not in the default set; it produced no mutants here.
- **No timing claim at all.** Every counter is a deterministic function of the
  bytes. Nothing here says what computing an address costs, and
  `perf/baseline.json` is the only thing entitled to answer that. It was run
  after the change and did not move: reading a difference adds no parse, no walk,
  no linked file and no laboratory run.

## Re-measured through the shipped code

The counters above came from a probe that walked the tree a second time to
compare mechanisms. `TestSurvivorsAreDistinguishable` asks the default virus set
for mutants exactly as a release does and reads the line the report would print,
over three shapes chosen from what round 1 found — including the nested
disjunction that produced the only collision. It was watched refusing before it
was trusted: dropping the change from the printed line takes it red with
`2 mutants print the identical line "nested disjunction.go:4:9 → Comparison
Replace"`, which is H2's collision reproduced through shipped code.

The golden covers the other half. Both the ordinary and the gated path print the
new addresses, byte for byte identical, over the same fixture.

## What running the binary found that the suite did not

Two things, and the suite was green for both.

The golden's two indistinguishable `Arithmetic` survivors became
`calc/calc.go:12:11` and `calc/calc.go:16:41`. That was the point of the work,
and it is visible only in the output.

The third survivor printed as `Comparison ((nothing) → =)`. Widening `>` to `>=`
inserts one byte and replaces nothing, so one side of the minimal difference is
empty, and an arrow with an empty side reads as a rendering fault rather than as
the mutation. Every unit test passed while it printed that. The operation is now
named — `inserts =`, `deletes =` — which is a change to the deliverable that only
reading the deliverable could have prompted.

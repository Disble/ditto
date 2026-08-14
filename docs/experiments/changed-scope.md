# Experiment — what the changed code can spoil without saying so

Written before the measurement.

## The research question

**What**, in the code this branch changed, can spoil a verdict without anything
reporting it — over the twelve product files `exp/test-invocation` touches on top
of `dfd8ed4`, measured on scratch projects in August 2026?

| Element | This question's |
| --- | --- |
| Interrogative phrase | what can spoil a verdict without reporting it |
| Variable, the counters that move | mutants reported killed against mutants a test actually killed; runs against gated mutants |
| Population, unit of analysis | the 12 product files this branch changed |
| Space and time | `exp/test-invocation` on `dfd8ed4`, Go 1.25, August 2026 |

The scope correction matters and is the reason this note exists. The defect that
started this line of work was found by an integration validation of **the changed
code**, checking that what worked before still works now. One verdict differed.
The population is what changed, not the 1293 mutants of a whole repository —
measuring those answered a different question, and every hypothesis in it died.

**FINER** — *Feasible*: scratch projects in `t.TempDir()`, seconds each.
*Interesting*: the gated path cannot be turned on until this is answered.
*Novel*: none of these paths has been measured; two carry no test at all.
*Ethical*: scratch modules built and destroyed by the test, never a checkout.
*Relevant*: ditto's reported score, and `backlog.md` entry 7.

**PICOT** — **P** the changed files and the mutants they process · **I** the gated
path · **C** the same fixture through the ordinary path, and through the gated
path with the fault removed · **O** mutants reported killed, and runs of the
shared binary, both exact integers · **T** one release per fixture variant.

## The hypotheses

**H1 — the discarded baseline hides a whole file's worth of false kills.**
`gatedlaboratory.go:101` runs the instrumented file with no mutant selected, so
every gate takes its original arm. That is a baseline. `gatedlaboratory.go:119`
throws its verdict away with `_ = unselected`. If the suite is red, the baseline
fails, every selected run fails too, and every gated mutant of that file is
scored killed.

*Prediction: on a fixture whose suite fails, the gated run reports every mutant
killed and a perfect score, with nothing naming the cause. Falsified if any
mutant is reported survived, or if the output names the failing baseline.*

**H2 — the fallback hands the false kill back.** A file whose shared build fails
returns to the delegate, which is the path that scores a non-compiling mutant as
killed. The gated path was credited with disagreeing and being right about
exactly that; the credit may reach only the files it manages to gate.

*Prediction: on a fixture with a mutant that cannot compile, the gated run and
the ordinary run report the same number killed. Falsified if the gated run
reports one more survivor than the ordinary run.*

**H3 — the baseline run is counted as a mutant's run.** `GoBuildRunner.Runs`
increments once per `Test`, and `TestAll` calls `Test` once before selecting
anything. Every ratio published from that counter would be inflated by one per
file.

*Prediction: the laboratory calls its runner exactly once more than the number of
mutants it gates. Falsified if the two are equal.*

**What would refute all three:** a spoiled verdict produced inside the
instrumentation itself — the gate expression, the site admission, the numbering
of ids — rather than in the orchestration around it. All three look at
`gatedlaboratory` and `gobuildrunner`; none of them reaches `schemata`. If the
three hold and the original disagreement is still unexplained, `schemata` is
where the next question goes.

## Decision rule, fixed in advance

- H1 holds → the baseline verdict is read and the run refuses, the way dharness's
  wrapper already refuses to score on a red baseline.
- H2 holds → the gated path is not credited with fixing the false kill, and
  `backlog.md` entry 7 is corrected.
- H3 holds → every ratio published from that counter is recalculated before it is
  quoted again.
- None holds → the changed scope is clean here and the question moves to
  `schemata`.

Nothing here belongs inside a hypothesis.

## Controls

1. **The harness distinguishes.** A fixture whose suite passes must report both
   killed and survived mutants. One that reports all of either is measuring its
   own error.
2. **Each fault is confirmed present before it is blamed.** The red-baseline
   fixture must be seen failing on unmutated code; the non-compiling mutant must
   be seen failing to build.
3. **The ordinary path is run over the same fixtures**, so a gated result is
   compared against something rather than described.

## Fixture

Scratch Go modules written by the probe into `t.TempDir()`, one per variant, each
with a `replace` back to the module root. Nothing is copied from a checkout and
nothing survives the run.

- **A, the control** — a green suite over a gateable comparison.
- **B, for H1** — the same, with one assertion made false.
- **C, for H2** — the same, plus a literal index that a decrement turns negative.

## Results

### Controls

1. **The harness distinguishes.** The green variant reported both outcomes:
   `total 4, killed 1, survived 3`. One that answered all of either would have
   been measuring its own error.
2. **Each fault was confirmed present before it was blamed.** The red variant's
   suite was run unmutated and had to fail; the non-compiling mutation was
   written by hand — `values[0]` to `values[-1]` — and had to fail to build.
   Either one passing stops the probe rather than being explained afterwards.
3. **The ordinary path was run over the same fixture**, so H2's gated number is
   compared rather than described.

### H1 holds, and the size of it is the same four mutants

Same source, same four mutants, one assertion made false:

| Fixture | Total | Killed | Survived | Score |
| --- | --- | --- | --- | --- |
| green suite | 4 | 1 | 3 | 0.25 |
| **red suite** | 4 | **4** | **0** | **1.00** |

A suite that fails before anything is mutated produces a **perfect score**, and
nothing in the output names the cause. The run does measure it — the unselected
run at `gatedlaboratory.go:101` is exactly that baseline — and discards the
answer at line 119.

The information is already paid for. `_ = unselected` is where it is thrown away.

### H2 is refuted, and its first result was an artefact

**Corrected.** This note first reported H2 holding, on these numbers:

    gated   : total 6, killed 2, survived 4      <- WRONG, the gate never ran
    ordinary: total 6, killed 2, survived 4

The probe launched the release with `-test.v`. `release.go` reads
`testing.Verbose()` and wraps the gated laboratory in `verboselaboratory`, which
does not forward `TestAll`, so the batch never reached the gate. Both columns
were the ordinary path. See `backlog.md` entry 11 and
`disagreement-class.md` for how it was found.

Re-measured with the gate actually engaged:

    gated   : total 6, killed 1, survived 5
    ordinary: total 6, killed 2, survived 4

**H2 is refuted.** The gated path does *not* hand the false kill back here. The
decrement that turns `values[0]` into `values[-1]` cannot compile as a literal,
but the gate selects it through a function call, which the compiler cannot fold
to a constant — so it builds, runs, and survives, because nothing tests the
function it lives in.

The credit `gated-gain-slow.md` took is wider than this note first said, not
narrower.

### H3 holds

Measured through the shipped laboratory with its runner seam: **3 runs for 2
gated mutants**. The assertion was then broken on purpose and seen to report
`expected 2, actual 3`, so it can go red.

Every ratio published from that counter is inflated by one run per file.

### Verdicts: 3 of 3

All three hold. Under the skill's rule they are corroborated, not proven: each
earns the next experiment rather than a line of code, and the decision rule fixed
in advance is what says which experiments those are.

## What this does NOT establish

- **Nothing about `schemata`.** All three looked at the orchestration. The
  instrumentation itself — gate expression, site admission, id numbering — is
  untouched by this note, and the outcome named in advance as refuting all three
  was precisely a defect living there.
- **H1's blast radius is the gated path only.** It is opt-in and off by default.
  The ordinary path scores a red suite as a perfect run too, but it never had a
  baseline to discard; what is new here is that the gated path measures one and
  throws it away.
- **Three scratch fixtures, one shape each.** Four to six mutants apiece, not a
  repository.
- **H3 was measured through a seam**, counting calls to the runner rather than
  reading `GoBuildRunner.Runs` from a real release. They increment together by
  construction, which is an argument and not a measurement.

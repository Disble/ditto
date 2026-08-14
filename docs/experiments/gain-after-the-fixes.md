# Experiment — is the gain still there

Written before the measurement.

## The research question

**To what extent** does the gated path still reduce the invocations a release
pays, after the changes of 2026-08-14 — over the 135 mutants of dharness's
`internal/jsconfig/jsconfig.go`, at `exp/test-invocation`, August 2026?

| Element | This question's |
| --- | --- |
| Interrogative phrase | to what extent is the reduction still there |
| Variable, the counter that moves | invocations of the test command, counted from outside ditto |
| Population, unit of analysis | the 135 mutants of one real file |
| Space and time | `exp/test-invocation`, dharness at `aa605f4`, Go 1.25, August 2026 |

`gated-gain-real.md` measured 135 invocations on the ordinary path against 38 on
the gated one, and 2.77×/2.71×/2.72× on the clock, on 2026-08-13. Since then the
branch gained a refusal on a red baseline, and a defect was found that turns the
gated path off under `-v`.

**The old measurement did not suffer that defect** — it reported 38 invocations
where a disengaged gate would have reported 135, which is the same evidence a
panic would have given. What it has not had is a re-run after the changes.

**FINER** — *Feasible*: six releases over one file, minutes each. *Interesting*:
performance is the metric this project is governed by, and the number quoted for
it is a day old and predates two changes. *Novel*: nothing has re-measured it.
*Ethical*: a copy of the repository, destroyed with the test. *Relevant*: whether
the gated path is worth turning on at all.

**PICOT** — **P** the 135 mutants · **I** the gated path at HEAD · **C** the
ordinary path at HEAD, same file, same rounds · **O** invocations of the test
command, an exact integer, with wall clock reported beside it and gating nothing ·
**T** one warm-up discarded and three measured rounds, the two paths rotated.

## The hypotheses

**H1 — the invocation count is unchanged.** Nothing in the changes touches how
many mutants gate: the refusal happens before selection and adds no run.

*Prediction: 135 invocations ordinary and 38 gated, the same numbers as
2026-08-13. Falsified by any other count on either path.*

**H2 — the verdicts are unchanged.** The same file, the same viruses.

*Prediction: 135 mutants, 127 killed, 8 survived, on both paths and every round —
the numbers `gated-gain-real.md` recorded. Falsified by one that differs.*

**What would refute both:** the release refusing to run at all, because
`internal/jsconfig`'s suite is red in the copy. That is not a gain result, it is
a fixture result, and it would say the fixture moved rather than the gain.

## Decision rule, fixed in advance

- Both hold → the published gain stands and can be quoted after the changes.
- H1 fails → the number in `gated-gain-real.md` is corrected in place, with the
  new count, before it is quoted anywhere again.
- H2 fails → this stops being a gain question and becomes a compatibility one,
  which `backward-compatibility.md` is the note for.

## Controls

1. **The counter counts.** The ordinary path must report exactly one invocation
   per mutant — 135 for 135. A counter that reports anything else is measuring
   itself.
2. **The gate is engaged.** A gated run reporting 135 invocations is a run with
   the gate off, which is what `backlog.md` entry 11 describes. The count is the
   evidence.
3. **The baseline is green** in the copy before anything is mutated, or every
   verdict below is a red-baseline artefact — which the branch now refuses
   outright on the gated path but not on the ordinary one.

## Fixture

A copy of dharness, scoped to `internal/jsconfig/jsconfig.go` with
`WithChangedRanges`, the way the wrapper scopes it. The package gains a
`TestMain` that appends one line per run to a file, which is how invocations are
counted from outside ditto rather than read out of its own bookkeeping.

## Results

One warm-up discarded, three rounds, the two paths rotated.

| Round | Path | total | killed | survived | counter | wall |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | ordinary | 135 | 127 | 8 | 132 | 113.0 s |
| 1 | gated | 135 | 127 | 8 | 135 | 42.5 s |
| 2 | gated | 135 | 127 | 8 | 135 | 42.6 s |
| 2 | ordinary | 135 | 127 | 8 | 132 | 111.8 s |
| 3 | ordinary | 135 | 127 | 8 | 132 | 111.9 s |
| 3 | gated | 135 | 127 | 8 | 135 | 42.7 s |

### Control 1 failed, so H1 is not decidable with this instrument

The ordinary path must report one invocation per mutant — 135 for 135. It
reported **132**, in every round.

The shortfall is not noise and is not a mystery: `false-kills.md` counted
**3 of 135** mutants of `internal/jsconfig/jsconfig.go` that do not build. A
mutant that does not compile never reaches the test binary, so the counter never
sees it. 135 − 3 = 132, from a number measured independently and a day earlier.

That also says what this counter is: it counts **executions of the package's test
binary**, because it is a `TestMain` inside that package. `gated-gain-real.md`'s
38 counted **invocations of the test command**, which the gated path removes and
this instrument cannot see — on the gated path the binary is run directly, so
every gated mutant still increments it.

The two numbers measure different quantities. 135 against 38 is not a
disagreement, it is a different question answered.

**H1 is recorded as not decidable here**, not as falsified. An instrument that
fails its own control has not refuted anything.

### H2 holds

135 mutants, 127 killed, 8 survived — on both paths, in all six runs, and
identical to what `gated-gain-real.md` recorded the day before. The changes moved
no verdict on this file.

### The clock, reported and gating nothing

    round 1: 113.0 s → 42.5 s   2.66×
    round 2: 111.8 s → 42.6 s   2.62×
    round 3: 111.9 s → 42.7 s   2.62×

Against **2.77× / 2.71× / 2.72×** the day before, on the same file and the same
machine. Same band, slightly lower, and wall clock on a working machine varies by
more than that difference — which is exactly why it gates nothing here.

### Verdicts, first run: H2 holds, H1 not decidable

## Second run, with an instrument that counts the right thing

The fix is not a better `TestMain`, it is a different place to count. The test
**command** is now a program of its own that appends a line and then runs
`go test`, so counting invocations means counting invocations. The old counter is
kept beside it, because the difference between the two is exactly what the gated
path removes.

One warm-up discarded, three rounds, the two paths rotated.

| Round | Path | total | killed | survived | invocations | executions | wall |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | ordinary | 135 | 127 | 8 | **135** | 132 | 112.5 s |
| 1 | gated | 135 | 127 | 8 | **38** | 135 | 41.8 s |
| 2 | gated | 135 | 127 | 8 | **38** | 135 | 42.0 s |
| 2 | ordinary | 135 | 127 | 8 | **135** | 132 | 113.5 s |
| 3 | ordinary | 135 | 127 | 8 | **135** | 132 | 113.7 s |
| 3 | gated | 135 | 127 | 8 | **38** | 135 | 42.3 s |

### Control 1 passes

The ordinary path reports **135 invocations for 135 mutants**, in every round.
One command per mutant, which is what the control asked for and what the first
instrument could not produce.

### Control 2 passes — the gate is engaged

38 against 135. A gated run with the gate off reports 135, which is what
`backlog.md` entry 11 describes; this one does not.

### H1 holds

*Predicted 135 ordinary and 38 gated, the same numbers as 2026-08-13. Falsified
by any other count on either path.*

    ordinary 135, gated 38, in all three rounds

Identical to the published measurement, from an instrument built independently of
it a day later. **The gain is intact after the changes.**

### H2 holds

135 mutants, 127 killed, 8 survived — both paths, all six rounds, both runs of
this experiment, and the same as `gated-gain-real.md`.

### The two counters together, which is what the first run could not see

    invocations of the test command  : ordinary 135, gated  38
    executions of the test binary    : ordinary 132, gated 135

The gated path invokes the command 38 times — the mutants it could not gate —
and runs the binary 135 times, once per mutant plus the unselected baseline
minus the ones it never reaches. The ordinary path invokes the command 135 times
and reaches the binary 132, because **3 of jsconfig's 135 mutants do not
compile**, a count `false-kills.md` made independently.

Every number in that pair is accounted for by a number measured somewhere else.

### The clock, reported and gating nothing

    112.5 s → 41.8 s   2.69×
    113.5 s → 42.0 s   2.70×
    113.7 s → 42.3 s   2.69×

Against 2.77× / 2.71× / 2.72× the day before.

### Verdicts: 2 of 2, both hold

## What this corrects

The first run of this note reported H1 as not decidable and said the instrument
had to be fixed "before quoting 38 again". That reads as though the published 38
were in doubt. **It was not.** `gated-gain-real.md`'s control passed — 135
invocations for 135 mutants — and this run reproduces its 38 exactly. The
instrument that failed was the one built here, and its control caught it.

## What this does NOT establish

- **One file.** `internal/jsconfig`'s suite runs in 188 ms, which
  `gated-gain-real.md` already identified as why its ratio is high: the variable
  is suite duration against the invocation toll, not whether the code is real.
- **The clock is not evidence of a gain size.** It is reported because it is what
  a person feels, and it agrees with the counter's direction. It is not what any
  decision here is allowed to rest on.

# Experiment — how many kills no test earned

Written before the measurement.

## Why this is being measured

`gated-gain-slow.md` found one mutant that ditto scores as killed because the
mutated file does not compile: `err == nil` replaced by `true` leaves `err`
declared and never read. The build fails, the test command exits non-zero, and a
failing command is how ditto recognises a mutant a test caught.

One case says nothing about the size of the problem. Every such mutant inflates
the mutation score of every run ditto has ever reported, and the score is the
number people act on.

## H — a measurable share of mutants do not compile

Each mutation is written on its own and the package is built.

*Prediction: at least 1% of mutants fail to build. Falsified below 1%*, which
would make `writer.go` an oddity rather than a systematic inflation.

## Controls, each to be ticked off against its evidence

1. **The harness distinguishes.** It must report mutants that build as well as
   mutants that do not. A probe that says everything fails, or nothing does, is
   measuring its own error.
2. **The mutant count matches ditto's.** `internal/setup/writer.go` was measured
   at 35 mutants by a real release. The probe must produce 35 for that file.
3. **The untouched package builds** before anything is attributed to a mutation.

## Metrics

Per package and in total: mutants, how many failed to build, the share, and the
distinct compiler messages behind them — because one cause repeated a hundred
times and a hundred causes are different problems.

## Fixture

A throwaway copy of dharness. Every mutation is written, built, and undone before
the next.

## Results

Run 2026-08-13 over `internal/` of a throwaway copy of dharness, all fourteen
viruses, each mutation written on its own and the package built.

**1293 mutants, 94 of which do not build — 7.3%.**

### Controls, each against its evidence

1. **The harness distinguishes.** 1199 built, 94 did not.
2. **The mutant count matches ditto's.** `internal/setup/writer.go` produced 35,
   the same number a real release reported for that file.
3. **The untouched tree builds**, and its identity was checked rather than
   assumed: `module github.com/Disble/dharness`.

The identity check is in the list because the first run of this experiment was
made against a copy of **ditto** — a `cd` left before the `tar` — and reported
87.3%. Control 2 caught it, answering 0 mutants where 35 were expected, and the
first explanation reached for was a plausible one about environment variables
rather than a look at what had actually been copied. A control that fires and
gets explained is worse than no control: it spends the signal and leaves the
feeling of having answered it.

### H held, seven times over

Predicted at least 1%, falsified below. Measured **7.3%**.

### Four causes, all systematic

| Count | Cause |
| --- | --- |
| 42 | `index -1 must not be negative` — an integer literal used as an index, decremented |
| ~22 | `declared and not used: x` — a comparison replaced by a constant, leaving its variable unread |
| ~20 | `operator - not defined on ... string` — arithmetic turning `+` into `-` over concatenation |
| 6 | `duplicate case` — two switch cases colliding after a literal changes |
| 1 | an import left unused |

Not exotica. Four shapes of ordinary Go, and the heaviest files are the ordinary
ones: 14 of 220 in `internal/report/human.go`, 12 of 68 in
`internal/project/detect.go`, 10 of 57 in `internal/setup/files.go`, 10 of 46 in
`internal/tool/command.go`.

### What it means

7.3% of the mutants ditto reports as killed were never executed. The score people
read, and the gate dharness fails builds on, are both inflated by that much on
this repository.

## What this does NOT establish

- **This is a lower bound.** It builds the package. A mutation that compiles as a
  package but breaks the *test* build would be a false kill this probe misses.
- **One repository.** The four causes are shapes of Go rather than of dharness,
  but their frequency is dharness's.
- **Nothing about how much of it the gated path fixes.** The `declared and not
  used` class it certainly does — the original expression stays in the file as
  the unselected arm, which is how the one mutant in `writer.go` was found. The
  others are untested, and guessing at them here would be the same mistake this
  note opens by describing.

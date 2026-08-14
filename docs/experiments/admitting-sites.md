# Experiment — how a site earns the gated path without paying for a compilation

Written before the measurement.

## Why this is being measured

Measured already: a mutation that does not compile fails the one shared build and
takes every other mutant in the run with it. So a site has to be known to compile
before it is admitted.

The obvious reading of that — compile each mutation to check it — costs one
compilation per mutant, which is the entire thing this work exists to remove. The
choice is between three ways of paying, and it is not an intuition call:

| | cost | what it loses |
| --- | --- | --- |
| **A** compile each mutation to admit it | N compilations | everything; this is today's cost |
| **B** compile the instrumented file, bisect on failure | up to log N extra | nothing, at some complexity |
| **C** compile the instrumented file once, and on failure the whole file keeps the old path | 1 wasted compilation per bad file | the fast path for that file |

C is the cheapest and the simplest. It is only acceptable if files that fail are
rare, and that is a number, not a feeling.

## What already fails, and why

Three of ninety-one integer-literal sites refused to compile under a call gate:

- `const wrapWidth = 70` — a constant declaration
- `const declaredSideIndent = 15` — the same
- `float64(ms)/1000` — the untyped `1000` has to become a `float64`

The first two are visible in the syntax tree. The third is not: it needs to know
what type the expression around it wants, which is type information this package
does not have and, being stdlib-only, would pay a lot to get.

## H1 — a syntactic rule catches the constant ones and only those

Refuse an integer literal that sits inside a `const` declaration or an array
length.

*Prediction: this removes exactly 2 of the 3 known failures — both constant
declarations, and not the float. Falsified if it removes fewer than 2, and
falsified differently if it removes all 3, which would mean the float case was
misdiagnosed.*

## H2 — with that rule, failing files are rare enough for C

Instrument every file of a real package with the actual `internal/schemata` code,
compile each one, and count the files whose instrumented form fails.

*Prediction: fewer than 20% of files with at least one gate fail to compile.
Falsified at 20% or above, which would make C wasteful and put B back on the
table.*

## Control

Files with no gates at all must be returned byte-identical and must still
compile. If instrumenting a file that has nothing to instrument changes it, every
number below is measuring the harness.

## Fixture

A throwaway copy of dharness, instrumented through the real package rather than a
prototype of it. Every prototype used before this one differed from the shipped
code in at least one way that mattered.

## Results

Run 2026-08-13 over a throwaway copy of dharness, through `internal/schemata`
itself rather than a prototype of it. Comparison, comparison-invert,
comparison-replace, integer-increment and integer-decrement — the five viruses
whose mutations this package can gate.

    mutants gated 925, refused 265
    files with at least one gate 42, of which failed to compile 2

**Control passed:** every file with nothing to gate came back byte-identical.

### H1 held, and the measurement added a case rather than removing one

Predicted: the syntactic rule removes exactly 2 of the 3 known failures — both
constant declarations, and not the float.

Both `const` declarations in `internal/report/human.go` are gone from the
failures. What is left there is exactly the case predicted to survive:

    human.go:181: invalid operation: float64(ms) / dittoInteger16()
                  (mismatched types float64 and int)

The second failing file is the same mechanism wearing a different type:

    linescope.go:145: cannot use dittoInteger42() (value of type int)
                      as int64 value in argument to ...Add

An untyped constant adopts whatever type its context wants — `float64` in one,
`int64` in the other — and a function returning `int` adopts nothing. Two
instances of one diagnosis, and no instance of any other.

### H2 held, and it decides the design

Predicted: fewer than 20% of files with at least one gate fail to compile,
falsified at 20% or above. **Measured 2 of 42, 4.8%.**

So **C**: instrument the file, compile it once, and if that compilation fails the
whole file keeps the path ditto has always taken. The wasted cost is one
compilation for 2 files in 42, against N compilations for every file under A, and
the bisection of B buys back 4.8% of files at a complexity nobody has to carry.

777 of 1190 mutants — 77.7% — are admitted to the gated path by the rules as they
stand.

## What this does NOT establish

- **Compiling is not behaving**, again. This says the instrumented files build.
  Verdict equivalence is proven for comparisons and integer literals on one file
  each, in `schemata-feasibility.md` and `gating-non-bool.md`, and not for these
  42.
- **One repository.** dharness has a house style, and `int64` counters and
  `float64` arithmetic are part of it. A repository that uses more untyped
  constants in typed contexts would fail more files, and the rule stays correct
  while the 4.8% does not travel.
- **Five viruses, not fourteen.** The other nine produce mutations this package
  refuses by design, and refusing costs the per-mutant path, not correctness.

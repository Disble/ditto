# Experiment — what a test invocation costs, and what removing it buys

Written before the measurement. Predictions and kill criteria are stated here
first so the numbers cannot be read charitably afterwards.

## The question this decides

The second wave has one target: stop paying `go test`'s startup for every
mutant. Three ideas reach it — schemata, skipping mutants no test covers, and a
precompiled binary per mutant — and all three break on the same wall. Ditto only
knows a command *string* (`WithTestCommand("make test")`) and runs it at
`internal/cmdtestrunner/cmdtestrunner.go:27`. It cannot pass `-c`, cannot
control the build, cannot ask for coverage.

So the contract question is: **does ditto learn to build the tests, not only run
them?** That is a delicate change, and it is not decided by argument. It is
decided by whether removing the invocation pays enough to be worth it.

## Modes

All three run on a throwaway project. Never on this repository, never on the
repository that consumes ditto, never on a checkout with work in it.

| Mode | What it does per mutant |
| --- | --- |
| **A** — control, ditto today | write the mutated source, run `go test` |
| **B** — compile per mutant | write the mutated source, `go test -c`, then run the binary |
| **C** — compile once (schemata's shape) | instrument once with a runtime switch, `go test -c` **once**, then only run the binary with one mutant selected |

## The exact counter this moves

Wall clock is measured and reported, never gated — a development machine is
always busy. The integer that this experiment moves, and the only one that would
enter `perf/baseline.json`, is **toolchain invocations per release**:

| Mode | toolchain invocations | test-binary starts |
| --- | --- | --- |
| A | 12 | 12 |
| B | 12 | 12 |
| C | **1** | 12 |

A and B share a counter and still differ in cost, because `go test` does more per
invocation than `go test -c`. That difference is exactly the toll under
measurement, so here the ratio is the evidence and the counter is the contract.

## Predictions, and what kills each one

From first-wave measurements: `go test` 1100 ms on a trivial suite and 1716 ms on
a real one; `go test -c` 334 ms hot; a precompiled binary 36 ms trivial and
979 ms real.

**P1 — compiling separately is already cheaper.** B/A ≈ 0.34 on the light suite.
*Falsified if B/A > 0.70.*

**P2 — compiling once is a different order of magnitude.** C/A ≤ 0.10 on the
light suite (334 ms + 12 × 36 ms against 12 × 1100 ms). *Falsified if C/A > 0.25.*

**P3 — the toll is per invocation, not per compilation.** The absolute saving
A − B per mutant is roughly the same whether the suite is light or heavy,
because it is startup, not work. *Falsified if the per-mutant toll on the heavy
suite differs from the light suite by more than 2×.*

P3 is the one that matters for design. If it holds, the thing to remove is
**invocations**, and only C removes them. If it fails, the whole model behind
the second wave is wrong and the plan gets rewritten before any code is.

## Decision rule, fixed in advance

- **C/A > 0.25** → schemata does not pay for its correctness risk. The contract
  stays as it is and the second wave is closed. This is the outcome that kills
  the whole line of work, and it is a real possibility, not a formality.
- **B alone brings a 12-mutant loop under 2 s** → take B. It needs the same
  contract change but carries no semantic risk, and the cheaper change wins.
- **Otherwise** → only C justifies opening the contract, and the shape of the
  option follows from what C actually needed to work.

## Control, because the machine is never idle

A is measured inside every round, interleaved with B and C on the same fixture,
same machine, same minute. One warm-up round is discarded, three rounds are
measured, and the mode order rotates each round. Ratios are computed **within** a
round; cross-round averages are not reported, because they would carry the load
drift the ratio exists to cancel.

## What this experiment does NOT answer

C measures the **ceiling** of schemata, not schemata. The instrumented build is
not the same compiled code as A's, so this says what the technique could buy — it
says nothing about whether the un-mutated baseline still behaves identically once
every mutable site sits behind a switch. That correctness question is separate,
it is the expensive part, and it must not be smuggled in under a green ratio.

## Results

Measured 2026-08-13 on a throwaway module outside every repository, Go 1.26.5,
windows/amd64. Twelve mutants: three comparison sites × four operators. One
warm-up round discarded, three rounds measured, mode order rotated.

**Control passed first.** Both suites: A and C reached identical verdicts on all
twelve mutants (`KKKKKKKKKKKK`), and the un-mutated instrumented binary stayed
green. Without that, the numbers below would describe two different jobs.

Light suite — the suite itself costs nothing, so the whole run is overhead:

| Round | A | B | C | B/A | C/A | toll removed per mutant |
| --- | --- | --- | --- | --- | --- | --- |
| A B C | 11428 ms | 15531 ms | 1323 ms | 1.36 | 0.116 | 842 ms |
| B C A | 10626 ms | 13922 ms | 1337 ms | 1.31 | 0.126 | 774 ms |
| C A B | 10333 ms | 13795 ms | 429 ms | 1.34 | 0.042 | 825 ms |

Heavy suite — the same package with ~920 ms of real work per run:

| Round | A | B | C | B/A | C/A | toll removed per mutant |
| --- | --- | --- | --- | --- | --- | --- |
| A B C | 21669 ms | 26159 ms | 12652 ms | 1.21 | 0.584 | 751 ms |
| B C A | 23303 ms | 27122 ms | 12744 ms | 1.16 | 0.547 | 879 ms |
| C A B | 22793 ms | 27190 ms | 11477 ms | 1.19 | 0.504 | 943 ms |

### P1 is dead

Predicted B/A ≈ 0.34, falsified above 0.70. **Measured 1.16 to 1.36 in all six
rounds.** Compiling with `go test -c` and then starting the binary is *worse*
than plain `go test`, by about 300 ms per mutant, consistently, on both suites.

The first-wave number that suggested otherwise — `go test -c` at 334 ms against
`go test` at 1100 ms — compared two things that are not alternatives. `go test`
compiles *and* runs; `-c` compiles, links to disk, and then a separate process
load follows. The 334 ms was never the whole job.

**A precompiled binary per mutant is off the table.** That is one of the three
second-wave ideas removed without writing a line of ditto.

### P2 survives on the suite it was stated for

Predicted C/A ≤ 0.10 on the light suite, falsified above 0.25. **Measured 0.042,
0.116 and 0.126** — inside the kill line, at the edge of the prediction. Twelve
mutants in 0.4–1.3 s against 10–11 s.

The heavy suite reads 0.50–0.58, and that is not a failure: P2 was stated for the
light suite, and the heavy ratio is arithmetic. Twelve runs of a 920 ms suite is
11.0 s that no technique can remove. C measured 11.5–12.7 s, so **C is running
within about one compilation of the floor.**

### P3 survives, and it is the one that decides the design

The toll removed per mutant is 774–842 ms on a suite that does nothing, and
751–943 ms on a suite doing 920 ms of work. **Same toll, and it does not care
what the suite costs.** It is startup, not compilation.

So the thing to remove is *invocations*. Only C removes them, and it removes
them by compiling once — which ditto cannot do while it only knows a command
string.

### The decision rule fires

- C/A stayed under 0.25 on the light suite, so the second wave is not closed.
- B was to be preferred if it landed the loop under 2 s. B is slower than doing
  nothing, so it is not preferred; it is discarded.
- Only C justifies opening the contract, and what C needed is now concrete:
  compile the instrumented package **once**, then start that one binary per
  mutant with the site selected from the environment.

### The honest brake on all of this

A real ditto run today is ~15 s for twelve mutants: ~10.6 s of test command and
~4.5 s of ditto's own startup. C removes almost all of the first number and none
of the second. The projection is therefore **15 s to ~5.8 s, about 2.6×** — not
the 30× the ratio in isolation suggests. After schemata, ditto's own 4.5 s is the
majority of the run, and it becomes the next target rather than a rounding error.

And the caveat stated before the run still stands untouched: this measured the
ceiling of schemata, not schemata. One trivial fixture kept its verdicts and its
green baseline under instrumentation. That is a first data point, not evidence
that a real package survives having every mutable site wrapped in a function
call — which, note, is itself a change to the compiled code.

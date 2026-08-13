# Experiment — where ditto's own ~4.5 s goes

Written before the measurement, same discipline as `test-invocation.md`.

## Why this is being measured before the contract is opened

The invocation experiment showed schemata can take the test-command slice of a
run from ~10.6 s to ~1.3 s. It also showed that leaves ditto's own ~4.5 s
untouched, so the real projection is 15 s → ~5.8 s, and after that the 4.5 s is
the majority of the run.

That 4.5 s has never been measured. It was **derived** — total minus test
commands — so it is a residue, not an observation, and nobody knows what is in
it. Opening ditto's contract for schemata while the larger remaining slice is
unexamined would be choosing the expensive, risky half first.

## What is actually inside that residue

A ditto run is not a program someone launches. It is a Go test:
`go test -tags=mutation -run TestMutation`. So the residue contains at least

1. `go test` compiling and linking the mutation test binary — which links ditto
   itself — and starting it;
2. the repository walk and the read of every source file;
3. parsing (4 parses per release) and AST walks (12 per release with 3 viruses);
4. building one sandbox, at a measured ~0.45 ms per linked file;
5. per mutant: writing the mutation into the sandbox, restoring it afterwards,
   and spawning the test process;
6. reporting, including a rendered diff per surviving mutant.

Items 1–4 happen once. Item 5 happens N times. They need very different fixes,
so the first question is not "what is slow" but **"does it scale with N?"**

## Method — no instrumentation

The test command is replaced with a binary that exits 0 immediately. Whatever
time the run still takes is, by construction, everything that is not the test
command: ditto's own work plus N process spawns. Nothing has to be instrumented,
so nothing about ditto's timing is changed by measuring it.

Then N is varied by generating fixtures with different numbers of mutable sites,
and the mutant count is read from ditto's own report rather than assumed.

Same rules as before: throwaway project, never a repository with work in it;
warm-up discarded; wall clock reported, never gated; comparisons made within a
round.

## Predictions, and what kills each one

**H1 — the residue is mostly fixed cost, not per-mutant work.**
overhead(N=48) ≤ 1.5 × overhead(N=4).
*Falsified if the ratio exceeds 1.5*, which would mean the residue scales and the
target is the per-mutant path, not startup.

**H2 — most of the residue is outside ditto's own code.** Specifically, `go test`
compiling and launching the mutation binary is at least half of it. Measured by
running the same mutation test twice: once through `go test`, once as a binary
precompiled with `go test -c` beforehand.
*Falsified if the difference is under 25% of the residue.*

Note this is not the idea that died in the invocation experiment. There, a
compile was paid **per mutant** and lost. Here it is paid **once per run**, which
is a different arithmetic, and a TDD loop that keeps a warm binary would not pay
it at all.

**H3 — ditto's per-mutant work is cheap.** (overhead(48) − overhead(4)) / 44 ≤
50 ms per mutant. *Falsified above 150 ms*, which would make writing and
restoring the mutation, or spawning the process, a target in its own right.

## What each outcome means for the contract question

- **H1 holds and H2 holds** → the 4.5 s is mostly the outer `go test`, it is paid
  once, and a warm loop avoids most of it. Schemata's 2.6× projection was
  pessimistic and the contract change is worth more than it looked.
- **H1 holds and H2 dies** → the fixed cost is inside ditto, and it is a cheaper,
  safer target than schemata. Do that first.
- **H1 dies** → the residue scales with mutants, so it grows exactly when
  schemata makes runs bigger. Then the per-mutant path is the target and
  schemata alone would not deliver what it promises.

## Results

Measured 2026-08-13, Go 1.26.5, windows/amd64, on a throwaway module that
consumes ditto through a `replace` directive. Each mutable site yields exactly
two mutants, so the count is set, not estimated. Warm-up discarded, three rounds
per size.

| Mutants | residue via `go test` | residue as a precompiled binary | `go test` slice | per mutant |
| --- | --- | --- | --- | --- |
| 4 | 1268 / 1263 / 1579 ms | 355 / 102 / 96 ms | 913 / 1161 / 1483 ms | — |
| 12 | 1518 / 1860 / 1419 ms | 222 / 184 / 144 ms | 1296 / 1676 / 1275 ms | — |
| 48 | 1677 / 1573 / 1555 ms | 434 / 418 / 384 ms | 1243 / 1155 / 1171 ms | — |

### H1 holds — the residue is fixed cost

Predicted overhead(48) ≤ 1.5 × overhead(4), falsified above 1.5. **Measured
1.24** on medians (1573 ms against 1268 ms). Twelve times the mutants cost about
a quarter more time.

### H3 holds, with room to spare

Predicted ≤ 50 ms per mutant, falsified above 150 ms. **Measured 6.9 ms** from the
`go test` path and 7.2 ms from the precompiled path — two independent slopes that
agree. A Windows process spawn is most of that, so ditto's own per-mutant work
(write the mutation, run, restore) is close to unmeasurable.

### H2 holds, and it is the finding

Predicted the `go test` slice is at least 25% of the residue. **Measured 73–75%
at 48 mutants**, and 72–94% at 4. Running the identical mutation from a binary
compiled beforehand costs **96–434 ms**; running it through `go test` costs
**1263–1677 ms**.

So ditto's machinery — walk the repository, parse, mutate, build one sandbox,
spawn N processes — is roughly **100 ms fixed plus 7 ms per mutant**. Everything
else in the residue is `go test` compiling and starting the binary that ditto
happens to live in.

### The two experiments describe one enemy

The invocation experiment found a fixed 750–950 ms toll for starting `go test`,
paid **once per mutant**. This one finds ~1200 ms of the same thing, paid **once
per run**. It is the same cost in two places, and it explains both numbers.

That also decides which of the two is worth attacking first: the per-run instance
is removed by compiling the mutation binary once and keeping it, which carries no
semantic risk at all. Note this is *not* the idea that died — that one paid a
compile per mutant and lost. Paid once and reused, the same compile wins.

### What this does NOT show

The ~4.5 s that motivated this experiment was measured on the **dharness wrapper**
running against its own fixture, not on this module. What is measured here is
that **ditto's machinery is not what is expensive**, so if 4.5 s is real over
there, it is coming from somewhere else in that pipeline.

The obvious candidate is unmeasured and stays that way until it is measured: the
wrapper verifies the baseline suite before scoring, which runs the whole suite
once. That is a full test invocation — the same 750–950 ms toll again, plus the
suite's own work — and it belongs to the wrapper, not to ditto. Nobody should
repeat the 4.5 s figure as ditto's cost. It never was.

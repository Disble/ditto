# Experiment — what the gated path actually saves

Written before the measurement. This is the hypothesis the whole line of work
exists to test, and the first one that can say it was worth doing.

## Why this is being measured

Everything so far establishes that the gated path is *safe*: same verdicts, same
golden output, byte for byte. Nothing yet establishes that it is *cheaper*. The
projection was 750-950 ms saved per mutant, but a projection is not a
measurement, and the wiring added work of its own — planning, instrumenting,
one compilation, a sandbox.

## The counter, and why this one

The thing being removed is an invocation of the test command. That is observable
from outside without reading ditto's internals: point `WithTestCommand` at a
script that appends a line to a file, and count the lines.

- Ordinary path: one invocation per mutant.
- Gated path: one invocation per mutant it could **not** gate, and none for the
  rest, because those run from the shared binary instead.

Exact, integer, identical on every machine. Wall clock is reported beside it and
gated on never.

## H1 — the two paths agree

Same total, same killed, same survived, same score.

*Falsified by any difference. If this dies, nothing else in the note matters.*

## H2 — the test command stops being invoked for gated mutants

Invocations drop from N to the number of mutants the gated path refused.

*Falsified if the gated run invokes the command as many times as the ordinary
one — which is what a wiring that silently did not engage would look like, and
has already happened once in this work.*

## H3 — the run gets faster, reported and not gated

At least 2× on this fixture, whose suite is small enough that the invocation toll
dominates.

*Falsified below 1.5×.* A suite that does real work keeps that work, so this
number belongs to this fixture and travels nowhere.

## Controls

**The counter must be shown to count.** Before any comparison, run the ordinary
path and check the count equals the mutant total. A counter that reports the same
number whatever happens is measuring nothing.

**The harness must be shown to refuse.** H1's comparison is only worth something
if a difference would fail it, so it is checked against a deliberately wrong
expectation first.

## Fixture

A throwaway Go module outside every repository: one package with comparison
sites and integer literals, a real suite that kills some mutants and not others,
and a mutation test that takes the gated option from the environment. One warm-up
discarded, three measured rounds, the order of the two paths rotated.

## Results

Run 2026-08-13 on a throwaway module. Warm-up discarded, three rounds, the two
paths rotated so neither is always first.

| Round | Path | total | killed | survived | invocations | wall |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | ordinary | 18 | 9 | 9 | 18 | 17304 ms |
| 1 | gated | 18 | 9 | 9 | **8** | 8792 ms |
| 2 | gated | 18 | 9 | 9 | **8** | 9019 ms |
| 2 | ordinary | 18 | 9 | 9 | 18 | 16955 ms |
| 3 | ordinary | 18 | 9 | 9 | 18 | 17076 ms |
| 3 | gated | 18 | 9 | 9 | **8** | 8784 ms |

### The controls, first

**The counter was shown to count, and it caught a mistake doing so.** Its first
reading was 36 for 18 mutants — exactly double, because `go test -o` compiles
*and runs*, so the release had happened twice. The number disagreed with the
mutant total and the reason was the harness. Compiling with `-c` and running the
binary once gave 18 for 18.

That is also the evidence that the counter is not stuck: it moved 0, 36, 18, 8
across runs.

**The verdict comparison was shown to refuse — after the fact, which is the point
worth recording.** This note declared that control and the first version of these
results did not run it. Comparing 18/9/9 with 18/9/9 twice felt like checking,
and a control written down is not a control run.

Run afterwards: with the gated path deliberately made to select nothing, so every
gated mutant executes unmutated, the two paths separate at once.

| Path | total | killed | survived |
| --- | --- | --- | --- |
| ordinary | 18 | 9 | 9 |
| gated, deliberately broken | 18 | **3** | **15** |

Restored, both report 18/9/9 again. A difference of that kind would have been
seen. Before this was run, H1 rested on two numbers agreeing and nothing showing
that they could have disagreed.

### H1 held

Eighteen mutants, nine killed, nine survived, in all six runs. The gated path
reaches the ordinary path's verdicts on a fixture it has never seen.

### H2 held

Invocations of the test command fell from **18 to 8**. Ten mutants ran from the
shared binary and never started it. This is the thing the work set out to remove,
and it is removed.

### H3 was optimistic, and survived

Predicted at least 2×, falsified below 1.5×. **Measured 1.97×, 1.88× and 1.95×.**

Under two, in every round. The prediction was not met and the kill line was not
crossed, and rounding 1.95 up to "about 2×" would be reporting the prediction
instead of the measurement.

### Why it is not more, which is the useful part

Eight of eighteen mutants — 44% — still pay full price, and they are all the same
shape. Every integer literal in this fixture sits *inside* a comparison:
`score > 90` is one comparison site holding one literal site. The nesting rule
drops the inner one, because rewriting both spliced the outer over the inner and
produced a file that did not parse.

On dharness that rule cost 3 sites in 38, 7.9%. Here it costs 44%, because
comparisons against literal thresholds are most of what this fixture does. The
rule is right and its price is not a constant.

**That is where the next gain is.** The two sites are the same expression tree,
and one multi-arm gate could carry both — the machinery for several mutants at
one site already exists, and this is the same problem one level down.

## What this does NOT establish

- **One fixture, and a small suite.** The invocation toll dominates here by
  design. A suite that does real work keeps that work, and the ratio falls
  towards one; the *counter* is what travels, not the clock.
- **Nothing about a large repository.** Eighteen mutants in one package.
- **Nothing about the nine viruses this path refuses**, which keep the ordinary
  path in full.

# What ditto measures about itself

Ditto reports a mutation score. **The score is not how ditto's own quality is
judged**, and this document exists because that sentence is a change of
direction, arrived at on 2026-08-28 after two systematic reviews.

The distinction that organises everything below:

> **The score measures the user's tests. These four metrics measure whether
> ditto's answer can be believed.** Only the second is ditto's quality.

## The four

| # | metric | threshold | where the threshold comes from | the decision it drives | status |
| --- | --- | --- | --- | --- | --- |
| 1 | `nonViableRate`, broken down by virus | ≤ 1.8% | **external**: Major 1.8%, PIT 0% | which mutation operator to fix | **live** — the report names the count and the viruses behind it, at no extra cost, because the run already knows which kills never compiled |
| 2 | `verdictsWithoutAReason` | 0 | binary — the reason is there or it is not | can this kill be audited? | **live** — `Assertion`, `BuildFailed` and `Deadline` are read; `Unknown` when the command is not `go test -json` |
| 3 | `verdictsOutsideTheChange` | 0 | measured: addresses the diff never touched | did this run measure my change? | **live** — guarded by an address discriminator with its own control, and a widened scope now announces itself |
| 4 | `mutantsPerRelease` | bidirectional ratchet | the previous number | does this change make the gate cost more? | **live** — and it fired three times while this was being written |

### 1. `nonViableRate` — mutants that do not compile, over mutants generated

A mutation that does not compile is not a mutant. It is a defect of the
**generator**, and it has an external benchmark: Major reports 1.8%, PIT 0%.
Measured here: **7.3%** on dharness (94 of 1293) and **12.8%** on one ditto file
(10 of 78). An order of magnitude off the state of the art.

Reported per virus, not as one rate — `internal/schemata/falsekill_bench_test.go`
already prints *"viruses that produced a mutant that does not build: N of 14"*
and `docs/experiments/false-kills.md` already carries the per-file breakdown.
The harness exists; it is skipped behind `DITTO_FALSEKILL_ROOT` and has to be
promoted from probe to gate.

**This is not a scoring question.** It was treated as one for a long time here,
and that was the mistake: `docs/experiments/what-the-field-already-decided.md`.

### 2. `verdictsWithoutAReason` — every verdict carries why

Today `internal/cmdtestrunner/cmdtestrunner.go` collapses every non-zero exit
into "killed": a failed assertion, a mutant that does not compile, a timeout, a
missing binary. Two categories where the field uses four or more.

**Not a rate — a presence check.** Either a verdict names its reason or it does
not, so the threshold is zero with no measurement needed to justify it.

Both missing reasons are available at no extra cost:

- **build failure**: `go test -json` emits `"Action":"build-fail"` and a `fail`
  event carrying `"FailedBuild"`. Measured on go1.27: neither appears when a test
  fails for real. Exit codes cannot tell them apart — `go test`, `go build`,
  `go vet` and `go test -json` all exit 1 — so this is the only cheap route, and
  it is JSON rather than prose, which is what this project already requires of a
  verdict.
- **deadline**: ditto's own clock. It has none today; see `docs/backlog.md`.

The limit, stated plainly: the JSON route works only when the test command *is*
`go test`. Ditto's default is (`release.go`). A user who configures `make` or
`gotestsum` gets no build-failure reason, and the metric has to say so rather
than report zero.

### 3. `verdictsOutsideTheChange` — the run measured what changed, and says when it did not

A scoped run must not report mutants at addresses the change never touched.
Measured on perfbench's own fixture: a scope that does not keep each range beside
its file mutates `pkg0/gate.go:12` when the change touched `pkg0` at line 4 —
**new addresses, not repeats**.

The per-file scope already holds. What is missing is the announcement:
`internal/staged/staged.go` fails open to whole files on an unparseable diff and
widens **every** file, not only the offending one, and `RunStaged` never prints
the `Derived`/`Reason` it carries — only `--dry` does. A run can report survivors
on lines nobody touched and say nothing.

### 4. `mutantsPerRelease` — what one full run has to pay for

Recorded in `perf/baseline.json`, ratcheting in both directions.

**The downward direction is a precision alarm, not a performance one.** If the
count falls without the code falling, mutants stopped being generated, the
denominator shrank, and the score improved by itself.

## What is deliberately NOT a metric

### The score

It ships **with its composition beside it** — generated / non-viable / killed by
reason / live, which is the shape Major reports — or it does not ship. It is
never a gate.

Four independent peer-reviewed results say a bare ratio is uninterpretable:

- **IEEE TSE 2021** (Petrović, Ivanković, Fraser & Just — Google, and the authors
  of Major and Defects4J) abandoned the ratio: *"it is neither concrete nor
  actionable, and it does not guide testing."* They report individual surviving
  mutants at code-review time instead.
- **ISSTA 2016** (Papadakis et al.): fewer than 5% of mutants are subsuming, and
  *"the proportion of all mutants killed is not generally strongly correlated
  with the proportion of subsuming mutants killed."* 62% of arbitrary experiments
  hit a Type I error through it.
- **ICSE 2018** (Papadakis, Shin, Yoo & Bae): *"all correlations between mutation
  scores and real fault detection are weak when controlling for test suite
  size"*; under 1% of mutants carry the fault signal.
- **ICSE 2017** (Chekam et al.): the relationship with fault revelation is a
  threshold effect — below a high score the number is *"completely disconnected
  from fault-revelation."* Ditto's own gate runs at 0.5, and the file measured
  most closely sits at 0.58–0.72. That is below.

`mutmut` and `cargo-mutants` publish no score at all. PIT publishes two.

### The fifth metric, named and not gated

`gatedVerdictsDifferingFromOrdinary`. Ditto ships two execution paths and they
disagree about the same mutant — `docs/experiments/disagreement-class.md` has it
measured, and `fidelity_probe_test.go` **asserts the disagreement is still
present**. Two implementations, two answers, at most one right.

**A focused review settled the threshold, and the answer is that there is no
published basis for a tolerant one.** Nobody has measured a disagreement rate
between schemata and compile-per-mutant execution, so nobody has proposed a
tolerance. The three nearest checks in the literature all demanded **exact set
equality**: the Clang mutation paper verified its invalid-mutant labels against
actual compilation failures, AccMut carries an equality theorem, and Dextool
assumes the generator is wrong and mitigates with a fallback and a sanity check
rather than tolerating a percentage. An execution optimisation is validated by
equality, not by proximity.

So the threshold is **0 — over the mutants both paths consider viable**, and that
qualifier is the whole finding. Comparing verdicts on a mutant one path refuses
to build is a category error, not a discrepancy: under schemata the compiled code
is not the mutant, it is the original plus a branch, and the source-text mutant
does not exist as a program. Running the branch does not rescue a mutant that
never built; it runs a different program that happens to evaluate the mutated
expression. **Neither path should score it.**

Ditto's one measured disagreement is exactly that case, so it is outside the
comparison rather than inside it. What is still unmeasured is the population of
disagreements among mutants both paths DO run, and that is what keeps this
ungated.

Two more things the review named, both recorded in `docs/backlog.md`:

- **Schemata is neither a superset nor a subset.** There are mutants the ordinary
  path compiles that a schema cannot express at all — compile-time constant
  contexts, static initialisers, type-incompatible branches. Stryker.NET skips
  constants for exactly this reason, and Stryker excludes static mutants from the
  score outright. So an expressiveness gap is expected and belongs in its own
  non-fatal metric, never in the agreement one.
- **Branch overhead is real and measured** — the Clang paper reports a 120%
  delay on CppCheck from the inserted ternaries. Where a timeout is a kill, the
  two paths can disagree legitimately without either being broken, so timeouts
  come out of the comparison.

## How a verdict is classified

Settled by the literature, not chosen here. See
`what-the-field-already-decided.md` for the sources.

| the mutant | treatment | why |
| --- | --- | --- |
| does not compile | out of the numerator **and** the denominator | the kill predicate is undefined for a program that does not exist |
| timed out | **a kill**, reported as its own reason | unanimous across PIT, Stryker and Infection |
| suspected equivalent | counted as **survived**, and the metric renamed to a stated lower bound | equivalence is undecidable |

The canonical definition, Zhu, Hall & May, *ACM Computing Surveys* 29(4), 1997,
Definition 3.1: **`S = D / (M − E)`** — dead mutants over all mutants minus the
equivalent ones. The denominator has never been "every mutant generated".

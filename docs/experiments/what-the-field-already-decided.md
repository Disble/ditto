# What the field already decided

Two systematic reviews, run independently on 2026-08-28 against separate corpora
— peer-reviewed literature, and the source and documentation of mature tools.
They were commissioned because a design argument here had run aground, and the
grounding was the point: **the question had already been answered, and answered
differently, decades ago.**

## What ran aground

Ditto collapses every non-zero exit of the test command into "mutant killed".
Measured on one file — `internal/schemata/instrument.go`, 78 mutants — it
reported 50 killed, of which **10 did not compile** and **1 hung until its
timeout**. 22% of the kills were not assertions.

Three score formulas were then compared on that data:

| | |
| --- | --- |
| unjudged counted as killed | 0.641 — what ditto does today |
| unjudged counted as survived | 0.500 |
| unjudged excluded from the denominator | 0.582 |

Only the third is invariant when unjudged mutants increase and the tests do not.
But under a **biased** loss all three move, and the third moves most. That is
where it stopped: if no formula survives, the framing is suspect.

It was.

## Finding 1 — "unjudged" is not one class. It is three.

The literature does not have a bucket to pick a formula for. It has three
classes with three established, non-interchangeable answers.

**A mutant that does not compile is removed from the numerator and the
denominator.** Not a modelling choice: the kill predicate is undefined for a
program that does not exist. Zhu, Hall & May, *ACM Computing Surveys* 29(4),
1997, Definition 3.1:

> "Adequacy Score S = D / (M − E), where D is the number of dead mutants, M is
> the total number of mutants, and E is the number of equivalent mutants."

The denominator has never been "every mutant generated". The review found **no
source** that counts a non-compiling mutant as killed.

**A timeout is a kill**, reported as its own *reason* beside assertion and
exception. It is not unjudged at all. Unanimous in the tools: PIT
`TIMED_OUT(true)`, Stryker `TimedOut` detected, Infection `TIMED_OUT` defeated.

**A suspected-equivalent mutant counts as survived**, and the metric is renamed
to *Mutation Score Indicator* — an explicitly stated lower bound. The rename is
the load-bearing part.

So of the 11: **the 1 timeout is a legitimate kill and stays**; the 10 leave the
computation entirely. On the measured file that is **49/68 = 0.721**, not 0.641
and not 0.582. And the 12.8% non-compilable rate is filed where it belongs — as
a **generator defect**, against a benchmark: Major 1.8%, PIT 0%.

## Finding 2 — the invariance analysis answered a question the field settles by definition

No source picks a denominator by asking which formula is stable under population
change. Zhu et al. fix `D/(M−E)` as a definition, and everything after is named
as an approximation to it.

The biased-loss worry is real, and the literature confirms both the worry and
that **no denominator repairs it**: Gopinath et al., ICSE 2016, on biased
sampling being potentially worse than random; Papadakis et al., ISSTA 2016, where
a biased subclass flips 62% of experimental conclusions; and the
predictive-mutation-testing paper where a systematic subclass collapsed AUC from
0.83 to 0.51. The remedy is not a better formula — it is to **publish the
composition next to the ratio** and stop treating the scalar as comparable.

## Finding 3 — the field already says: do not report one number

This is the finding that changes direction.

- **IEEE TSE 2021** — Petrović, Ivanković, Fraser & Just, at Google, and Just is
  the author of Major and Defects4J. They abandoned the ratio: *"it is neither
  concrete nor actionable, and it does not guide testing."* They report
  **individual surviving mutants** at code-review time instead, capped at seven
  times the number of files in the change.
- **ISSTA 2016** — fewer than 5% of mutants are subsuming, and *"the proportion
  of all mutants killed is not generally strongly correlated with the proportion
  of subsuming mutants killed."*
- **ICSE 2018** — *"all correlations between mutation scores and real fault
  detection are weak when controlling for test suite size"*; under 1% of mutants
  carry the fault signal.
- **ICSE 2017** — fault revelation is a threshold effect; below a high score the
  number is *"completely disconnected from fault-revelation."* **Ditto's gate
  runs at 0.5.** That is below.

And in the tools: **`mutmut` and `cargo-mutants` publish no score at all.** PIT
publishes two — *mutation score* over all mutants and *test strength* over
covered ones. Major reports generated / covered / killed-by-reason / live.

## Where the two reviews disagree, kept rather than smoothed

The literature is unanimous that non-compiling mutants leave both sides. **The
tools are not.** PIT marks `NON_VIABLE` as detected — counted as killed.
Infection counts `SYNTAX_ERROR` in the MSI numerator. `go-mutesting` keeps
`skipped` out of the numerator but **inside** the denominator. StrykerJS,
gremlins, MutPy and cargo-mutants remove them from both.

So ditto's behaviour has precedent in shipped tools. It has none in the
literature, and the tools that keep it do not defend it.

## One claim from the review that was wrong, and checked

The tooling review reported that `go test` exits 1 for a failed test and **2**
for a package that does not compile, and that a Go tool reads that. Measured on
go1.27.0:

    test that fails          go test  -> 1
    package that will not compile      -> 1     (not 2)
    go build / go vet / go test -json  -> 1
    gotestsum                          -> 1
    make                               -> 2     (make's own code for any recipe failure)

`docs/experiments/fixing-false-kills.md` H1 was right: the exit code cannot tell
them apart.

**But checking it found the cheap route the whole argument had been missing.**
`go test -json` emits structured events:

    {"ImportPath":"...","Action":"build-fail"}
    {"Action":"fail","Package":"...","FailedBuild":"..."}

Neither appears when a test fails for real. The distinction costs **no extra
subprocess** — only `-json` and two fields — and it is JSON rather than prose,
which is what this project already demands of a verdict. It refutes the "+25%
per mutant" premise that three rounds of argument had rested on.

## What changed as a result

`docs/metrics.md` — the score is no longer how ditto's quality is judged. It
ships with its composition beside it or not at all. Four metrics take its place,
and each names the decision it drives and where its threshold comes from.

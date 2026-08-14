# Experiment — where the false kill gets fixed

Written before the measurement.

## The research question

**To what extent** does each candidate mechanism eliminate the mutants ditto
scores as killed without ever running them, and **at what cost in subprocesses
per mutant**, over the 1293 mutants its fourteen viruses generate on `internal/`
of dharness at `aa605f4`, measured on throwaway copies in August 2026?

| Element | This question's |
| --- | --- |
| Interrogative phrase | to what extent, and at what cost |
| Variable, the counters that move | false kills eliminated, of 94; subprocesses added per mutant |
| Population, unit of analysis | 1293 mutants over 42 files of `internal/` |
| Space and time | dharness at `aa605f4`, Go 1.25, August 2026 |

**FINER** — *Feasible*: the probe exists and one full pass costs 300 s.
*Interesting*: the answer decides what ditto ships, and every score it has ever
reported is inflated until then. *Novel*: nothing reports which mutants failed to
compile — that absence is the defect. *Ethical*: fixture and tool are both
throwaway copies; the probe rewrites files in place and is skipped without a root
naming one. *Relevant*: ditto's reported score, and dharness's gate at 0.80.

**PICOT** — **P** the 1293 mutants · **I** the mechanism under test · **C** the
same mutants with no mechanism, which is the 94 already measured · **O** false
kills eliminated and subprocesses added per mutant, both exact integers ·
**T** one pass per mechanism, bound to `aa605f4` and Go 1.25.

## Prior measurements — inputs, not verdicts

Measured already, and used here as the control the mechanisms are compared
against. None of it answers the question; all of it constrains the answer.

- **1293 mutants, 94 of which do not build — 7.3%.** Reproduced twice, once with
  the probe inside the worktree and once with both sides sandboxed, identical to
  the mutant.
- **The causes**: 42 `index -1 must not be negative`, 24 `declared and not
  used`, 21 `operator - not defined on … string`, 6 `duplicate case`, 1 unused
  import.
- **The viruses**: 45 of 228 Integer Decrement, 25 of 142 Comparison Replace,
  17 of 63 Arithmetic, 4 of 8 Arithmetic Assignment Invert, 3 of 228 Integer
  Increment. Refusing a whole virus is not available: Integer Decrement's 228
  mutants include 183 that build.
- **Every one of the five causes is a `go/types` diagnostic**, verbatim. None
  comes from a later compiler phase.
- **Building before running costs a median 25% of a run** (`fixing-false-kills.md`),
  and the exit code cannot tell a dead mutant from one that never ran.

## The hypotheses

Each is a tentative answer to the question. Each can be false while the others
are also false.

**H1 — the AST is enough.** Guards that read only the syntax tree, refusing a
mutation at incubation by the shape of its site, eliminate all 94.
*Prediction: 94 of 94 eliminated, 0 subprocesses added per mutant.*
*Falsified by any false kill remaining, or by any subprocess added.*

**H2 — an in-process type check is enough, and free per mutant.** A `go/types`
check over the mutated file eliminates all 94 without spawning a process for
each one; whatever import resolution costs is paid once per package.
*Prediction: 94 of 94 eliminated, 0 subprocesses added per mutant.*
*Falsified by any false kill remaining, or by any per-mutant subprocess.*

**H3 — building each mutant is enough, at one subprocess each.** Compiling the
mutant before running its tests, and reporting "did not build" separately,
eliminates all 94.
*Prediction: 94 of 94 eliminated, exactly 1 subprocess added per mutant.*
*Falsified by any false kill remaining, or by more than one subprocess.*

**What would refute all three:** any of them leaving a false kill behind. The
probe builds the *package*; a mutation that compiles as a package and breaks the
*test* build is a false kill none of these three mechanisms addresses, and
`false-kills.md` already records the 94 as a lower bound for that reason. If the
whole set falls, the answer is that this is not fixable where the mutant is made,
and the next conjecture is about where the verdict is read — ditto infers a kill
from a non-zero exit code, and could instead require positive evidence that a
test ran and failed.

## Decision rule, fixed in advance

- Exactly one mechanism eliminates all 94 → ship it.
- More than one → ship the one adding fewest subprocesses per mutant; a tie goes
  to the one that loses fewest viable mutants.
- None → return to the question with the verdict-reading conjecture above.

Nothing here belongs inside a hypothesis.

## Controls

0. **The decoy.** A throwaway repository with `GIT_DIR` and `GIT_INDEX_FILE`
   aimed at it, whose `HEAD`, commit count and local config must be unchanged
   after every pass. It protects every number below.
1. **The baseline reproduces** — the unchanged probe reports 1293 and 94 before
   any mechanism is added.
2. **Each mechanism is shown to refuse.** A mutant known not to build must be
   classified as such, and the unmutated file must not be. A mechanism that
   classifies everything, or nothing, is measuring its own error.
3. **No mechanism removes a mutant that compiles.** Total mutants must fall by
   exactly the number of false kills eliminated. A larger drop hides real
   survivors, which is worse than the defect being fixed.

## Fixture

`$SCRATCH/dharness-fixture` — the copy being mutated, identity confirmed by
`head -1 go.mod` reading `module github.com/Disble/dharness` before anything is
measured. `$SCRATCH/ditto-sandbox` — the probe itself, copied out of the
worktree, because a run launched from a checkout with work in it inherits that
checkout's git addressing.

`DITTO_FALSEKILL_ROOT` points at the fixture and the probe is skipped without it.
`DITTO_FALSEKILL_ONLY` narrows it and takes a value with no slashes.

## Results

Not yet run. Verdicts: 0 of 3.

## What this note corrects

Its first version had no research question, and both of its defects follow from
that.

1. **Its H1 was a task.** "The 94 come from at most 4 viruses" is a number needed
   to design a mechanism, not a tentative answer to which mechanism to ship. It
   was measured, committed, and reported as though something had been settled.
   The measurement stands and is above, as an input.
2. **Its H2 and H3 were one hypothesis.** "Refusal removes ≥75" and "refusal
   removes <75" partition the outcome: exactly one survives every possible
   measurement. The second was the first's kill line promoted to a hypothesis,
   and the threshold that selected an action was a decision rule, which now has
   its own section.

## What this does NOT establish

- **The 94 are a lower bound.** The probe builds the package, not the test
  binary.
- **One repository.** The five causes are shapes of ordinary Go; their frequency
  is dharness's.
- **Nothing about the gated path.** Whether refusing at incubation and dropping
  after generation give the same counts there is a separate question: one
  non-viable mutant fails the shared build for its whole file.
- **Nothing about whether the defect changes a verdict.** Whether a corrected
  score crosses dharness's 0.80 is a separate question.

# Experiment — does `ditto staged` reach the verdicts the wrapper reaches?

Written before the measurement.

## The research question

**To what extent** does `ditto staged` report the same mutants and the same
verdicts as `dharness/tools/mutationstaged`, over the staged changes of a real
repository, on ditto `docs/removing-the-wrapper` and dharness `12bbb45`,
August 2026?

| Element | This question's |
| --- | --- |
| Interrogative phrase | to what extent the two agree |
| Variable, the counter that moves | mutants reported, killed, survived — three exact integers |
| Population, unit of analysis | the mutants of a staged change in dharness's own tree |
| Space and time | ditto `docs/removing-the-wrapper`, dharness `12bbb45`, Go 1.27, August 2026 |

**FINER** — *Feasible*: two runs per case, minutes each. *Interesting*: nothing
should be deleted from dharness until this is answered, and nothing should be
built in autoreas-bridge until it is deleted from dharness. *Novel*: the command
did not exist until this branch. *Ethical*: a clone, never the checkout.
*Relevant*: whether `tools/mutationstaged` can go.

**PICOT** — **P** the mutants of one staged change · **I** `ditto staged` ·
**C** `go run ./tools/mutationstaged` on the identical staged tree · **O**
mutants, killed and survived, as integers · **T** one run per path per case.

## Method

dharness is cloned into a scratch directory and staged there; the checkout it was
cloned from is never touched. Both paths are given the same test command, the
same exclusions and the same threshold, so the only variable is which of them
computed the scope and materialised the tree.

A single agreement proves little — two paths could agree on a number neither of
them derived. So the case is moved: a second staged change, of a different shape,
must move both paths to the same new numbers.

## Hypotheses, and what kills each one

**H1 — the two paths report the same verdicts on one staged file.**
*Prediction: the same mutants, the same killed and the same survived.*
*Falsified if any of the three integers differs.*

**H2 — they still agree when the change spans two files.** This is the shape that
used to break: a scope holding one flat set of ranges makes every file answer to
every range, and the count grows as the square of the file count.
*Prediction: the same three integers again, and different from H1's.*
*Falsified if either they disagree, or the numbers did not move from H1 — the
second meaning the case never changed and H1's agreement was with a constant.*

**What would refute both:** the two agreeing on numbers that do not move with the
input at all, which would say the comparison is reading something other than the
run.

## Decision rule, fixed in advance

- Both hold → `tools/mutationstaged` can be deleted once a ditto release carries
  the command.
- H1 holds and H2 dies → the per-file scope is wrong in the new path and nothing
  is deleted.
- Numbers do not move between cases → the instrument is wrong; nothing is
  concluded and the harness is fixed first.

## Results

One staged file, `internal/cli/mutate.go`, one condition rewritten equivalently
so the suite stays green:

| Path | Mutants | Killed | Survived | Score |
| --- | --- | --- | --- | --- |
| `tools/mutationstaged` | 4 | 1 | 3 | 0.25 |
| `ditto staged` | 4 | 1 | 3 | 0.25 |

Same three survivors either way: Comparison, Integer Decrement, Integer
Increment. The new path also gives each an address — `mutate.go:70:19`,
`70:21`, `70:21` — which the old one cannot, because dharness pins ditto v0.2.0
and addresses arrived in v0.3.0.

Two staged files, `internal/cli/flags.go` and `internal/cli/mutate.go`:

| Path | Mutants | Killed | Survived | Score |
| --- | --- | --- | --- | --- |
| `tools/mutationstaged` | 9 | 5 | 4 | 0.56 |
| `ditto staged` | 9 | 5 | 4 | 0.56 |

### Controls

- **The case moved.** 4 mutants against 9, 0.25 against 0.56, and both paths
  moved together. The agreement is reproducible rather than a constant the two
  happen to print.
- **Never the real checkout.** The measurement ran in a clone; `git status` in
  `D:/dev/disble/dharness` was checked after and showed the working tree as the
  commit left it.

### H1 — corroborated

4, 1 and 3 on both paths, and the same three survivors by mutator.

### H2 — corroborated

9, 5 and 4 on both paths, and different from H1's numbers, so the input reached
the instrument.

## Verdicts: 2 of 2

## Conclusion

The command reaches the verdicts the wrapper reaches, on two shapes of staged
change, and says more about each survivor than the wrapper can. Corroborated is
not proven: two cases in one repository is what this rests on.

## What this does NOT establish

- **One repository.** Both cases are dharness. autoreas-bridge has an embedded
  frontend and a different exclusion set, and neither was run.
- **Not the fail-open path.** Both cases derived a scope. A change whose diff
  cannot be turned into ranges takes a different branch in both paths, and that
  branch was not compared.
- **Not the refusals.** A red baseline and a partially staged file are refused by
  both, and the wording differs; only the ditto side has a test.
- **Nothing about cost.** Wall clock was not compared, and the counters that
  matter for that live in `perf/baseline.json`, which measures a release rather
  than a staged run.

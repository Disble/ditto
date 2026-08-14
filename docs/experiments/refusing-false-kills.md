# Experiment — refusing a false kill instead of classifying it

Written before the measurement.

## What is being decided

`false-kills.md` measured that **94 of 1293 mutants (7.3%) do not build**, and
ditto scores every one of them as killed. `fixing-false-kills.md` searched for a
way to tell them apart at run time: the exit code cannot (H1, dead), a second
build costs a median 25% (H2), and an in-process type check is unmeasured (H4).

Every one of those asks the same question: *given a mutant that cannot compile,
how does ditto notice?* There is a rival that was never put beside them: **do not
generate it.** A virus that refuses to write `a[-1]` costs nothing at run time,
because nothing runs.

The two fixes are not exclusive, but their order is decided by one number — how
much of the 94 refusal reaches — and that number has never been measured.

## What this note corrects

Three claims were made out loud while planning this, and none of them was
measured. They are recorded because the plan they produced was wrong.

1. **"42 of the 94 are `integerdecrement` on a literal used as an index."** The
   probe records the *compiler's message*, never the virus
   (`falsekill_bench_test.go`, `reasons[message]++`). Which virus produced a
   failure has never been read. That attribution is H1 below, not a premise.
2. **"Syntax alone can refuse 48 of 94, so refusal goes first."** 48 is 42+6 from
   the same unmeasured reading, and it reordered the whole plan.
3. **"At least one package crosses dharness's threshold."** dharness gates at
   **0.80** and over the *staged* scope, not per package
   (`tools/mutationstaged/main.go:21`). The mechanism in that claim was not the
   one in the code.

## H1 — the 94 come from few viruses

Refusal is written per virus. Two viruses to teach is a cheap fix; ten is a
rewrite, and then classifying at run time wins on effort alone before any
measurement of speed.

*Prediction: at most 4 of the 14 viruses account for all 94. Falsified at 5 or
more.*

Mechanical: the probe records `infectionName` for each mutant that fails to
build, and the viruses are counted. No judgement is exercised, which is the point
— the last attempt at this number was a judgement.

## H2 and H3 — mutually exclusive, and the measurement kills one

Once H1 names the viruses, the refusals are written and the probe is re-run over
the same population. The count that moves is exact: how many of the 94 are no
longer generated.

**H2 — refusal is enough.** *Prediction: the refusals remove at least 80% of the
94 (75 or more), leaving a residue small enough that no run-time classification
is worth 25% of a run. Falsified below 75*, which selects H3.

**H3 — refusal is partial and a run-time check is required.** *Prediction: fewer
than 75 are removed. Falsified at 75 or more.*

They meet at 75 and cannot both hold. The measurement also reports what the
residue is made of, which is what H4 of `fixing-false-kills.md` would have to
carry.

80% is where the decision changes, not where the guess landed: below it, the
cheap fix leaves enough behind that the expensive one still has to be built, and
building both is worse than building the right one.

## H4 — the correction moves a verdict

Removing a false kill lowers the numerator and the denominator by the same
amount, so the corrected score is always lower whenever anything survived. How
much, and whether it crosses a line anyone acts on, is not determined by that.

*Prediction: at least one file of dharness's `internal/` scores at or above 0.80
today and below 0.80 with its false kills removed. Falsified if none does* —
which would mean the inflation is real and has never changed an outcome on this
repository, and that is worth knowing before a public report grows a category.

## H5 — dropping a mutant after generating it is the same as never generating it

The two fixes are being compared as if they were interchangeable. On the ordinary
path they may well be. On the gated path they are not obviously so: every mutant
of a file shares one compilation, so **one mutant that does not compile fails the
build for all of them**, and the file falls back.

*Prediction: killed and survived counts are identical whether the non-viable
mutants are refused at incubation or dropped after the fact, on both paths.
Falsified by any difference in either count.*

If this dies, the counting rule for the report is not a convention to be chosen —
it is a consequence to be read off.

## Controls, to be ticked off against evidence, not against having been written

1. **The baseline reproduces.** The probe is run unchanged before anything is
   added to it, and must report 1293 mutants and 94 failures again. A different
   number means something else moved, and nothing measured afterwards means
   anything.
2. **The attribution can be wrong.** After `infectionName` is recorded, one known
   mutant is checked by hand against the file it came from. A table of names that
   was never confronted with a case is a table of strings.
3. **Each refusal is seen to refuse.** For each guard written, the mutant it is
   meant to stop is confirmed to exist before the guard, and to be absent after.
   A guard that ships against a case that never occurred removes nothing.
4. **The refusals do not remove viable mutants.** Total mutants must fall by
   exactly the number of failures removed. Any larger drop means a guard is
   eating mutants that compiled — which would hide real survivors and is worse
   than the defect being fixed.
5. **H5's comparison is shown to be able to disagree.** Two counts that match
   prove nothing until one path is deliberately broken and they are seen to
   diverge. Comparing equal numbers twice is not a check.

## Fixture

**Both sides are sandboxes.** The fixture being mutated and the tool doing the
mutating are each a throwaway copy, and neither is a checkout with work in it:

- `$SCRATCH/dharness-fixture` — what gets mutated. Identity confirmed by
  `head -1 go.mod` reading `module github.com/Disble/dharness` before anything is
  measured; the first run of `false-kills.md` was made against a copy of ditto
  and reported 87.3%.
- `$SCRATCH/ditto-sandbox` — the probe itself, copied out of the worktree.

The first runs of this note were made with the probe executing **inside the
linked worktree**, which is precisely the case `AGENTS.md` names: git exports
`GIT_DIR` and `GIT_INDEX_FILE` as absolute paths from a linked worktree, and
everything spawned inherits them. Finding the development trees clean afterwards
shows that nothing happened that time. It does not show that nothing could.

`DITTO_FALSEKILL_ROOT` points at the fixture. The probe rewrites files in place
and is skipped without it. `DITTO_FALSEKILL_ONLY` narrows it and takes a value
with no slashes, because the shell rewrites anything that looks like a path.

### Control 0 — the decoy

`AGENTS.md` prescribes the measured version of "it stayed inside its sandbox":
a throwaway repository with `GIT_DIR` and `GIT_INDEX_FILE` aimed at it, whose
`HEAD`, commit count and local config must be unchanged after the run. A probe
that can reach outside its fixture reaches the decoy, and nothing real is lost.

It is control **0** because it protects every number the note contains.

## Metrics

Exact counters, per virus and per compiler message:

- mutants generated, before and after the refusals
- mutants that fail to build, before and after
- for H4: per file, killed and total, today and corrected, against 0.80
- for H5: killed and survived on each path, and the number of files whose shared
  build failed

Wall clock is reported where a run is timed and gates nothing.

## Results

### Control 0 — the decoy was untouched, and the numbers survived the correction

The first runs below were made with the probe executing inside the linked
worktree. Every number was then re-measured with the probe in its own sandbox and
a decoy repository aimed at by `GIT_DIR` and `GIT_INDEX_FILE`.

Cross-check first, on one file whose count is known from a real ditto release:

    mutants 35, of which do not build 2 (5.7%)     internal/setup/writer.go

Then the whole population, 272 s:

    mutants 1293, of which do not build 94 (7.3%)
    viruses that produced a mutant that does not build: 5 of 14
    45 of 228  Integer Decrement      25 of 142  Comparison Replace
    17 of  63  Arithmetic              4 of   8  Arithmetic Assignment Invert
     3 of 228  Integer Increment

Identical to the first harness, to the mutant. And the decoy:

    HEAD 2f092c10 (unchanged)   commits 1   user.name decoy-untouched   0 changed

The three development trees were unchanged as well — but that is the weaker
evidence, and it is recorded as the weaker evidence. A clean tree says nothing
happened this time. The decoy says the probe had somewhere to reach and did not.

### Control 1 — the baseline reproduces

The probe was run unchanged, over a fresh throwaway copy, before anything was
added to it. 300 s.

    mutants 1293, of which do not build 94 (7.3%)

The same 1293 and the same 94 as `false-kills.md`, and the same heavy files:
14 of 220 in `internal/report/human.go`, 12 of 68 in `internal/project/detect.go`,
10 of 57 in `internal/setup/files.go`, 10 of 46 in `internal/tool/command.go`.
Ticked.

### The causes, counted rather than approximated

`false-kills.md` reported three of these with a tilde. They are exact:

| Count | Cause |
| --- | --- |
| 42 | `index -1 must not be negative` |
| 24 | `declared and not used` — 11 `err`, 3 `raw`, 2 `info`, and 8 others once each |
| 21 | `operator - not defined on ... string` — 4 `body`, 3 `linePrefix`, 14 others once each |
| 6 | `duplicate case` — 2 each of `1` and `0`, 2 of `syscall.Errno` |
| 1 | `"bytes" imported and not used` |

42+24+21+6+1 = 94. The earlier note's `~22` and `~20` were 24 and 21.

### H1 is dead, on its kill line

Predicted at most 4 of the 14 viruses, falsified at 5 or more. Measured **5**.

| Fail | Produced | Virus |
| --- | --- | --- |
| 45 | 228 | Integer Decrement |
| 25 | 142 | Comparison Replace |
| 17 | 63 | Arithmetic |
| 4 | 8 | Arithmetic Assignment Invert |
| 3 | 228 | Integer Increment |

45+25+17+4+3 = 94.

**The second column is what the first one hides.** Integer Decrement produces 228
mutants and 45 of them fail — so the virus cannot be switched off, or 183 viable
mutants leave with it. Arithmetic Assignment Invert fails on **half of everything
it writes here**, 4 of 8, and is still not a candidate for removal at 8 mutants.

Whatever refusal is written has to name a **site shape**, not a virus. That is the
finding H1 was for, and it arrives against the prediction rather than with it.

### Control 2 — the attribution can be wrong, and was confronted with a case

The counters were checked against source by hand, in both directions:

- `internal/app/app.go` holds exactly two literal index sites, `args[0]` at
  lines 32 and 46. Integer Decrement turns `0` into `-1`, giving `args[-1]`. The
  per-file table reads `2 of 15: internal/app/app.go`. The count is not merely
  plausible, it is the same number.
- `internal/project/detect.go:383` holds `switch len(runners) { case 0: … case 1: … }`.
  Increment on `case 0` collides with `case 1`; Decrement on `case 1` collides
  with `case 0`. The table carries both directions — `2 Integer Increment →
  duplicate case 1` and `2 Integer Decrement → duplicate case 0` — with detect.go
  as the example for each.

`virusOf` refuses instead of trimming, so a label whose prefix did not match would
have failed the run rather than filled the table with a virus named after a path.

Ticked.

### The attribution is passive

The modified probe reported the same 1293 mutants and the same 94 failures as the
unchanged one. Recording who caused a failure did not change what was generated
or built, which is what makes the two runs comparable rather than similar.

### What the messages are, which was not the question and matters anyway

Every one of the five causes is a **`go/types` diagnostic**, verbatim: `index -1
… must not be negative`, `declared and not used`, `operator - not defined on …`,
`duplicate case … in expression switch`, `imported and not used`. `go build`
reported them, but none of them comes from a later compiler phase.

That licenses a hypothesis rather than a conclusion, and it belongs to
`fixing-false-kills.md`'s H4:

**H6 — an in-process `go/types` check reproduces every one of these.**
*Prediction: 94 of 94 are reported. Falsified by one it misses*, which would mean
the check cannot replace the build and the residue needs a subprocess anyway.

It is written here because the data arrived here. It has not been run.

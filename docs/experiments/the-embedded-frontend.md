# Experiment — can `ditto staged` replace a wrapper in a repository whose index does not compile?

Written before the measurement. Nothing below has been run.

## The research question

**To what extent** can `ditto staged` reach the verdicts `tools/mutationstaged`
reaches in autoreas-bridge, **given that a checkout of its index does not
compile** — measured over the mutants of staged changes in each of its two
compilation regions, on ditto v0.4.0 and autoreas-bridge `5bdd2ff`, August 2026?

| Element | This question's |
| --- | --- |
| Interrogative phrase | to what extent the command reaches the wrapper's verdicts |
| Variable, the counter that moves | mutants reported, killed, survived; and whether a run scores or refuses |
| Population, unit of analysis | the mutants of one staged change, in the root package and in a subpackage |
| Space and time | ditto v0.4.0, autoreas-bridge `5bdd2ff`, Go 1.27, August 2026 |

**FINER** — *Feasible*: two staged changes, four runs, minutes each.
*Interesting*: it decides whether the wrapper can be deleted there, and if not,
what would have to change first. *Novel*: nothing has run `ditto staged` against
this repository. *Ethical*: a clone, never the checkout. *Relevant*: the
14 files of `tools/mutationstaged` in autoreas-bridge, and a `.gitignore` line.

**PICOT** — **P** the mutants of one staged change per region · **I** `ditto
staged` · **C** `go run ./tools/mutationstaged` on the identical staged tree,
which still carries the stub · **O** mutants, killed, survived, and the run's
verdict — scored or refused · **T** one run per path per region.

## What is already established, and is not a hypothesis

A clean checkout of autoreas-bridge does not build: `main.go` carries
`//go:embed all:frontend/dist`, `frontend/dist` is in `.gitignore`, and
`go build ./...` on a fresh clone reports
`pattern all:frontend/dist: no matching files found`. Measured, not assumed.

The wrapper worked around this inside its own sandbox, with
`stubEmbeddedFrontend` writing a placeholder `index.html` after
`git checkout-index`. `ditto staged` materialises its sandbox internally and
exposes no seam to do that — a fact about code in this repository, not a claim
needing a measurement.

39 of autoreas-bridge's production Go files are in the root package and 238 are
in subpackages. Counted with `git ls-files`.

## Hypotheses, and what kills each one

**H1 — a subpackage is unaffected.** The embed only binds the root package, so a
staged change under `internal/` should mutate exactly as the wrapper mutates it,
with no workaround at all.

*Prediction: `ditto staged` and the wrapper report the same mutants, killed and
survived, for a staged change in a subpackage.*

*Falsified if any of the three integers differs, or if either path refuses.*

**H2 — the root package refuses rather than lies.** With no `frontend/dist`, the
test command cannot compile, so the suite fails before it runs anything. Ditto
reads a failing command as a killed mutant, which is exactly the shape that once
reported 431 of 431 killed in 5.46 seconds — and the guard added for it should
catch this one too, because the baseline fails first.

*Prediction: `ditto staged` on a staged root-package change reports no score and
names the red baseline.*

*Falsified if it reports any score — which would mean the guard does not cover a
build failure, and would be a defect in ditto rather than a limitation here.*

**H3 — one tracked placeholder is the whole difference.** Committing a minimal
`frontend/dist/index.html`, with a `.gitignore` negation, should make the index
compile on its own and bring the root package back under mutation.

*Prediction: with the placeholder tracked, `ditto staged` on the same root-package
change reports the same mutants, killed and survived as the wrapper does with its
stub.*

*Falsified if the numbers differ, or if the run still refuses.*

**H4 — the placeholder changes nothing else.** A file that only exists so a
pattern matches should not move any verdict that did not depend on it.

*Prediction: H1's subpackage numbers are identical before and after the
placeholder is tracked.*

*Falsified if any of the three integers moves.*

**What would refute all four:** the two paths disagreeing in a region where both
compile — which would say the disagreement is not about the embed at all, and
send this back to the question of whether `ditto staged` reproduces the wrapper
anywhere, which `replacing-the-wrapper.md` answered for a different repository.

## Decision rule, fixed in advance

- H1, H2, H3 and H4 all hold → track the placeholder and delete the wrapper.
  The repository's index compiling on its own is a property worth having beyond
  mutation, and the cost is one file.
- H1 and H2 hold, H3 dies → the placeholder is not the answer; adopt for
  subpackages only and record the root package as uncovered, with the number.
- **H2 dies → stop, and fix ditto first.** A build failure scored as a perfect
  run is the defect this project exists to refuse, and it outranks the migration.
- H4 dies → the placeholder has a side effect nobody predicted; nothing is
  tracked until it is understood.
- H1 dies → the disagreement is not about the embed, and this note is asking the
  wrong question.

Nothing here is a preference. If every hypothesis holds, the decision is made by
the numbers rather than offered as a choice.

## Results

A clone of autoreas-bridge, staged twice, both paths given the same test command
and threshold. The wrapper was given `-v` in the clone so its report could be
read at all — without it `go test` swallows the output of a passing run, which is
its own small finding.

### H1 — corroborated

One staged file, `internal/download/decision.go`, one condition rewritten
equivalently:

| Path | Mutants | Killed | Survived | Score |
| --- | --- | --- | --- | --- |
| `tools/mutationstaged` | 4 | 4 | 0 | 1.00 |
| `ditto staged` | 4 | 4 | 0 | 1.00 |

### H2 — corroborated

One staged file in the root package, `app_defaults.go`. `ditto staged` **exits 1
and reports no score**, naming the red baseline. It does not score a run whose
build failed.

### H3 — REFUTED, and the reason is bigger than the placeholder

With `frontend/dist/index.html` tracked and a `.gitignore` negation, the index
does build — `go build ./...` on the checkout is clean, and `go test -count=1 .`
against a plain `git checkout-index` passes in 4.95s. `ditto staged` refused
anyway.

The instrument to find out why did not exist, so it was built: the refusal now
carries the test command's own output. One run then said it:

    internal/tray/icon.go:8: pattern tray-icon.ico: cannot embed irregular file
    main.go:11: pattern all:frontend/dist: contains no embeddable files
    FAIL autoreas-bridge [setup failed]

**"Irregular file" is a symlink.** A release mirrors the repository as one
symlink per file, and `go:embed` refuses to embed a symlink. The tracked
placeholder becomes a symlink like everything else, so the directory it was added
to still "contains no embeddable files" — and `tray-icon.ico`, which has nothing
to do with the frontend, fails the same way.

So the placeholder was never the fix. **No repository that uses `go:embed` can be
mutated in this sandbox**, which is a limitation of ditto and not a property of
autoreas-bridge. `docs/backlog.md` entry 13.

### H4 — undecidable, and recorded rather than skipped

It asked whether the placeholder moves anything else. H3 refuted the placeholder,
so it was never tracked and the question has no measurement behind it. It is not
"passed"; it was not run.

### The finding neither hypothesis asked for

The wrapper's root-package run reports **9 mutants, 9 killed, a score of 1.00**.
It is a false perfect score, and its own log carries the proof:

- Each mutant "died" in **0.92–2.32 s**, where one real run of that suite takes
  **4.95 s**. The subpackage run, by contrast, shows real assertion failures from
  the suite itself.
- The same `cannot embed` lines appear in the wrapper's output, followed by
  `FAIL autoreas-bridge [setup failed]`.

ooze v0.2.0 has no baseline guard and reads a failing command as a killed mutant,
so nine mutants that never compiled were scored as nine tests doing their job.
The mechanism is identical to `false-perfect-score.md` — 431 of 431 in 5.46 s —
and it is the same code: ooze and ditto build the sandbox with byte-identical
`os.Symlink` calls.

**This inverts the decision.** The root package of autoreas-bridge was never
under mutation coverage; it was under a number that looked like coverage.
Adopting `ditto staged` there does not lose 39 files of coverage, it replaces a
silent 1.00 with a loud refusal.

## Verdicts: 4 of 4

H1 corroborated · H2 corroborated · **H3 refuted** · H4 undecidable, unrun.

## Conclusion

By the decision rule fixed in advance — H1 and H2 hold, H3 dies — autoreas-bridge
adopts `ditto staged` for its subpackages and records the root package as
uncovered. What the rule could not anticipate is that the root package was
already uncovered and reporting otherwise, which makes the trade a strict
improvement rather than the loss it was written as.

The placeholder is not tracked. It fixes a real thing — a repository whose index
does not build on its own — but it does not fix this, and tracking it here would
have been a change made for a reason the measurement withdrew.

## What this does NOT establish

- **Whether the symlink sandbox can be fixed**, and at what cost. Copying instead
  of linking is the obvious answer and its price is unmeasured; it is entry 13,
  not a conclusion here.
- **One staged change per region.** Two files, one each.
- **Nothing about the frontend's own mutation testing**, which is Stryker and was
  never in question.

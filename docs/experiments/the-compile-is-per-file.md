# Experiment — what sharing one compile directory would buy

Written after the phase split of `chain-shaped-module.md` identified the cause, and before any code changes shape.

## The research question

**To what extent** does compiling the module test binaries into one directory rather than a fresh one per batch reduce compilation wall time, over ten batches on a ten-package module, on 2026-09-18?

| Element | This question's |
| --- | --- |
| Interrogative phrase | To what extent |
| Variable, the counter that moves | Compilation milliseconds for ten batches; compilations per release is the integer |
| Population, unit of analysis | Ten consecutive `go test -c -o <dir> ./...` runs over one ten-package module |
| Space and time | Revision `616c842`, a disposable ten-package module, 2026-09-18 |

**FINER** — Feasible: two loops of ten compiles, differing only in whether the output directory is fresh. · Interesting: `chain-shaped-module.md` showed the compile is charged per source file with mutants, and this says what removing that charge is worth before anything is built. · Novel: the ten-package ratio of 0.3995 was measured **with** ten compiles included, and nothing has priced them separately. · Ethical: throwaway copies only. · Relevant: it sizes the next core change and decides whether it is worth its risk.

## Why the answer is expected to be large

`go test -c -o <dir> ./...` skips any package whose output is already up to date, which backlog entry 14 measured as one relink after a mutation and none when nothing changed. A fresh directory throws that away by definition: every package is written again. So the difference between the two loops is not a tuning difference, it is whether the toolchain's own up-to-date check is allowed to work.

## Hypotheses, and what kills each one

**H1 — a fresh directory costs a full rebuild every time.** Ten fresh-directory compiles total more than 10× the first compile of the shared loop.
*Falsified if the fresh total is less than 10× the shared loop's first compile.*

**H2 — a shared directory lets the up-to-date check work.** After the first, each shared-directory compile costs a small fraction of it.
*Falsified if any shared compile after the first exceeds the first by more than a factor of two.*

**H3 — the difference is large enough to matter at release scale.** The saving over ten batches is at least a third of the ten-package gated run measured in `gated-through-the-binary.md`.
*Falsified if the saving is under 8,500 ms.*

**What would refute all of them:** the first compile of the shared loop is not comparable to a fresh one. That would mean the two loops are not measuring the same job.

## Decision rule, fixed in advance

- All three corroborated → the next core change is to give the module runner one compilation directory per release, with the measured prize as its budget and `compilations per release` as its guard.
- H1 or H2 refuted → the compile is not the lever the phase split said it was, and `chain-shaped-module.md` needs re-reading before anything is built.
- H3 refuted → the change is real and not worth its risk on this fixture; it would need a repository-shaped argument instead.

## Method

One disposable ten-package module. Loop A compiles into ten fresh directories. Loop B compiles into one directory reused ten times. Both run the same command in the same tree, with no mutation in flight, so the loops differ in exactly one thing.

That last point is also the limit of the measurement and it is stated here rather than discovered later: in a real release each batch instruments a different file, so the tree changes between compiles. Go would then relink the changed package and anything depending on it, which on a fan-shaped module is one package and on a chain is all of them.

## Results

Measured 2026-09-18, Go 1.27, windows/amd64, from a complete `.git`-free disposable copy.

**Discovery, `go list -deps -test -json ./...`:** 190, 177, 174 ms — about 180 ms per release, which is the closure's own price and is paid once.

**Loop A, ten fresh directories:** **15,466 ms** total.

**Loop B, one shared directory:**

| Run | ms |
| ---: | ---: |
| 1 | 1,405 |
| 2 | 288 |
| 3 | 168 |
| 4 | 197 |
| 5 | 189 |
| 6 | 171 |
| 7 | 166 |
| 8 | 169 |
| 9 | 167 |
| 10 | 167 |
| **total** | **3,087 ms** |

### H1 corroborated

15,466 ms against a first compile of 1,405 ms: the fresh loop cost 11.0 times its own first compile, so every fresh directory really does pay a full rebuild.

### H2 corroborated

Every shared compile after the first landed between 166 and 288 ms against a first of 1,405 ms. The up-to-date check works exactly as backlog entry 14 described.

### H3 corroborated

The saving is **12,379 ms** over ten batches, against a threshold of 8,500 ms. For scale: the ten-package gated run measured 25,607 ms in `gated-through-the-binary.md`, so this is 48% of that run's entire wall clock.

## Verdicts: 3 of 3

## Conclusion

The decision rule selects the first outcome. Giving the module runner one compilation directory per release, instead of one per batch, saves 12,379 ms of 25,607 on a ten-package module — a gated ratio of about 0.21 instead of 0.3995, before any other change.

The mechanism is Go's own up-to-date check, so nothing is re-implemented: the change is which directory the binaries are written to, and the rest follows.

## What this does NOT establish

**The tree did not change between compiles, and in a real release it does.** Each batch instruments a different file, so the saving measured here is a ceiling for a fan-shaped module and an overestimate for a chain, where instrumenting a package everything imports would relink all of them. The real figure lies between this and zero and has to be measured after the change, not cited before it.

It also says nothing about where the shared directory should live, or about who removes it. The release already reclaims sandboxes through the temporary directory, and a directory that outlives its sandbox is a new lifetime the current design does not have.

Finally, this prices the change and does not make it. `compilations per release` is the integer that would say whether it happened, and nothing counts it yet.

# Experiment — can a build-output directory hold a tracked placeholder without the build fighting it?

Written before the measurement. The results below were appended after.

## The research question

**To what extent** can `frontend/dist` carry a tracked file that makes
autoreas-bridge's index compile, **without** a frontend build leaving the working
tree dirty — measured over one build of that repository, on `32025a5`,
August 2026?

| Element | This question's |
| --- | --- |
| Interrogative phrase | to what extent a placeholder can survive a build |
| Variable, the counter that moves | files reported modified or deleted by `git status` after one build; and whether `go build ./...` succeeds from a bare index |
| Population, unit of analysis | one repository, one frontend build, one index checkout |
| Space and time | autoreas-bridge `32025a5`, vite 7, Go 1.27, August 2026 |

**FINER** — *Feasible*: one build per variant, about a minute each.
*Interesting*: it decides whether PR #1 merges as it stands. *Novel*: the side
effect was found by accident and has not been characterised. *Ethical*: a clone,
never the checkout. *Relevant*: the placeholder, `.gitignore`, and whether the
root package can be mutated at all.

## What is already established

- **The index does not compile without something in `frontend/dist`.** Measured:
  `pattern all:frontend/dist: no matching files found`, and the same checkout
  builds once the placeholder is there.
- **A build overwrites `index.html`.** Observed once, by accident: after
  `render-smoke` ran the real bundle, `git status` reported
  `M frontend/dist/index.html`.
- **A dotfile did not survive one build.** A probe file placed in `dist` was gone
  afterwards, which points at vite emptying the directory.

## Hypotheses, and what kills each one

**H1 — the dirty tree is every build, not one build.** If it is intermittent it
is a smaller problem than it looks.

*Prediction: two consecutive builds each leave `git status` reporting
`frontend/dist/index.html` modified.*

*Falsified if a build leaves the tree clean.*

**H2 — `emptyOutDir: false` lets a tracked dotfile survive, and the tree stays
clean.** vite empties the output directory by default; told not to, it should
overwrite what it produces and leave what it does not.

*Prediction: with `emptyOutDir: false` and a tracked `frontend/dist/.gitkeep`,
one build leaves `git status` clean of tracked files and `.gitkeep` present.*

*Falsified if `.gitkeep` is removed, or any tracked file is reported modified.*

**H3 — that costs correctness.** Not emptying the directory leaves last build's
assets behind, and a bundle assembled beside stale hashed files may still be
served.

*Prediction: after a build with `emptyOutDir: false`, `dist` holds files no
current build produced.*

*Falsified if the file set is identical to a clean build's, in which case the
setting costs nothing.*

**H4 — `.gitkeep` is enough for the embed.** `//go:embed all:frontend/dist` with
the `all:` prefix includes files whose names begin with `.` or `_`.

*Prediction: an index holding only `frontend/dist/.gitkeep` builds.*

*Falsified if it reports `no matching files found` or `contains no embeddable
files`.*

**What would refute all four:** the placeholder turning out to be unnecessary —
which would mean the earlier `no matching files found` had another cause and the
question is wrongly framed.

## Decision rule, fixed in advance

- H1, H2 and H4 hold and H3 dies → track `.gitkeep`, set `emptyOutDir: false`,
  drop `index.html` from the PR. The tree stays clean and the index compiles.
- H3 holds → not emptying the directory is a correctness cost, and it is not paid
  for tidiness. The choice narrows to the dirty tree or no placeholder, and that
  is a trade for the repository's owner rather than a measurement.
- H4 dies → the `all:` prefix does not do what it says here, and only a
  non-dotfile can be the placeholder.
- H1 dies → the side effect is intermittent, the problem is smaller than it
  looks, and nothing needs to change before merging.

## Results

Measured in the repository itself, restored to its committed state afterwards.

### H1 — corroborated

Two consecutive builds, each from a clean tree: both left
`M frontend/dist/index.html`. It is every build, not one.

### H4 — corroborated

An index holding only `frontend/dist/.gitkeep` builds. `//go:embed all:` does
include dotfiles, so the placeholder does not have to be a real page.

### H2 — corroborated, in one combination only

`.gitkeep` survives a build **only** with `emptyOutDir: false`; under the default
it is deleted like everything else. And tracking `index.html` dirties the tree
whatever the setting, because the build writes that name.

So the clean-tree shape is exactly one: track `.gitkeep`, not `index.html`, and
turn `emptyOutDir` off. Measured: after a build, `git status` showed no
modification beyond the swap itself.

### H3 — corroborated, and it decides this

Two attempts at this failed before one worked, and both failures were the
instrument rather than the design. Appending a comment to a source file did not
change the bundle, because the build strips comments; adding a `void`-ed
constant did not either, because the minifier removes dead code. Both times the
file count stayed at four and that read exactly like "nothing accumulates".

The direct test settles it. A stale `assets/index-OLDHASH00.js` planted in
`dist`:

| Setting | the stale file after a build | `.gitkeep` after a build |
| --- | --- | --- |
| `emptyOutDir: false` | **survives** | survives |
| default (`true`) | removed | **removed** |

Not emptying the directory is what lets a placeholder live, and it is the same
property that lets last build output live. Output names are content-hashed
(`index-CHUHrUkj.js`), so a changed bundle leaves its predecessor behind.

## Verdicts: 4 of 4

H1 corroborated · H2 corroborated in one combination · **H3 corroborated** ·
H4 corroborated.

## Conclusion

By the rule fixed in advance, H3 holding means this stops being a measurement.
There are three shapes and the measurement can price all three, but not choose:

- **Track `index.html`, keep `emptyOutDir` on** — what the pull request does
  today. No stale assets; `git status` reports one modified file after every
  build.
- **Track `.gitkeep`, turn `emptyOutDir` off** — clean tree; `dist` accumulates
  superseded chunks until something removes them.
- **Track nothing** — clean tree, no stale assets, and a clean checkout does not
  compile, so the root package cannot be mutated. Worth saying that this is no
  longer *dangerous*, only limited: since v0.5.0 ditto refuses such a run rather
  than reporting a perfect score for it.



## What this does not establish

Whether a placeholder is the right shape at all, against the alternative of the
frontend build being a prerequisite of the Go build. That is a question about
this project's build order, and no measurement here touches it.

# Experiment — should a sandbox be a copy of the tree rather than a reference to it?

Written before the measurement. Nothing below has been run.

## The research question

**To what extent** does materialising a sandbox as copies rather than symlinks
change what a release can measure, and what it costs — over the files of one
synthetic fixture and one real checkout, on ditto `main` at v0.4.0, August 2026?

| Element | This question's |
| --- | --- |
| Interrogative phrase | to what extent copying changes what is measurable, and its cost |
| Variable, the counters that move | files materialised per sandbox, sandboxes per release, wall clock per sandbox, and the count of Go features that fail in one |
| Population, unit of analysis | the files of a sandbox: 6 on the synthetic fixture, 164 on one real checkout |
| Space and time | ditto `main` at v0.4.0, Go 1.27, August 2026 |

**FINER** — *Feasible*: one function has two implementations and the counters
already exist. *Interesting*: it decides whether `go:embed` is a limitation
consumers route around or a defect ditto fixes. *Novel*: the symlink strategy has
never been measured against an alternative; it was inherited from ooze.
*Ethical*: fixtures and throwaway copies. *Relevant*: `perf/baseline.json`,
`docs/backlog.md` entry 13, and every repository that embeds anything.

**PICOT** — **P** the files of one sandbox · **I** materialising by copy ·
**C** the same sandbox materialised by symlink, which is today ·
**O** `filesLinkedPerSandbox`, `sandboxesBuiltPerRelease`, the failing-feature
count, and wall clock reported beside them · **T** one release per variant.

## The premise this note exists to check

`fsrepository.LinkAllToTemporaryRepository` creates one symlink per file. A
symlink is not a copy of a file, it is a **reference** to one, and the two differ
wherever something inspects what a path *is* rather than what it contains.

`go:embed` is one instance, measured: it refuses an irregular file, so any
package with an embed directive fails to build in a sandbox
(`the-embedded-frontend.md`). The question this note asks is whether that is one
bug or one symptom.

**And the reason to ask now.** The symlink was an optimisation for a sandbox
built *per mutant* — 48 of them on the synthetic fixture. `sandboxesBuiltPerRelease`
has been **1** since sandboxes were pooled. Whatever the trade bought when it was
paid 48 times, it is now paid once, and nobody re-measured it after the thing
that justified it went away.

## Hypotheses, and what kills each one

**H1 — the breakage is a class, not an instance.** If a symlink differs from a
copy wherever identity is inspected, `go:embed` should not be alone.

*Prediction: at least one more Go behaviour, independent of `embed`, differs
between a symlinked sandbox and a copied one.*

*Falsified if a fixture exercising file identity — `os.Stat` mode bits,
`filepath.WalkDir` type, `os.ReadFile` through a broken link, a build that reads
its own source — behaves identically under both.*

**H2 — copying costs nothing that matters, because it is paid once.** Linking is
~0.45 ms per file and a copy is more, but `sandboxesBuiltPerRelease` is 1.

*Prediction: on the real checkout of 164 files, one copied sandbox costs under
500 ms more than one linked sandbox — under 1% of a release, since starting the
test command alone is 750–950 ms per mutant.*

*Falsified if the difference exceeds 500 ms, or exceeds 1% of a measured release.*

**H3 — copying removes the breakage entirely.** Not "reduces": the fixture that
fails under links must pass under copies.

*Prediction: the autoreas-bridge root package, which reports `cannot embed
irregular file` today, builds and its suite runs in a copied sandbox.*

*Falsified if it still fails, which would mean the sandbox was never the whole
cause and this note is asking the wrong question.*

**H4 — copying only what `embed` names is not worth it.** The narrower fix
parses directives and copies those paths, leaving the rest linked.

*Prediction: it saves less than 100 ms against a full copy on the real checkout —
less than the noise on a working machine, so it buys nothing for its complexity.*

*Falsified if it saves more than 100 ms, in which case the cheaper fix has a case
and the decision is not obvious.*

**What would refute all four:** copies and links behaving identically everywhere,
including on the fixture that fails today — which would say the embed failure has
another cause entirely and the symlink is innocent.

## Decision rule, fixed in advance

- H1, H2 and H3 hold, H4 holds → **copy, always.** The strategy is wrong for a
  class of behaviour and its saving is inside the noise of one test-command
  start. Backlog entry 13 closes, and no consumer routes around anything.
- H2 dies → copying is not free at scale; the decision becomes which files must
  be real, and H4's narrower fix is the candidate.
- H3 dies → stop. The sandbox is not the cause and nothing should change until
  something else is measured.
- H1 dies, H3 holds → `go:embed` is a special case rather than a class. Fix it
  as one, and say so in the entry rather than generalising a single instance.

The counters in `perf/baseline.json` ratchet in both directions, so whichever
way this goes, `filesLinkedPerSandbox` and anything that moves with it are
re-recorded with the reason beside them.

## A fifth hypothesis, written after H2's numbers and before its own

**H5 — a hard link is a copy where it matters and a link where it costs.** A hard
link is a second name for the same bytes: it is a *regular file*, so `go:embed`
accepts it, and it copies nothing.

It is only safe because of how a mutant is written. `FSTemporaryRepository.Overwrite`
**removes** the path and then writes a new file. Removing a name leaves the
original alone; a write in place would have corrupted the repository being
measured. That is a fact about ditto's own code, read before this was run.

*Prediction: the fixture that fails under links passes under hard links, at a
cost within noise of links, and the source repository is unchanged afterwards.*

*Falsified if any of the three fails — and the third is the one that matters: a
modified source repository makes this unusable at any speed.*

## Results

Fixture: a clone of autoreas-bridge, 1960 files materialised per sandbox.

### H1 — undecidable, and recorded rather than skipped

Two independent embed sites failed — `frontend/dist` and `internal/tray/icon.go`,
which have nothing to do with each other — so the failure is not one directive.
But no behaviour **outside `embed`** was found to differ, and none was
constructed and run. The claim "it is a class" is not established; what is
established is that it is more than one instance of one class.

### H2 — REFUTED

Copying is not free at this scale. Three rounds, order rotated, same work:

| Strategy | r1 | r2 | r3 |
| --- | --- | --- | --- |
| symlink | 22.6 s | 19.8 s | 19.8 s |
| copy | 39.4 s | 35.0 s | 34.2 s |
| hard link | 19.7 s | 33.8 s | 19.1 s |

Copying costs roughly **70% more**, far past the 500 ms the prediction allowed.
1960 files is why: the strategy was chosen when a sandbox was built per mutant
and the file count was the thing being avoided, and it is still the thing.

The hard-link r2 of 33.8 s is an outlier against its own 19.1 and 19.7. Wall
clock on a working machine varies by more than half here, which is exactly why
this project gates on counters and only reports the clock. The deterministic
counter says it plainly: a copy writes 1960 files' worth of bytes, a hard link
and a symlink write none.

### H3 — corroborated

The autoreas-bridge root package, which reports `cannot embed irregular file`
under symlinks, builds and runs under copies: **9 mutants, 8 killed, 1 survived,
score 0.89**.

The control is the same run under symlinks, which refuses, naming both embed
sites.

### H4 — not run, and no longer worth running

It proposed copying only the paths an `embed` directive names, as the cheaper
half-measure if H2 died. H2 did die, but H5 answered the same question better:
there is no need to parse directives to decide which files must be real when
making all of them real costs nothing.

### H5 — corroborated, on all three

- **It fixes the embed.** Same fixture, same numbers as copies: 9 mutants, 8
  killed, 1 survived, 0.89.
- **It costs what links cost.** 19.1 s and 19.7 s against links' 19.8 s twice.
- **The source repository is unchanged.** `git status` after a full run shows only
  the change that was staged on purpose, and `git diff --quiet` on the mutated
  file passes.

## Verdicts: 5 of 5

H1 undecidable · **H2 refuted** · H3 corroborated · H4 withdrawn, unrun ·
H5 corroborated.

## Conclusion

**Hard links, falling back to copies.** A hard link is a regular file, which is
what `go:embed` requires and what a symlink is not, and it writes no bytes, which
is what the file count demands. The fallback is not decoration: hard links cannot
cross a filesystem, and a temporary directory frequently lives on another one.

The decision rule fixed in advance sent H2's death to H4's narrower fix. H5
arrived instead and is better than both branches, which is worth saying plainly:
the rule did not anticipate the answer, and the answer is not the one the rule
would have picked.

**What this changes about the repository that started it.** Nothing in
autoreas-bridge routes around anything. Its root package needs one tracked
placeholder — because `frontend/dist` genuinely is not in git, which is its own
gap and not ditto's — and then it mutates like any other package. And what that
surfaces is the point: the wrapper reported **9 of 9 killed, a perfect 1.00**,
where the truth is **8 killed and one survivor**. There is a real gap in that
package's tests that nobody could see.

## What this does NOT establish

- **Whether anything outside `go:embed` differs.** H1 is undecidable, not passed.
- **~~Windows only~~ — withdrawn.** CI runs on `ubuntu-latest`, so the suite that
  includes the regular-file assertion and both goldens passes on Linux too. But
  green there proves the sandbox is *correct*, not that hard links were *used*:
  the fallback is silent, and a copy is a regular file as well. So the split is
  now an exact counter — `HardLinked` and `Copied` — and a test fails if anything
  had to be copied while source and sandbox share a filesystem. On Windows it
  reports 2 hard links and 0 copies.

  **Linux says the same, and it was checked rather than assumed.** CI on
  `ubuntu-latest` is green on both Go legs for the commit that carries the
  assertion, and reports `DONE 355 tests` — the same count as locally, so the new
  test ran rather than being skipped past. A green there is `copied == 0` on
  Linux: the cheap path is taken, not the fallback. The verdict arrives as an
  exit code, which is the only form this project accepts.
- **Nothing about the gated path**, which builds its own instrumented file and
  compiles it once. It shares the sandbox and was not separately measured.
- **One repository's file count.** 1960 files is autoreas-bridge; the synthetic
  fixture is 6 and dharness is smaller. The ratio between strategies at other
  sizes is unmeasured.

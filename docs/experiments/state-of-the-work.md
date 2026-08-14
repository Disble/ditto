# State of the work — the gated path

Written to be read by someone with no memory of any of it. Everything here is
measured unless it says otherwise, and where it says otherwise that is the point.

## The one-sentence version

Ditto pays a fixed 750–950 ms to start `go test` for **every** mutant, which is
most of a run; the gated path instruments a file once, compiles once, and selects
each mutant at run time; it is wired, opt-in, proven equivalent on two fixtures
and **not** equivalent on a third — where it turned out to be right and ditto
wrong.

## Where everything is

| | |
| --- | --- |
| Branch | `exp/test-invocation`, 27 commits on top of `dfd8ed4`, **not pushed** |
| Worktree | `D:/dev/disble/ditto-worktrees/test-invocation` |
| dharness integration worktree | `D:/dev/disble/dharness-worktrees/ditto-integration`, branch `exp/ditto-integration`, carries only a `replace` in `go.mod` — the harness, never committed |
| dharness skill branch | `docs/measurement-skill`, three commits, **not pushed**, holds the same skill as this repo |

Scratch tools live outside every repository and are disposable. They are listed
in each experiment note that used them.

## What is proven

- **The invocation toll is the cost.** 750–950 ms per mutant regardless of what
  the suite does. `test-invocation.md`.
- **Compiling once and selecting at run time reaches the same verdicts**, on
  comparisons and on integer literals, on two files.
  `schemata-feasibility.md`, `gating-non-bool.md`.
- **Both families gate together** once sites contained in other sites are
  dropped. `gating-both-families.md`.
- **97 of 135 mutants gate on a real file** — 71.9%. `gated-gain-real.md`.
- **The saving is real and its size is not portable**: 2.7× on a package whose
  suite runs in 243 ms, 1.5× on one that runs in 1155 ms. The *counter* travels —
  invocations fell 135→38 and 35→12 — the clock does not.
- **7.3% of the mutants ditto calls killed never ran**, because they do not
  compile and a failing command is how ditto recognises a kill. `false-kills.md`.

## What is open, with the hypothesis already written

1. **H4, in-process type checking** — `fixing-false-kills.md`. Under 50 ms per
   mutant, falsified above 200 ms. If it holds, ditto can classify a non-viable
   mutant without a subprocess; if it dies, the 25% subprocess measured there is
   the cheapest honest fix.
2. **H3, does the gate remove the `declared and not used` class** — same note.
   It rests on one observed mutant and has never been counted.
3. **Nested sites gated together.** `gated-gain.md` measured the nesting rule
   costing 44% of mutants on a threshold-heavy fixture and 7.9% on dharness. The
   two sites are the same expression tree and the multi-arm machinery already
   exists.
4. **Arithmetic and the statement families** are refused by design and keep the
   ordinary path. Nothing has measured what gating them would need.

## What must be decided, not measured

**The golden asserts both paths print the same bytes.** That is only sound while
no fixture holds a non-compiling mutant. On one that does — `internal/setup/writer.go`
is such a case — it would demand the gated path reproduce a bug. Fixing the false
kill at its source resolves this; until then the assertion is narrower than it
looks.

**`Gated()` is opt-in and was never turned on by default.** That was deliberate:
it is proven on two fixtures and a golden, not on a repository.

## The traps that cost the most time

- **`cd` before `tar`.** Three times a copy was made of the wrong repository
  because a command changed directory before archiving `.`. The identity of the
  fixture is now checked — `module github.com/Disble/dharness` — before anything
  is measured.
- **A control that fires and gets explained.** The wrong-repository run was
  caught by a control answering 0 where 35 were expected, and the first
  explanation reached for was a plausible story about environment variables. It
  spent the signal and left the feeling of having answered it.
- **A green that was never reached.** A golden extended to exercise a new path
  passed without touching it, twice over, for two different reasons. A `panic`
  dropped into the path is what found it.
- **Scripted substitutions that match nothing** look exactly like a change with
  no effect. `rg -c` on what was just written catches it, and did.
- **Two heredocs chained in one shell invocation** run their `git add` before the
  first commit. One commit per invocation.
- **`go test -o` compiles *and runs*.** A counter read double until `-c` was used.

## How to run any of it again

Every experiment note carries its own fixture and command. The two probes that
live in this repository are skipped unless pointed at a **throwaway copy**:

    DITTO_ADMIT_ROOT=<copy>      go test ./internal/schemata/ -run TestAdmits...
    DITTO_FALSEKILL_ROOT=<copy>  go test ./internal/schemata/ -run TestCounts...

`DITTO_FALSEKILL_ONLY` narrows the second to one file, and takes a value without
slashes because the shell rewrites anything that looks like a path.

The repository's own gate is `CPUS=1 mingw32-make lint test.failfast` on this
machine — `make` is not on PATH, only the native `mingw32-make`, and the MSYS2
build of make loses the environment its recipes need.

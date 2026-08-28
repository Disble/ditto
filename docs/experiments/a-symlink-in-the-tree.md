# A symlink in the tree

Written before the fix, so the predictions can be wrong.

## Where it came from

ditto's own mutation gate has been red on every push since 2026-08-27, and three
releases went out over it. Nobody read it. The panic:

    panic: failed scanning '/home/runner/work/ditto/ditto':
      failed reading '.../.devbox/nix/profile/default':
      read .../.devbox/nix/profile/default: is a directory

`.devbox/nix/profile/default` is a symlink to a directory. `filepath.WalkDir`
reports a symlink as a **non-directory entry** — it does not follow it — so
`materialize` takes it for a file and `os.ReadFile` returns `EISDIR`.

That is the same date the sandbox stopped being a reference. Under `link` the
walk called `os.Symlink` and never read anything, so any symlink worked by
accident. The copy default made reading mandatory, and every symlink became a
crash.

**This is not a devbox problem.** `node_modules/.bin`, a pnpm store, a vendored
checkout, a `docs -> ../shared/docs` in a monorepo: any of them, and ditto dies
before the first mutant.

## The question

What should a sandbox do with a symlink? Three answers, and they are not
equivalent:

- **recreate the link** — the sandbox holds a symlink with the same target
- **skip it** — the sandbox has nothing at that path
- **follow it** — copy what it points at, as real files

## Hypotheses

**H1 — the cause is symlink-to-directory, not devbox.** A tree holding one
`os.Symlink` to a directory reproduces `is a directory` under `copy` and
materialises fine under `link`.
*Kill line:* `copy` succeeds on that fixture. Then the cause is something in
devbox and this note is about the wrong thing.

**H2 — git already answered, and the two paths must agree.** `ditto staged`
builds its sandbox with `git checkout-index`, which materialises a tracked
symlink **as a symlink**. If the copy path follows or skips, a staged run and a
full run disagree about what the repository *is*.
*Kill line:* `checkout-index` writes a regular file holding the target's bytes.
Then following is what agreement requires, and recreating the link is wrong.

**H3 — recreating the link must preserve the RAW target, not an absolute one.**
An in-tree relative link re-created verbatim resolves inside the sandbox, so a
suite writing through it writes to the sandbox's own copy. Re-created as an
absolute path it resolves to the real repository — the exact write-through
`the-sandbox-is-a-reference.md` was written to close.
*Kill line:* the source repository comes back untouched under both. Then the
distinction does not matter and the simpler code wins.

## Control that has to move

ditto's own gate, on the runner that produced the panic. It is red now. If the
diagnosis is right it goes green, and if it stays red the fix is not the fix.

## Decision rule, fixed here

Recreate the link with its raw target if H1 and H2 hold and H3 shows the
absolute form writing through. Follow the link if H2 dies. Skip only if both
recreating forms write through — a sandbox that edits the repository it measures
is worse than one missing a path.

## Results

**H1 — corroborated.** A `t.TempDir()` holding one `os.Symlink` to a directory,
materialised with `NewWithStrategy(dir, "copy")`:

    panic: failed scanning '...': failed reading '.../dirlink':
      read .../dirlink: is a directory

Nothing about devbox. Any symlink to a directory, anywhere in the tree.

The same fixture turned up a **second failure nobody was looking for, and it is
the quiet one.** A symlink to a *file* did not crash: `os.ReadFile` follows it,
so the sandbox came back holding a regular file where the repository has a link.
No error, no message, a tree that is not the tree. The loud bug is the one that
got reported; this one had been shipping since 0.5.0 in silence.

**H2 — corroborated, and it answers the question.** A repository with a tracked
link to a directory and a tracked link to a file, staged and materialised the way
`ditto staged` does it:

    120000 1de5659 dirlink
    120000 0b97555 filelink
    100644 ce01362 target/file.txt

Mode `120000` is git's symlink mode. `checkout-index` wrote **the link's own
content — its target path — and never what the target holds**: `dirlink`
contained `target`, six bytes, not the directory. (On this Windows box
`core.symlinks` is `false`, so it landed as a regular file holding that text
rather than as a link; on a platform with symlinks git writes the link. Either
way it reproduces the link, never follows it.)

So git already decided, and the two sandboxes have to agree: **reproduce the
link.** Following it would also copy an unbounded tree — the panic's own path,
`.devbox/nix/profile/default`, resolves into the nix store.

**H3 — corroborated, and the control moved.** With the link recreated from
`os.Readlink`'s raw target, a suite writing through it in the sandbox left the
source repository holding `original`. With the same link recreated from the
absolute source path, the source came back holding `REWRITTEN BY THE SUITE` —
the write-through of `the-sandbox-is-a-reference.md`, reintroduced through one
path. Two of three tests go red when the raw target is swapped for the absolute
one, which is what makes them worth having.

## What was done

`filecopy.File` — the one function `fsrepository` and `staged` share — puts a
symlink somewhere as the same symlink, and copies only what is a file. Both
sandboxes get it: the full-repository walk, and the `.ditto.json` generated
paths, where `node_modules/.bin` makes this ordinary rather than exotic.

Three tests in `internal/fsrepository` and one in `internal/staged`. All four go
red with the branch disabled; that check was run rather than assumed.

## The part worth keeping

**ditto's own mutation gate had been red on every push since 2026-08-27, and
three releases went out over it.** The failure was not subtle — a panic, in the
first seconds, naming the file. Nobody opened it.

`0.6.0` shipped with a documented setting missing from three of five surfaces,
and the release that fixed that found this by reading a check it had been
walking past. The release skill's step 7 says to read the checks that run. It
now means every one of them, including the workflow that is not on the pull
request.

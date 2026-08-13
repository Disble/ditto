# CLAUDE.md

Project-specific guidance for Claude Code. `AGENTS.md` is the canonical
context — what ditto is for, the isolation rule, and the performance contract.
Read it first. This file names the two things that actually go wrong here.

## The one that damages something real

**Never run a mutation run, or the test suite, against a repository that holds
work.** Not this one, not the project consuming ditto, not any live checkout.
Build a throwaway project and point ditto at that.

The failure mode is silent. Git hands hooks an absolute `GIT_DIR` in a linked
worktree, spawned processes inherit it, and a fixture's `git init` /
`git config` / `git commit` lands in the real repository — successfully. It has
already left a stray commit on a live branch, set `core.bare=true` in a shared
config, and rewritten a repository's committer identity so later commits were
attributed to a test fixture. Nothing failed while it happened.

If you need to prove something is isolated, use a decoy repository and assert
its `HEAD`, commit count and config are untouched afterwards. `AGENTS.md` has
the recipe.

## The one that quietly undoes the whole point

**Performance is the metric, and the verdict comes from `perf/baseline.json`,
not from a stopwatch and not from a feeling.**

Run `go test ./internal/perfbench/` and read the exit code. The counters ratchet
both ways: a number that grows is a regression, and a number that shrinks fails
too until you write it down. Do not "fix" a failing counter by editing the
baseline to match — edit it only when you know which change moved it and why.

Wall clock never gates anything here. The machine is busy; that is normal.

Before claiming any change made something faster, say which counter moved and
by how much. If none did, it did not.

## Before proposing anything structural

The hot path is small and worth knowing by heart: `Ditto.Release` selects
sources and mutators, `GoSourceFile.Incubate` parses and produces infections,
and `Laboratory.Test` builds a temporary repository **per mutant** and runs the
test command in it. The two costs that dominate are that per-mutant repository
and the `go test` invocation itself — measured at 700–1100ms of fixed overhead
regardless of how fast the suite being mutated actually is.

[`docs/learning-log.md`](docs/learning-log.md) is the append-only why-log: one
dated line, newest at the bottom, never rewritten. Read it before optimising
anything — it holds the measurements that already redirected this project's
plan, including why the gate counts integers instead of milliseconds and why a
counter that looked correct reported 12 when the answer was 4. Append to it
when something non-obvious turns out to be true, and enforce the lesson with a
check whenever one exists.

[`docs/backlog.md`](docs/backlog.md) holds observations worth acting on later.
An entry whose evidence turns out to be a misreading goes away with the
misreading.

## Commands

    go build ./...
    go vet ./...
    go test ./...                      # must be green on Windows too
    go test ./internal/perfbench/      # the performance gate
    gofmt -l .

The Go floor is 1.25, and CI tests both that and `latest` — neither is exempt
from failing.

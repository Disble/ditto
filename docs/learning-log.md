# Learning Log

A human-readable record of _why_ things in ditto are the way they are:
non-obvious problems solved and measurements that changed a plan, so a later
session inherits the reasoning instead of rediscovering it.

This is a memory aid, **not** a substitute for a check that enforces the
lesson. Whenever one can be enforced — a test, a counter in
[`../perf/baseline.json`](../perf/baseline.json), `go vet` — do that too. This
file only explains the _why_; it never replaces the _how_.

## How to append

- One lesson per line, kept to a single sentence.
- Format: `- [YYYY-MM-DD]: text`
- Newest entries at the bottom. Never rewrite past entries; add a new line if
  something changes.

## Entries

- [2026-08-13]: A counter that measures one thing and is documented as measuring another holds only until the two diverge, which is exactly when it is needed — `sourceParsesPerRelease` counted visits to the `*ast.File` node and called them parses, true only while every parse implied one walk, and it silently kept reporting 12 after parsing was shared and the real answer became 4; counting *distinct* `*ast.File` pointers measures the thing itself, because a shared tree hands every mutator the same pointer and a re-parsed one cannot.
- [2026-08-13]: Git exports `GIT_DIR` and `GIT_INDEX_FILE` to hooks as **absolute** paths in a linked worktree and not at all (or relative) in a plain clone, so anything spawned from a hook in a worktree addresses the real repository instead of its own temporary one — and the misdirected command *succeeds*, which is why it left a stray commit on a live branch and set `core.bare=true` in a shared config without any test failing.
- [2026-08-13]: The dominant per-mutant cost is the fixed overhead of invoking `go test` (measured 700–1100 ms regardless of how fast the suite is), not compilation — recompiling the mutated file measured identically to not changing it at all, while the same tests run from a prebuilt binary took 36 ms, which redirected the optimisation plan from "avoid recompiles" to "avoid invocations".
- [2026-08-13]: Wall clock on a working development machine varied by more than fifty percent for the identical workload while mutant counts stayed exact, so the performance gate is built on integer counters and timings are reported for humans only.
- [2026-08-13]: A source file's relative path is its **identity** — `IgnoreSourceFiles` matches patterns against it and every report prints it — so leaving the host separator in made a pattern written with `/` match nothing on Windows and made the same run name files differently on different machines.
- [2026-08-13]: `T.Parallel` returns without recording anything when its parent has no `barrier`, which is the shape of any hand-built `*testing.T`, and supplying one makes it block forever sending on a nil `signal` channel — so whether a subtest was marked parallel is not observable from outside `testing`, and asserting it needs a seam in the code under test rather than `reflect` and `unsafe`.
- [2026-08-13]: What a run *says* had nothing ratcheting it — `perf/baseline.json` pins what a run costs, so a change could flip a mutant from killed to survived with every unit test still green; the golden release test pins the whole console output of a real run against a throwaway fixture with deliberately mixed verdicts, and it was proved to refuse before it was trusted, by weakening the fixture's own suite and watching it fail.

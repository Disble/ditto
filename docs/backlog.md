# Backlog

Things worth doing that were not worth derailing the change that found them.
Each entry records what was observed, not what was guessed. An entry whose
evidence turns out to be a misreading goes away with the misreading.

Nothing here is a promise.

---

## 1. A surviving mutant has no address

**Observed 2026-08-13**, while using the wrapper against a real change.

Ditto identifies each mutant by file and mutator, as a subtest name:

    --- PASS: TestStagedMutation/internal\exp\alpha.go_→_Comparison
    --- PASS: TestStagedMutation/internal\exp\alpha.go_→_Comparison#01
    --- PASS: TestStagedMutation/internal\exp\alpha.go_→_Integer_Decrement

There is no line number anywhere in that, and when one file carries several
mutants of the same kind they are separated only by `#01`, `#02`. So a report
of five survivors tells you which files to look at and nothing more precise.

What that costs, in the loop this tool is for: the author stops reading the
report and starts grepping the raw log to recover the diff of each survivor,
one at a time, to find out which line it landed on. The report becomes an
index into the log rather than an answer. Observed repeatedly in one session —
the phrase that ended it was "I'll stop guessing and pull the real diff of a
surviving mutant out of the log."

Ditto already holds everything needed. `goinfectedfile` carries the `token.FileSet`
and the AST, so the infection's position is one `fileSet.Position(node.Pos())`
away from a `file:line:col` that an editor can jump to.

What would close it: every reported mutant carries `file:line:col` and the
mutated expression, in the summary, not only inside a rendered diff. A survivor
should be clickable.

Worth pairing with: survivors are the only part of the output anybody acts on,
and they are currently printed last, after the diffs. In a TDD loop the first
thing on screen should be the list of addresses.

## 2. Nothing reclaims a sandbox left by a killed process

**Measured 2026-08-13.** Six abandoned `ooze-*` sandboxes were found in one
machine's temp directory, the oldest eight days old. One had the expected shape
— 81 symlinks and a single real file — and two held 2.4M and 3.6M. All three
contained links into a real repository's `.git`.

Sandbox lifetime is now owned by the run: one parent directory per process,
removed through `t.Cleanup` in `Release`, with retries because Windows holds a
handle for a moment after the process that worked in the directory exits. That
covers a normal finish, a failed run and a `t.Fatal`. It cannot cover the
process being killed — `go test` timing out, Ctrl-C, a kill — because no Go
code runs then. Ctrl-C is an accepted risk; what is missing is reclaiming what
it leaves.

The prerequisite is in place: the parent directory carries the owning process
id (`ditto-<pid>-<random>`), so an abandoned one is distinguishable from one
still in use.

What would close it: at the start of a release, remove `ditto-*` parents whose
process id is no longer alive. Keyed on liveness rather than age, because a
concurrent run is the thing that must never be swept — and a long run could
outlive any age threshold worth using. Process id reuse makes the check
occasionally conservative, which is the right direction to fail in: it leaves
garbage rather than deleting a live run's sandbox.

## 3. Ditto's own mutation test runs against a repository that holds work

`ditto_mutation_test.go` calls `Release` with `WithRepositoryRoot(".")`, which
is precisely what `AGENTS.md` says not to do. It is safe where it was designed
to run — a CI checkout holds no work, and the mutation happens in a sandbox
rather than in the source — but the guidance and the code disagree, and the
build tag is the only thing keeping a local `make test.mutation` from being the
exception nobody remembered.

What would close it: point the self-check at a fixture project built for the
run, or state in the file itself that it is CI-only and why.

## 4. The logo still belongs to upstream

`.assets/logo.svg` is ooze's. The licence permits reuse, but the mark is the
one thing a fork should not inherit — it is upstream's identity, not its code.
Needs replacing before ditto is presented anywhere as its own project.

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

## 4. Linters turned off to get CI running again, not because they were wrong

**Recorded 2026-08-13**, while making CI run for the first time.

Raising the Go floor to 1.25 broke `make lint`: golangci-lint 1.61.0 is built
with go1.23 and refuses a config targeting a newer language version. Upgrading
it meant migrating `.golangci.yml` from the v1 format to v2, and the newer
linter enables checks that did not exist when this code was written. That
surfaced 58 findings, roughly three quarters of them in inherited code.

Seven linters are disabled in `.golangci.yml` to keep the check meaningful
rather than deafening: `exhaustruct`, `funcorder`, `goconst`, `lll`,
`noinlineerr`, `prealloc`, `varnamelen`. Each flags style on code that predates
it. None of them was judged and rejected; they were postponed.

One finding was a real defect and is fixed: `nilnesserr` caught
`FSTemporaryRepository.Overwrite` folding two checks into one condition, so the
branch could only be entered while `Stat`'s error was nil and the panic wrapped
that nil — the removal error that actually mattered was discarded. It is on the
path every mutant now takes.

Two more are annotated at the site rather than disabled, because the reason is
local: `gosec` objects to walking a repository and mirroring it with symlinks,
which is what this tool does, and `noctx` wants `exec.CommandContext` for the
configured test command, which has no cancellation contract today. Giving it
one is an API decision.

`golangci-lint` is pinned to 2.12.2, which is what `latest` resolved to on the
first green run and what the maintainer runs locally. Pinned rather than
floating because an unpinned linter means somebody else's release can turn this
build red.

## 5. The logo still belongs to upstream

`.assets/logo.svg` is ooze's. The licence permits reuse, but the mark is the
one thing a fork should not inherit — it is upstream's identity, not its code.
Needs replacing before ditto is presented anywhere as its own project.

## 6. A mutant that cannot compile is scored as killed

Ditto recognises a killed mutant by the test command exiting non-zero, and a
mutation that does not compile makes it exit non-zero too. The mutant is counted
as caught by a test that never ran.

Measured over `internal/` of dharness: **94 of 1293 mutants, 7.3%**, from four
ordinary causes — an integer literal decremented into a negative index, a
comparison replaced by a constant leaving its variable unread, arithmetic
subtracting strings, and switch cases colliding after a literal moved. It is a
lower bound: the probe builds the package, so a mutation that breaks only the
test build is one it misses.

Every such mutant inflates the score, which is the number people act on and the
number dharness's gate fails builds on.

`go test`'s exit status cannot tell the two apart — it is 1 for both — and
reading `[build failed]` out of the output would tie ditto to one tool's
phrasing. Building before running costs a median of 25%. Asking the same question
in process with `go/types` is unmeasured and is the next thing to try.

See `docs/experiments/false-kills.md` and `docs/experiments/fixing-false-kills.md`.

## 7. The gated path is opt-in and unproven at scale

`ditto.Gated()` compiles a file's mutants once and selects each at run time. It
reaches the same verdicts as the ordinary path on every fixture measured except
one, where it disagreed and was right — the ordinary path was scoring a
non-compiling mutant as killed.

It is proven on two fixtures and a golden, not on a repository. It is not on by
default, and the golden's assertion that both paths print the same bytes is only
sound while no fixture holds a non-compiling mutant.

See `docs/experiments/state-of-the-work.md` for everything measured and everything
still open.

## 8. Does the false kill ever change a verdict?

Removing a false kill lowers numerator and denominator by the same amount, so a
corrected score is always lower whenever anything survived. How much, and whether
it crosses a line anyone acts on, is not determined by that.

dharness gates at **0.80** over its staged scope (`tools/mutationstaged/main.go`).
The question is whether any real run crosses it once the 94 are gone — because if
none does, the inflation is real and has never changed an outcome, and that is
worth knowing before ditto's public report grows a category.

Split out of `docs/experiments/refusing-false-kills.md`: it answers a different
question from the one that note asks, so it needs its own.

## 9. Is refusing a mutant the same as dropping it after generation?

On the ordinary path the two probably coincide. On the gated path they may not:
every mutant of a file shares one compilation, so one that does not build fails
the build for all of them and the file falls back.

If they differ, the counting rule for the report is not a convention to be
chosen — it is a consequence to be read off. Also its own question.

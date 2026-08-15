# Backlog

Things worth doing that were not worth derailing the change that found them.
Each entry records what was observed, not what was guessed. An entry whose
evidence turns out to be a misreading goes away with the misreading.

Nothing here is a promise.

---

## 1. A surviving mutant has no address — **closed 2026-08-14**

Closed rather than deleted, because entries 6 and 7 point at their neighbours by
number and renumbering would silently redirect them.

Every mutant now carries `path:line:col` and the text it replaced, read off the
one `schemata.Difference` the tool already computes — no virus was touched and
`viruses.NewInfection` kept its signature. Survivors are listed by address before
any diff is rendered.

The number that justified it, measured over 135 mutants of four gofmt'd files:
**129 of them could not be told apart from another mutant by what ditto
printed**. The mechanism was chosen by measurement rather than by preference, and
the first round's three hypotheses were all refuted before the second round found
the answer — `docs/experiments/mutant-address.md`.

`TestSurvivorsAreDistinguishable` is the guard, and it was watched refusing
before it was trusted.

What remains open, and deliberately not done here: the survivor list is printed
in the order the viruses produced the mutants, not in source order, so that it
pairs one-to-one with the diffs below it. Whether a reader would rather have it
sorted by address is a question about the loop, not about the address.

The original observation follows.

---

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
tight bound, since measured: compiling the test binary over the same 1293 mutants
rejects the same 94 and not one more, because no virus rewrites a declaration.

Every such mutant inflates the score, which is the number people act on and the
number dharness's gate fails builds on.

`go test`'s exit status cannot tell the two apart — it is 1 for both — and
reading `[build failed]` out of the output would tie ditto to one tool's
phrasing. Building before running costs a median of 25%, which is the incumbent
and the bar.

Two cheaper mechanisms were measured and both are refuted. Syntax alone reaches
61 of the 94 and wrongly refuses 2 mutants that compile, which disqualifies it
whatever its coverage. An in-process `go/types` check reaches 92, refuses nothing
that compiles, and needs no process per mutant — it misses one package it cannot
read at all. See entry 10.

See `docs/experiments/false-kills.md`, `docs/experiments/fixing-false-kills.md`
and `docs/experiments/refusing-false-kills.md`.

## 7. The gated path is opt-in and unproven at scale

`ditto.Gated()` compiles a file's mutants once and selects each at run time. It
reaches the same verdicts as the ordinary path on every fixture measured except
one, where it disagreed and was right — the ordinary path was scoring a
non-compiling mutant as killed.

Measured since: on a fixture built to contain that class, exactly one label
differs between the paths and it is the mutant that leaves a local unread —
survived gated, killed ordinary. `docs/experiments/disagreement-class.md`. The
credit is real and its width is that class.

Also measured since: the branch changed no verdict the ordinary path used to
give, on 31 mutants of a fixture and 35 of a real file
(`docs/experiments/backward-compatibility.md`), and the gain is unchanged at 135
invocations against 38 (`docs/experiments/gain-after-the-fixes.md`).

What still blocks a default is entry 11, not the verdicts.

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

## 10. A type checker that honours build constraints

`refusing-false-kills.md` measured an in-process `go/types` check reaching 92 of
the 94 mutants that cannot compile, refusing nothing that compiles, at one
`go list -export -deps` per package and no process per mutant. It is refuted as
stated — the population is 94 — and the two it missed are one package,
`internal/runner`, which pairs `busy_windows.go` with `busy_other.go`.

The checker reads every `.go` file in a directory and ignores build constraints,
so that package already fails to check before any mutation, and the control
requiring silence on unmutated code refuses it. `go/build.Context.MatchFile` is
the obvious answer and nothing has measured what honouring it costs, nor whether
the remaining two then fall.

This is a conjecture born from that note's data, so it needs its own question and
its own note rather than an amendment to that one.

## 11. `go test -v` silently turns the gated path off — **closed 2026-08-14**

Closed in place, for the same reason entry 1 is: entries point at their
neighbours by number.

Both halves are done. `verboselaboratory` forwards `TestAll` when the laboratory
below can take a batch, and `gatedreporter` prints how much of the run came from
one compilation — zero spelled `none`, because `Gated: 0 of 7` scans as a
formatting placeholder and this line exists to be noticed when it says zero.

Measured on the golden fixture, eight runs, one variable
(`docs/experiments/forwarding-the-batch.md`): **none of 7 gated under `-v` before,
4 of 7 after, against a control of 4 of 7 without `-v` on both sides.** No verdict
moved, and the three survivors' addresses are identical character for character
between the gated `-v` run and the ungated one — a comparison only entry 1 made
possible.

The golden now runs the fixture under `-v` and refuses if the gate count differs
from the quiet run. It was watched refusing: with the forward removed it fails
with `the gated release gated 0 mutants under -v and 4 without it`. That is the
guard the entry asked for — the defect had cost three measurements and was found
by a hand-placed `panic`, because nothing in the suite could see it.

The original observation follows.

---

`release.go` reads `testing.Verbose()` and wraps the gated laboratory in
`verboselaboratory`. That decorator implements `Test` and **not** `TestAll`, so
`ditto.Release`'s type assertion to `BatchLaboratory` fails, the batch is never
forwarded, and every mutant takes the ordinary path one at a time.

`ditto.Gated()` is therefore a no-op under `-v`, which is what CI usually runs.
Nothing reports it: `Gated()` and `FellBack()` are counters the report never
prints, so a run that gated nothing looks exactly like a run that gated
everything.

Found by putting a `panic` inside `TestAll` and watching a gated run finish
without it. A second `panic` in `Test` did fire, which separated "the laboratory
is not in the stack" from "the batch is not forwarded".

It cost three measurements. `changed-scope.md` and `instrumentation-fidelity.md`
both reported a hypothesis holding on numbers produced by the ordinary path
compared with itself, and both are corrected in place.

What would close it: `verboselaboratory` forwards `TestAll` the way
`testingtlaboratory` already does, and the report prints how many mutants were
gated, so a path that silently did not engage is visible in the output rather
than only in a panic someone thought to add.

## 12. A decorator that drops a capability is refused by nothing

`verboselaboratory` dropping `TestAll` was entry 11. The same shape is still in
the tree, one interface over: `VerboseTemporaryDir` implements `New` and not
`RemoveAll`, so wrapping the temporary directory in it hides the method that
reclaims a run's sandboxes.

Nothing goes wrong today, and only because of where one line sits.
`reclaimSandboxes` is called in `Release` **before** the verbose block wraps
`opts.TemporaryDir`, so the type assertion sees the unwrapped `FSTemporaryDir`.
Move that call twelve lines down and every verbose run leaks its sandboxes with
no test failing — which is exactly how entry 11 survived, and entry 2 is about
the sandboxes that get left behind.

The comment above the call says so, and a sentence is not a guard.

What would close it: either every decorator forwards the optional interfaces it
sits in front of, or something refuses a stack in which a capability present at
the bottom cannot be reached from the top. The second is the general answer and
it covers decorators nobody has written yet.

**Not measured**, and named so it is not mistaken for cleared: whether
`verboserepository` and `verbosetestrunner` have the same gap. Neither was
examined.

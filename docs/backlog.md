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

**It recurred on 2026-08-27, and the pin was not the problem.** With Go 1.27.0
installed and the floor still at 1.25, every commit was refused:

    panic: file requires newer Go version go1.27 (application built with go1.26)

Nothing in the repository asked for 1.27 — what golangci-lint could not
type-check was the standard library shipped with the installed toolchain.
Measured as pre-existing: on a clean `main` with zero changes, `make lint`
panicked the same way, so no change had caused it.

The version was not what needed moving. `go install …/golangci-lint@v2.12.2`
under the local Go rebuilds **the same pin** with the current toolchain, and
lint went to `0 issues` without enabling one new check. A bump to 2.13.1 would
have brought the checks entry 4 is about; the toolchain mismatch would not have
needed it.

**That open question was answered the same day, and the answer was yes.** The
note above predicted that the nixpkgs golangci-lint would carry its own build
toolchain and that CI would hit this where a local rebuild could not help. It
did, on both legs of the matrix:

    can't load config: the Go language version (go1.26) used to build
    golangci-lint is lower than the targeted Go version (1.27)

So the linter stopped being a package and became a build. `makefile` pins
`golangci-lint-version` and installs that exact version into `.bin` with the
repository's own toolchain, and `PATH` already puts `.bin` first — so local and
CI now run a binary built with the Go they are targeting. `devbox.json` no longer
names it, which is what leaves one pin instead of two.

**And then the version had to move anyway, which the sentence above got wrong.**
It said a stale build and a wrong version were different problems and only the
first was here. Rebuilt 2.12.2 got past the config check and CI failed again, on
both legs, somewhere else entirely:

    buildir: package "poll": unexpected expr: *ast.KeyValueExpr

That is `honnef.co/go/tools@v0.7.0`, vendored inside 2.12.2, unable to build IR
for Go 1.27's `internal/poll`. No build of 2.12.2 can analyse a 1.27 standard
library, so the version was genuinely wrong and not merely stale.

**Local green did not catch it, and the reason is worth keeping.** The same
rebuilt 2.12.2 reported 0 issues on Windows. Linux and Windows have different
`internal/poll` sources, and only the Linux one carries the syntax that panics.
A local gate that agrees with itself proved nothing about the platform CI runs
on — the same lesson as the false perfect score, one layer down.

So the pin moved to **2.13.1**, and it cost exactly what this entry predicts a
linter upgrade costs: `exhaustruct` was renamed to `exhaustruct_v5`, the old
name in `.golangci.yml` stopped matching anything, and **58 findings came back**
on code nobody had touched — the same shape and very nearly the same count as
the v1-to-v2 migration above. Both names are now listed. `gomodguard` is
deprecated in favour of `gomodguard_v2` and is the next one to do this.

Also unmeasured and worth knowing: `devbox.lock` is stale. It pins `go@1.22.1`
and `golangci-lint@1.61.0` while `devbox.json` declares 1.27.0 and 2.12.2, so
devbox has been re-resolving from the manifest and the lock records a state
nothing runs.

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

## 13. `go:embed` cannot work in the sandbox — **closed 2026-08-27**

Closed the same day it was opened, because routing around it per repository was
the wrong shape and saying so was the whole point.

A sandbox is now materialised with **hard links, falling back to copies**. A hard
link is a second name for the same bytes: it is a regular file, which is what
`go:embed` requires and what a symlink is not, and it writes nothing, which is
what 1960 files per sandbox demands. It is safe only because `Overwrite` removes
a path before writing a mutant — removing a name leaves the original alone, where
a write in place would have corrupted the repository being measured.

Measured, three rounds with the order rotated, on a 1960-file repository:
hard links **19.1-19.7s** against symlinks **19.8-22.6s**, and copies
**34.2-39.4s**. Copying was the obvious answer and is 70% slower, which is why it
is the fallback for a filesystem boundary rather than the rule.
`filesLinkedPerSandbox` did not move, and that is the finding: the strategy that
makes a sandbox usable costs what the one that did not cost.

The fixture that failed now reports **9 mutants, 8 killed, 1 survived, 0.89** —
against the 9 of 9 and a perfect 1.00 that a symlinked sandbox and a guardless
ooze had been reporting for a package that never compiled. There is a real
survivor in it that nobody could see.

`docs/experiments/the-sandbox-is-a-reference.md`. What is still open there: no
behaviour outside `embed` was shown to differ, so "it is a class" is recorded as
undecidable rather than established; and hard links were measured on Windows only.

The original observation follows.

---

## 13 (original). `go:embed` cannot work in the sandbox

**Measured 2026-08-27**, while checking whether `ditto staged` could replace a
wrapper in a repository that uses `//go:embed`.

A release mirrors the repository as one symlink per file
(`fsrepository.LinkAllToTemporaryRepository`). Go refuses to embed a symlink, so
any package with an `embed` directive fails to build in the sandbox:

    internal/tray/icon.go:8: pattern tray-icon.ico: cannot embed irregular file
    main.go:11: pattern all:frontend/dist: contains no embeddable files
    FAIL [setup failed]

This is not about one repository's frontend. It is every `go:embed` in every
package a release touches, and there is no way for a caller to work around it:
the sandbox is built inside the run.

**What it cost before it was found, and this is the part worth keeping.** The
repository in question had been running mutation testing over that package with
ooze, which has no baseline guard, and reading a failing command as a killed
mutant. It reported **9 mutants, 9 killed, a score of 1.00** for a package that
never compiled. Each mutant "died" in 0.92–2.32 s where one real run of the suite
takes 4.95 s — the same signature, and the same cause, as the 431 of 431 in
5.46 s in `docs/experiments/false-perfect-score.md`. Ditto refuses it now, which
is how it was noticed at all.

Also measured: with the guard refusing, the message named a red baseline and
nothing else, and finding the embed took four rounds of elimination. The refusal
now carries the test command's own output, which named it on the first run. That
half is closed.

What would close the rest: materialise the sandbox as copies rather than links,
or copy the files an `embed` directive names. The first is simple and its cost is
unmeasured — `filesLinkedPerSandbox` is 6 on the synthetic fixture and 164 on one
real checkout, at ~0.45 ms per link, and a copy is not a link. The second is
cheaper and needs the directives parsed, which is more code than it sounds.

Until then, a repository that embeds anything can mutate only the packages that
do not. `docs/experiments/the-embedded-frontend.md`.

## 14. A verdict does not carry its reason

`internal/cmdtestrunner/cmdtestrunner.go` reads any non-zero exit as a killed
mutant: a failed assertion, a mutant that does not compile, a timeout, a missing
binary. `internal/gobuildrunner/gobuildrunner.go` does the same on the gated
path. Two categories where PIT has ten and Stryker eight.

Measured on `internal/schemata/instrument.go`: 78 mutants, 50 reported killed, of
which **10 did not compile and 1 hung until its timeout** — 22% of the kills were
not assertions.

Both missing reasons are available cheaply:

- **build failure**: `go test -json` emits `"Action":"build-fail"` and a `fail`
  event carrying `"FailedBuild"`, and neither appears when a test fails for real.
  No extra subprocess. Only works when the test command *is* `go test`; a user
  who configures `make` or `gotestsum` gets no such signal, and the metric has to
  say so rather than report zero.
- **deadline**: ditto's own clock — see entry 15.

`docs/metrics.md` metric 2. `docs/experiments/what-the-field-already-decided.md`.

## 15. Ditto imposes no deadline anywhere

`grep -rn "timeout" --include=*.go .` finds one hit in the whole product, and it
is a doc comment in `options.go`.

`gobuildrunner.go` runs the compiled binary directly, and a test binary invoked
that way takes `-test.timeout` **0 — disabled**; only the `go test` driver
injects the 10-minute default. So on the gated path **a mutant that loops never
returns**, and `loopcondition`, `loopbreak` and `rangebreak` are all in the
default virus set.

On the ordinary path the bound is whatever the user's command carries: ditto's
default `go test` gives 10 minutes, and ditto's own gate gives 5 through
`make test.failfast`. Measured: one mutant of `internal/schemata/instrument.go`
deadlocked at 0% CPU for the full 300s, a sixth of a thirty-minute budget.

`cmdtestrunner.go` already records the omission: *"noctx wants CommandContext.
The configured test command has no cancellation contract today."*

A deadline is a prerequisite for metric 2, not a metric itself: once it exists, a
mutant that hits it is a **kill with its own reason**, which is what PIT, Stryker
and Infection all do.

**Closed 2026-08-28**, and one part of the fix is worth keeping: `CommandContext`
alone does not bound anything. It kills the process it started, and `go test`
starts the test binary — killing the parent leaves the child holding the pipe, so
`CombinedOutput` waits forever on a command that was already cancelled. Measured:
the deadline test ran to its own 30s safety net on Linux while Windows happened
to return. `WaitDelay` is what closes it. The hang this deadline exists to stop,
living inside the fix for it.

## 16. The two paths disagree and nothing counts by how much

`fidelity_probe_test.go` asserts the gated and ordinary paths still disagree
about the same mutant, and `docs/experiments/disagreement-class.md` has the class
measured. Two implementations, two answers, at most one right.

It is named in `docs/metrics.md` as the fifth metric and deliberately not gated:
**its population is unmeasured**, because the probe sits behind `DITTO_PROBE=1`
and nobody has run it against a real repository. A threshold set before that
number exists would be invented.

## 17. The gated path cannot express every mutant, and nothing counts the gap

Measured by others, not here. A mutant switched at run time has to be a branch
both arms of which compile and type-check, so a whole class cannot be expressed:
compile-time constant contexts, static initialisers, type-incompatible operands.
Stryker.NET skips constants for exactly this reason -- *"MutantControl.IsActive(0)
is not a constant value. That is why we skip constant values from mutating"* --
and Stryker excludes static mutants from the score entirely.

**Schemata is therefore neither a superset nor a subset of the ordinary path.**
That gap will never be zero, so it belongs in its own non-fatal metric rather
than inside the agreement one, which is gated at zero.

## 18. A sanity check the field has and ditto does not

Dextool runs the test suite against the instrumented binary with **no mutant
active** and requires it to pass, before using the schema at all. A schema that
fails there is blacklisted.

Ditto's `instrumentation-fidelity.md` H1 asserts the same property, but only
inside a probe behind `DITTO_PROBE=1`. It is the check that catches the worst
failure mode of instrumentation -- the suite behaving differently under the
schema with nothing mutated, which INFLATES the score rather than breaking it.

## 19. Gating is measured and not turned on

`docs/experiments/turning-gating-on.md` measured it: 60.1% of this repository's
729 mutants are gateable, and gating removes **394 of 729 compilations, 54%**.
The two paths agreed on every mutant of the scope tested.

It is not turned on, and the reason is written rather than hidden. **The scope
that agreed held no survivor** — ten mutants, all killed on both paths — and a
set with nothing surviving in it cannot show a disagreement about survival. The
run that would settle it is one scope with survivors in it; `instrument.go` alone
has 28, and that run costs about forty minutes.

Also unanswered, and separately: **whether the gate then fits in thirty minutes.**
Gating removes compilations and removes no invocations at all. That is a wall
clock question, this repository reports wall clock rather than gating on it, and
no integer measured here answers it.

## 20. The generator makes 11.1% garbage, and the cheap refusal is measured

Measured on ditto itself for the first time on 2026-08-29 — the harness had been
broken since the report gained line and column, so `docs/experiments/false-kills.md`
records the fix as well as the number:

    mutants 748, of which do not build 83 (11.1%)
    H2 go/types : caught 83 of 83, wrongly refused 0

Benchmark: **Major 1.8%, PIT 0%.** Six times worse than the state of the art, from
four operators of fourteen.

It no longer inflates the score — `internal/verdict` names them and the reporter
excludes them from both sides — so this is no longer an accuracy defect. It is
**11.1% of every release run and discarded**, and a generator that should not
have produced them.

The refusal is `internal/schemata/typecheck_test.go`, which is test-only. Making
it product code costs one `go list` per package, against 748 test-command
invocations. That ratio is why this is worth doing and why nothing cheaper is
needed: the AST-only alternative catches 42 of 83 and **wrongly refuses one that
compiles**, which is the wrong direction to be wrong in.

## 21. The gate asks a repository-sized question on every push

Measured three times, all to the same end: with gating on and `make` resolved,
the gate reaches **424 of 727 mutants** and dies at `-timeout=30m`. Two levers
are spent.

- **Gating** removes 54% of compilations and the run still does not finish.
- **Cutting the suite** the mutant is judged by, from 69.3s to 37.2s, changed the
  gate by 424 against 422. `-failfast` already stops a killed mutant at its first
  failing test, so the expensive tests are only paid by SURVIVORS, and they are
  the minority. A 46% cut in the suite bought 0.5% in the gate, and the change
  was reverted.

What is left is not a lever. **Ditto's own answer to a repository-sized bill is
`ditto staged`** — mutate what the change touched. `WithChangedRanges` is
recorded at 4 laboratory runs where the whole fixture charges 48, dharness uses
the staged path, and this repository's own gate does not.

Raising the timeout is not on this list. The number would move and the question
would not.


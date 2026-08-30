# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.9.0] - 2026-08-30

The release that makes ditto's own gate finish, by making it ask a question it
can afford.

### Added

- **`ditto changed --since <ref>`**, and `PlanChanged` / `RunChanged` behind it.
  It is `staged` asked of a committed range instead of the index: the scope is
  `<ref>...HEAD`, the diff against their merge base, so a base that has moved on
  does not drag somebody else's commits into the bill. The diff parsing, the byte
  offsets, the fail-open rule and the sandbox are the staged path's, unchanged.

  It exists because `ditto staged` cannot be a CI gate. A CI checkout has nothing
  staged — the change is already committed — so a gate pointed at the staged
  scope skips, reports success, and measures nothing.

  It refuses a checkout with uncommitted work in it. A range scope names bytes of
  `HEAD` while the sandbox is written from the index, and those are the same tree
  only while nothing is modified or staged; scoping against one and mutating the
  other is the defect already measured at seven of eight verdicts moving.

  There is no default base, and there will not be one: on a CI checkout the
  useful base is the last release, on a branch it is the trunk, and a base
  guessed wrong is either a bill nobody asked for or a scope of nothing reported
  as green.

### Changed

- **ditto's own CI gate mutates the change rather than the repository.** The
  repository-sized run asks for 783 mutants and dies at its thirty minutes having
  reached about 424, measured four times. Both levers were already spent: gating
  removes 54% of the compilations and does not close it, and cutting the mutant's
  suite by 46% moved the gate by 0.5%, because `-failfast` already stops a killed
  mutant at its first failing test. The bill was the wrong size rather than badly
  paid — backlog entry 21, open since the measurement and now closed.

  `make test.mutation` is untouched and `workflow_dispatch` still reaches it. The
  repository-sized question is worth asking on purpose; it was being asked on
  every push, against a clock it could not beat.

## [0.8.0] - 2026-08-30

A release about what a run SAYS. Every item came from one exchange with a
consumer repository whose `ditto staged` run printed nothing for over ten
minutes, twice, and was reported as a hang. It was not stuck. Nothing it did was
wrong; nothing it did was visible either, and that turned out to be the defect.

### Added

- **A progress line per mutant.** A release names each file and its mutant count
  before running any of them, and names each mutant before handing it over —
  before, not after, because a line printed once a mutant has finished cannot say
  which mutant a stall is inside.

  This is the whole of the reported problem. ditto printed nothing at all between
  `Releasing Ditto…` and the report, so a run advancing normally and a run that
  was genuinely stuck produced identical bytes. "No mutants appeared after ten
  minutes" was evidence of nothing, which is why it was misread as a setup stall
  and then misdiagnosed a second time as a baseline that had hit its deadline.
  The reporter's own measurement — a 27.4s suite — already refuted that, and
  nobody multiplied it by the mutant count because neither number was on screen.

- **The baseline says what the suite cost.** The laboratory already ran the suite
  once on unmutated code and threw the duration away. It is the per-mutant price
  of the test command, and printed beside the mutant count it is the whole bill.

- **`ditto version`**, which also answers `--version` and `-version`. The number
  comes from `runtime/debug` rather than a constant, so there is no second place
  for it to go stale, and a binary built from a checkout says it has no released
  version instead of naming one.

  It is not cosmetic: whether a mutant that never compiled is scored as killed
  differs between 0.6.0 and 0.7.0, and until now nothing a user could type said
  which build they had.

- **`--gated` on `ditto staged`.** `run` always had it. Nothing about one
  compilation per file is specific to a whole-repository release, and the staged
  path mutates fewer files, which is where a compilation is easiest to repay.

- **`--confirm-kills`**, opt-in on both subcommands. The baseline check is a
  `sync.Once`: it refuses a suite that is ALREADY red, and cannot see one that
  goes red at mutant 37, where a spurious failure becomes a kill no test earned
  and nothing in the report tells it from a real one.

  Only assertion kills are re-run — a mutant that never built already leaves the
  score on both sides, and a deadline is a clock ditto fired itself. A survivor
  is never re-run either: a flake manufactures failures, and cannot make a mutant
  the tests caught look like it escaped. Off by default, because it doubles the
  price of every assertion kill and buys nothing on a suite that does not flake.

### Fixed

- **`--test-command` rendered as `-test-command -json` in `-h`**, which reads
  like a second flag. `flag.PrintDefaults` takes the first backquoted word in a
  usage string as the flag's VALUE NAME. Found by running the binary; no
  assertion on the help constant could have seen it, and the new test renders the
  flag set instead.

### Changed

- **`--test-command`'s description names its cost.** It said what the flag does
  and what its default is, and a reader who understood every word still ran
  `./...` against a 27-second suite — once per mutant, sequentially. It now says
  that, and says what dropping `-json` costs rather than only what it does.

  The readme carries the same warning with the measured delta, because two
  separate repositories wrote that sentence into their own docs rather than
  getting it from here.

## [0.7.0] - 2026-08-29

A release about ditto's own answers: whether they can be believed, and whether
the gate that produces them ever finishes. Both were measured rather than
assumed, and both were wrong.

### Breaking

- **The mutation score changes value for the same code.** A mutant that never
  compiled used to be counted as killed — a failing compile exits non-zero, and
  that is how ditto recognises a kill — and it now leaves the numerator **and**
  the denominator. Measured on a fixture with two non-viable mutants of five, the
  reported score went from **0.60 to 0.33**: one earned kill of one viable
  mutant, which is what the tests actually did.

  Anyone with a `WithMinimumThreshold` tuned to the old number will see it move,
  and it moves toward the truth. The kill predicate is undefined for a program
  that does not exist: Zhu, Hall & May, *ACM Computing Surveys* 29(4) 1997,
  Def. 3.1 is `S = D / (M − E)`, and gremlins, cargo-mutants, Stryker and
  go-mutesting all exclude it.

  Measured on this repository: **83 of 748 mutants do not build, 11.1%**, against
  a benchmark of Major 1.8% and PIT 0%.

- **The default test command gains `-json`**, in both places it is defined, so
  the captured output of a failing command is now the `go test` event stream. It
  is what lets ditto say WHY a mutant died; `verdict.Text` renders it back for
  anything a human reads.

### Added

- **`internal/verdict` — a verdict carries why it happened.** Ditto read any
  non-zero exit as a killed mutant: a failed assertion, a mutant that never
  compiled, a timeout, a missing binary. Measured on one file, 78 mutants: 50
  reported killed, of which **10 did not compile and 1 hung until its timeout**.
  22% of the kills were not assertions.

  The reason comes from `go test -json`, which emits an `Action: build-fail`
  event and a `fail` event carrying `FailedBuild` — neither of which appears when
  a test fails for real. **No extra subprocess.** Exit codes cannot carry it:
  measured on go1.27, `go test`, `go build`, `go vet` and `go test -json` all
  exit 1 either way, and a test pins that so a Go release which changes it is
  caught here.

  A command that is not `go test -json` yields `Unknown` rather than a guess.

- **A deadline, at last.** Searching the whole product for a timeout used to find
  one hit, and it was a doc comment. A test binary invoked directly takes
  `-test.timeout` **0 — disabled** — so on the gated path a mutant that loops
  never returned, and `loopcondition`, `loopbreak` and `rangebreak` are all in
  the default virus set.

  Both paths are bounded now, and a mutant the deadline stops is a **kill
  carrying its own reason**, which is what PIT, Stryker and Infection all do.

- **The report names what nobody's test earned**, with the operator behind it,
  because the rate has an external benchmark and a rate alone names no work:

      ┃ • Killed:       1
      ┃ ✓ Score:     0.33 (minimum: 0.00)
      ┃ 2 of the 5 mutants generated never compiled, and are out of the score entirely.
      ┃   1 from Integer Decrement
      ┃   1 from Integer Increment

- **`make test.mutation.staged` — a gate that measures the change and finishes.**
  The repository-sized gate reaches **424 of 727 mutants** and dies at thirty
  minutes, measured three times. The same gate over one staged file: **4 mutants,
  82 seconds, green.** Not a smaller timeout — a smaller question, and the one
  this tool was built to ask.

  It reads the index rather than the worktree, which is the whole point: one
  tracked file left dirty and unstaged moved **seven of eight verdicts**.

- **`mutantsPerReleaseOnThisRepository`** in `perf/baseline.json`, behind a
  `livetree` build tag. Every other recorded counter measures a six-file fixture,
  and all eight stayed green while the real cost went from 431 mutants to 727.

  The tag is not packaging: untagged, the counter read the tree it was compiled
  inside, and ditto's sandbox carries every test file while each mutant runs the
  whole package list — so it ran inside every mutant's sandbox and answered for
  the mutant instead of measuring it. A census puts **66 of 727** mutants in
  reach of that.

- **A staged run announces a widened scope.** When the diff cannot be turned into
  byte ranges the plan falls back to whole files — and widens *every* file, not
  only the one it could not read. `Derived` and `Reason` had been on the plan
  since the beginning and only `--dry` ever printed them.

### Fixed

- **A symlink anywhere in the tree broke the sandbox, two different ways.**
  `filepath.WalkDir` does not follow links, so one arrives as a non-directory
  entry and was read as a file. A link to a **directory** returned `EISDIR` and
  killed the whole walk before the first mutant — any repository with a
  `node_modules/.bin`, a pnpm store or a vendored checkout. A link to a **file**
  was worse for being quiet: the sandbox came back holding a regular file where
  the repository has a link.

  Both sandboxes reproduce the link with its **raw** target now. Rewritten to an
  absolute path it resolves back to the repository under measurement, and a suite
  writing through it edits the tree ditto was asked to leave alone — measured,
  the source came back rewritten by the suite.

- **`CommandContext` alone bounds nothing.** It kills the process it started, and
  `go test` starts the test binary: killing the parent leaves the child holding
  the pipe, so `CombinedOutput` waits forever on a command already cancelled.
  Measured — the deadline test ran to its own 30-second safety net on Linux while
  Windows happened to return. `WaitDelay` closes it.

- **The gate could not run on Windows at all.** `ditto_mutation_test.go`
  hardcoded `make`, where GNU make answers to `mingw32-make`; the command failed
  instantly with no output and ditto read that as a red baseline. It resolves the
  name the way `.githooks/pre-commit` has since the day the hook was unrunnable
  for the same reason.

- **A schema that breaks one file gives that file back**, rather than refusing
  the whole run. An instrumented file failing its own suite means the schema
  broke it, and that is a reason to stop schematising that file, not to stop
  measuring the repository. Falling back cannot hide a genuinely red repository:
  the ordinary path runs `verifyBaseline` once per release and refuses there.

- **The false-kill harness could not run.** It read the virus name off a label by
  cutting the file path from the front, and the label gained line and column at
  some point after that was written. The **7.3%** quoted from it all along
  predates the change; the rate is **11.1%**.

- **The test double now scores the way the reporter does.** The exclusion landed
  in `consolereporter` and stopped there, so every test through the fake went on
  agreeing with a version of ditto that no longer exists.

- **`ditto staged -h` advertised a default that changed two releases ago** — it
  said `link` was the default for `--sandbox`, which has been `copy` since 0.5.0
  — and **`-h` exits 0**, since `flag` reports a help request as an error under
  `ContinueOnError`.

### Changed

- **Gating is on for ditto's own gate.** It removes **394 of 727 compilations,
  54%**, because 60.1% of this repository's mutants can be expressed as a runtime
  branch — and over `internal/schemata/instrument.go`, 78 mutants with **28
  survivors** in them, both paths returned the same mutants and the same verdict
  for every one. The survivors are the point: an earlier comparison over a scope
  where everything died proved nothing and was thrown away.

- **Most of the performance counters are diagnosis, not gates.**
  `mutantsPerRelease` fired six times in the session that added it, and all six
  times the response was to write the new number down. A ratchet that only
  records is a changelog with a threshold on it.

  Two systematic reviews of the 2021–2026 literature say why: there is **no
  accepted performance metric for a mutation run** — the canonical enumeration
  holds 18 cost metrics and not one is the absolute cost of a run — the mutant
  count as a surrogate is a documented threat to validity at **44% average
  error**, and **nobody gates a build on the cost of a mutation run**.

- **The mutation score is no longer how ditto's own quality is judged.** It
  measures the user's tests. It ships with its composition beside it or not at
  all, and it is never a gate here: Google abandoned the ratio in IEEE TSE 2021
  as neither concrete nor actionable, fewer than 5% of mutants are subsuming, the
  correlation with real faults is weak once suite size is controlled, and below a
  high threshold the number is disconnected from fault revelation altogether —
  ditto's own gate runs at 0.5, which is below.

  `docs/metrics.md` carries the four metrics that replace it, each naming the
  decision it drives and where its threshold comes from.

### Documentation

- `docs/metrics.md`, and in `docs/experiments/`:
  `what-the-field-already-decided.md`, `what-the-field-measures-about-cost.md`,
  `turning-gating-on.md`, `a-counter-that-answers-for-itself.md` and
  `counting-the-real-repository.md` — the reviews, the measurements, and the
  predictions that were wrong.
- `AGENTS.md` gains what the score is and is not, and that a ratchet which only
  records is not a gate. Both contradict what was written there before, under
  that file's own rule: when a measurement contradicts something already written
  down, the measurement wins.
- `readme.md`: **read the survivors, not the number.**
- `.ditto.json` reached the readme's Settings section, the doc comments
  `pkg.go.dev` shows, and `ditto staged -h` — it shipped in 0.6.0 documented in
  two of five surfaces, and the release skill grew a checklist for the class.
- `docs/backlog.md` entries 14–21: what is measured and not built, including the
  11.1% of mutants the generator produces and discards, and the repository-sized
  gate that does not finish.

## [0.6.0] - 2026-08-28

### Added

- **`.ditto.json`, for the repositories that do not build from their index.** A
  sandbox is built from the index, and that is what makes a verdict about the
  change rather than about the desk it was written on. Some repositories still
  need something git does not carry: a generated directory the build embeds,
  generated bindings. Name those paths and they are copied from the working tree
  after the index is materialised.

  ```json
  {"generated": ["frontend/dist", "frontend/wailsjs"]}
  ```

  It does not widen what is read, and the guards are the point. **Naming a path
  git tracks is refused**: a tracked path has an index version, and letting the
  working tree win there is the hole the sandbox exists to close -- measured at
  7 of 8 verdicts moving. A named path that is not there is refused too, rather
  than skipped, because it was named for a reason. And every copy is announced,
  since those are the only bytes in the sandbox that did not come from git.

  Measured on a repository whose root package could not compile from its index:
  refused without the file, and **9 mutants, 8 killed, 1 survived** with it --
  the same verdicts a tracked placeholder produced, and no placeholder.

## [0.5.0] - 2026-08-27

The release that makes a sandbox a sandbox. Both entries below are the same
defect seen from two sides, and both were found by measuring rather than by
reading.

### Changed

- **A release copies the repository into its sandbox instead of linking it.** A
  large repository pays a fixed ~15s once per release for it — 16.5s over four
  mutants and 15.0s over nine on a 1960-file tree, because the sandbox is built
  once. `WithSandboxStrategy` reaches `"link"` and `"hardlink"` for a caller who
  has measured that their suite writes nothing.

### Fixed

- **A sandbox is a copy of the repository, not a reference to it.** A symlink is
  a reference to a file rather than a copy of one, and Go refuses to embed an
  irregular file — so **no package carrying a `go:embed` directive could build in
  a sandbox at all**. Every release before this one silently could not measure
  them.

  What that cost, measured on a real repository: it reported **9 mutants, 9
  killed, a perfect 1.00** for a package that never compiled, because a failing
  command is how ditto recognises a killed mutant. The truth is **8 killed and
  one survivor** — a real gap in that package's tests that nobody could see.

  **And the second reason is worse than the first.** A symlink is written
  *through* to its target, and a hard link shares the inode, so it is written
  through too. A suite that writes a file -- a golden it updates, a fixture it
  rewrites -- therefore writes into the repository being measured. Measured on a
  fixture whose test rewrites one tracked file: under links and hard links the
  source came back holding `REWRITTEN BY THE SUITE`; under copies, untouched.

  ditto own mutant write was always safe, because `Overwrite` removes a path
  before writing. That is why nobody noticed: the only writer anyone checked was
  ditto.

  The price is a **fixed** one, not a percentage: copying adds ~15s to a release
  on a 1960-file repository however many mutants it runs -- 16.5s over four and
  15.0s over nine. That is 85% of the smallest run and about 2% of a large one.
  Against a tool that can silently edit the repository it was asked to measure,
  it is worth paying.

  `WithSandboxStrategy` reaches `"link"` and `"hardlink"` for a measurement that
  wants them. `docs/experiments/the-sandbox-is-a-reference.md`.

- **A refusal says why the baseline is red**, carrying the test command's own
  output. Without it the message named a red baseline and left the reader to
  guess which of a hundred reasons it was, inside a sandbox they cannot open —
  measured at four rounds of elimination over an embedded file the output named
  on the first run.

## [0.4.0] - 2026-08-27

The release that stops making callers build a test to get in. Ditto runs as a
command now, and it knows what a staged change justifies — which is the job
every consumer had been writing for itself.

### Breaking

- **The Go floor is 1.27**, up from 1.25. Nothing in the code needs it; the
  toolchain does, and a floor that never moves is one that eventually blocks the
  tools. Consumers still on 1.26 have to move with it.

### Added

- **`ditto staged` mutates exactly what a staged change justifies.** It reads
  which files the index touches, which of their bytes, and — the part that is
  easy to skip — it runs the suite against a checkout of the **index**, not of
  the working tree.

  That last one is not caution, it is the difference between two answers.
  Measured on a fixture built for it: with the release pointed at the worktree
  instead, and one tracked file left dirty and unstaged, **seven of eight
  verdicts moved** — a score of 0.13 against 1.00 for the identical eight mutants
  of an identical file. Checking that the staged files themselves are clean does
  not cover it, because the file that moved them was never staged.

  Held against the wrapper it replaces, on two staged changes in a real
  repository: **4 mutants, 1 killed, 3 survived** on one file and **9, 5, 4** on
  two, identical on both paths, with the same survivors by mutator. The second
  case is the control — the numbers had to move, and both moved together.
  `docs/experiments/replacing-the-wrapper.md`.

- **`cmd/ditto`, with `run` and `staged`.** `go install
  github.com/Disble/ditto/cmd/ditto@latest`. Policy stays with the caller:
  `--threshold`, `--test-command` and `--exclude-prefix` are flags, and ditto has
  an opinion about none of them beyond its defaults. `--dry` answers what a
  staged change would cost without paying for it.

- **`ditto.Run`, a release that needs no `*testing.T`.** Everything below the
  entry point was already free of `testing` — `Release` used `t` in exactly four
  places — so this is the same run with other answers for them: cleanup by defer,
  an error instead of `t.Fail`, no subtest per mutant, and `ditto.Verbose()`
  instead of `testing.Verbose`, which panics outside a test binary.

  A golden holds the command to the same bytes as the library, minus the `PASS`
  a test binary adds and nothing else.

- **`ditto.RunStaged` and `ditto.PlanStaged`** for callers that want the staged
  pipeline without the command.

### Fixed

- **A red baseline reaches a command as an error, not a panic.** A process that
  panics prints a stack and reads as a defect, and a gate must never make its own
  refusal indistinguishable from being broken. Only that one type is recovered;
  every other panic is re-raised untouched.

### Unchanged, and measured to be

The eight exact counters in `perf/baseline.json` did not move, including
`sandboxesBuiltPerRelease`, whose own note records that it is registered through
`t.Cleanup` in `Release` — which is precisely what this release restructured. The
golden release output is byte-identical.

## [0.3.0] - 2026-08-14

The release that stops paying to start `go test` once per mutant, and stops
reporting survivors nobody can find. Every number below was measured, and the
notes that produced them are in `docs/experiments/`.

### Added

- **`Gated()` runs a file's mutants from one compilation instead of one each.**
  It instruments a source file so that every mutant of it becomes a gate chosen
  at run time, compiles the package once with `go test -c`, and selects each
  mutant by environment variable. Anything it cannot express that way keeps the
  path ditto has always taken, so no mutant is lost by turning it on.

  The reason it exists: **starting the test command costs 750–950 ms per mutant
  regardless of what the suite does**, and that fixed toll is the dominant cost
  of a run (`docs/experiments/test-invocation.md`).

  **It is opt-in, and that is measured rather than assumed.** On a real
  repository the two paths report the same 69 mutants, the same 58 killed and the
  same 11 survivors at identical addresses, for 51 invocations of the test
  command against 69 — but only **18 of 69 mutants gated**. The gating rate is a
  property of the file: `Expand` admits comparisons and integer literals and
  refuses everything else, so it ranges from **26% to 72%** across the real files
  measured. A compilation paid for a quarter of the mutants is an option, not a
  default. `docs/experiments/gated-by-default.md`.

  It builds with `go test -c`, so it applies to a Go package and it replaces
  `WithTestCommand` for the mutants it takes.

- **A gated run says how much of it came from one compilation**, as
  `Gated: 4 of 7 mutants ran from one compilation; 3 kept their own.` The two
  counters existed and were exact; nothing printed them, which is why a run that
  gated everything and a run that gated nothing looked identical for three
  measurements. Zero is spelled `none`, so the line reads as a fact rather than
  as an unfilled placeholder.

  Ungated runs print nothing, so their output is unchanged.

- **A performance baseline that ratchets in both directions.**
  `perf/baseline.json` pins exact integer counters — files linked per mutant,
  source parses per release, AST walks, laboratory runs — and `internal/perfbench`
  fails when one grows *and* when one shrinks, until the gain is written down. A
  gain nobody recorded is one that can be handed back later without anybody
  noticing. Wall clock is measured and reported, never gated on.

- **A golden release test.** `perf/baseline.json` pinned what a run costs and
  nothing pinned what it *says*, so a change could flip a mutant from killed to
  survived with every unit test green. The whole console output of a real run
  against a throwaway fixture is now compared byte for byte, on both paths.

### Fixed

- **`Gated()` was a no-op under `go test -v`.** `Release` wraps the laboratory
  stack in a verbose decorator whenever `testing.Verbose` is set, and that
  decorator implemented `Test` and not `TestAll` — so the batch assertion failed
  one layer above the gated laboratory, every mutant took its own compilation,
  and nothing said so. `go test -v` is what CI runs.

  Measured on the golden fixture, eight runs with one variable: **none of 7
  mutants gated under `-v` before, 4 of 7 after**, against a control of 4 of 7
  without `-v` on both sides. No verdict moved and no survivor's address
  changed. `docs/experiments/forwarding-the-batch.md`.

- **A red baseline used to report a perfect score.** The gated path already ran
  the instrumented file with no mutant selected — which is the file's own suite —
  and threw that verdict away. Under a suite that fails before anything is
  mutated, every selected run fails too, so every mutant was scored killed and the
  report said 1.00 without naming a cause. Measured at 4 killed of 4 against 1 of
  4 for the same mutants. It now refuses to score against a red baseline.

- **The compiler was found through `PATH`.** `gobuildrunner` named `go` and let
  the operating system search, which re-reads `PATH` at every call — so a
  directory an attacker can write to, or prepend, decides which compiler runs
  (SonarQube `go:S4036`, CWE-426). It is resolved once now, to an absolute path,
  preferring the `GOROOT` toolchain that built the running binary.

  The security reading is the smaller half. Ditto exists to compare verdicts, and
  building a mutant's tests with a different toolchain from the one running the
  suite would make a disagreement that is nobody's mutation look like one that is.

- **The test command inherited git's addressing.** Git exports `GIT_DIR` and
  `GIT_INDEX_FILE` to hooks, as absolute paths in a linked worktree, and
  everything spawned below inherited them — so a command meant for a sandbox
  addressed the real repository instead, succeeded, and said nothing. Measured
  twice: once leaving a commit named `fixture` on a live branch, once setting
  `core.bare=true` in a shared config and rewriting the committer identity.
  `internal/cmdtestrunner` now strips them, and a decoy test reproduces the
  incident against the unchanged code.

### Changed

- **A survivor is now clickable.** Every mutant carries `path:line:col` and the
  text it replaced, and survivors are listed by address before any diff is
  rendered. A report of five survivors used to name the files and nothing more
  precise: `go test` separates repeated subtest names with `#01` and `#02`, so
  two mutants of one kind in one file were told apart by a counter that says
  nothing about where either of them is.

  Measured over 135 mutants of four gofmt'd files: **129 of them could not be
  told apart from another mutant** by what ditto printed. The address is read off
  the difference between the file before and after the mutation — the same one
  the gated path already builds its gates from — so no virus was changed and
  `viruses.NewInfection` keeps its signature. See
  `docs/experiments/mutant-address.md`.

  The label a mutant is reported under has changed shape, from
  `path → Virus` to `path:line:col → Virus`. Subtest names change with it.

## [0.2.0] - 2026-08-13

The first release that makes ditto cheaper to run rather than only better
named. Every number below was measured against the fixture in
`internal/perfbench` and is enforced by `perf/baseline.json`.

### Added

- **`WithChangedRanges`** restricts a release to named byte ranges of named
  files. Every mutant costs a full run of the test command, so mutating a line
  the change never touched is charged at the same rate as one that matters. On
  the fixture, one changed function costs **4** laboratory runs where the whole
  repository costs 48.

  The ranges are held beside their file, and callers should keep them that way
  too. A byte offset only means something against the file it was measured in,
  because every file is parsed on its own and so every file's positions start
  from the same base. A scope that has lost track of which file a range came
  from makes each file answer to every range: mutants appear in code no diff
  touched, and the count grows as the square of the number of files. Two files
  scoped to different functions cost **8**, twice the single-file number, where
  a flat scope costs 16 — measured, because the fixture files are written to
  the same byte layout precisely so the offsets collide.

### Changed

- **One sandbox per run instead of one per mutant.** A sandbox is a walk of the
  repository and a symlink per file, roughly 0.45ms per file, and nothing about
  it depends on which mutant runs in it. Sandboxes are now pooled and the
  mutated file is restored before one is handed back. `sandboxesBuiltPerRelease`
  is 1, down from the mutant count.
- **Each source file is parsed once per release, not once per mutator.** With
  the default set that is fourteen parses per file reduced to one. Mutants come
  out in the same order; only the parsing is shared.
- **`.git` is no longer linked into a sandbox.** Nothing in it is Go source, so
  it produced no mutants while being copied for every one of them — measured at
  164 working files against 1324 git objects on one checkout. It is also the
  safer answer: the sandbox is one symlink per file, so linking Git's own
  directory handed anything running in there a live handle on the real
  repository's objects, config and refs.

### Fixed

- **Sandbox lifetime.** Sandboxes now live in one parent directory per process,
  named after the owning process id, removed through `t.Cleanup` in `Release`
  with retries — Windows holds a handle for a moment after the process that
  worked in the directory exits, and the unguarded removal panicked mid-run,
  which was itself orphaning every other sandbox. Six abandoned sandboxes were
  found on one machine, the oldest eight days old. A killed process is still
  not covered; reclaiming what it leaves is recorded in `docs/backlog.md`.

## [0.1.0] - 2026-08-13

First release under the ditto name. It is a fork of
[gtramontina/ooze](https://github.com/gtramontina/ooze) at v0.3.1 plus the
changes below; the copyright and licence remain Guilherme J. Tramontina's.

Version 0.1.0 rather than 0.4.0 on purpose: this is a new module path with no
published history, and the two things the fork exists for — latency inside a
TDD loop, and refusing to report a verdict it cannot stand behind — are stated
here, not yet built.

### Changed

- **Renamed the project and its module path to `github.com/Disble/ditto`.**
  The public entry point is now `ditto.Release(t, options...)`. Internal
  packages moved accordingly, and the exported testing helper package is now
  `dittotesting`.
- **The Go floor is now 1.25**, up from 1.22, and CI pins the same version.
  The `latest` toolchain job is no longer exempt from failing: that exemption
  existed to hide the breakage fixed below, and leaving it in place would have
  hidden the next one just as well.
- **A source file's identity no longer depends on the host.**
  `FSRepository.ListGoSourceFiles` now reports relative paths with forward
  slashes on every platform. That string is what `IgnoreSourceFiles` matches
  patterns against and what every report prints, so with the host separator in
  it a pattern written with `/` silently matched nothing on Windows, and the
  same run named files differently on different machines.

### Added

- **A recorded performance contract.** `perf/baseline.json` holds exact
  counters — files linked per mutant, source parses per release, test-command
  runs per release — enforced by `internal/perfbench`. A counter that grows
  fails the build; a counter that shrinks fails it too, until the gain is
  written down, so an improvement cannot be quietly handed back later.
  Wall clock is measured and reported but never gated: on a normal development
  machine the same workload varied by more than fifty percent between runs, and
  a threshold tight enough to catch a regression would fire on ambient load.
- `docs/backlog.md`, for observations worth acting on later.

### Fixed

- **The test suite builds and passes on Windows.** It did neither before, and
  upstream CI marked newer Go toolchains `continue-on-error`, so these went
  unnoticed:
  - `fstemporaryrepository_test.go` called `syscall.Umask`, which does not
    exist outside Unix, so the package failed to compile. The permission
    assertion is now split by platform.
  - `cmdtestrunner_test.go` compared `path.Base` output against a host path.
    `path` only understands forward slashes, so on Windows it returned the
    whole path. It uses `filepath.Base` now.
  - `fsrepository_test.go` built expected link paths by concatenating `/`
    while `filepath.WalkDir` returns host-native ones.
  - `testingtlaboratory_test.go` asserted that a subtest had been marked
    parallel by reading `testing.T`'s unexported `isParallel` field through
    `reflect` and `unsafe`. That cannot work on current Go: `T.Parallel`
    returns early when its parent has no barrier, which is the shape of any
    synthesized `*testing.T`, and supplying one makes it block forever sending
    on a nil signal channel. The laboratory exposes a seam for the assertion
    instead, and the fake no longer imports `reflect` or `unsafe`.

### Removed

- The `retract` block. It named published versions of the upstream module path,
  which do not exist under this one.

[0.9.0]: https://github.com/Disble/ditto/releases/tag/v0.9.0
[0.8.0]: https://github.com/Disble/ditto/releases/tag/v0.8.0
[0.7.0]: https://github.com/Disble/ditto/releases/tag/v0.7.0
[0.6.0]: https://github.com/Disble/ditto/releases/tag/v0.6.0
[0.5.0]: https://github.com/Disble/ditto/releases/tag/v0.5.0
[0.4.0]: https://github.com/Disble/ditto/releases/tag/v0.4.0
[0.3.0]: https://github.com/Disble/ditto/releases/tag/v0.3.0
[0.2.0]: https://github.com/Disble/ditto/releases/tag/v0.2.0
[0.1.0]: https://github.com/Disble/ditto/releases/tag/v0.1.0

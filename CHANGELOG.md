# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- **A sandbox is materialised with hard links, not symbolic ones.** A symlink is
  a reference to a file rather than a copy of one, and Go refuses to embed an
  irregular file — so **no package carrying a `go:embed` directive could build in
  a sandbox at all**. Every release before this one silently could not measure
  them.

  What that cost, measured on a real repository: it reported **9 mutants, 9
  killed, a perfect 1.00** for a package that never compiled, because a failing
  command is how ditto recognises a killed mutant. The truth is **8 killed and
  one survivor** — a real gap in that package's tests that nobody could see.

  A hard link is a second name for the same bytes: a regular file, which is what
  `go:embed` requires, and no copy, which is what 1960 files per sandbox demands.
  Three rounds with the order rotated: hard links **19.1-19.7s** against symlinks
  **19.8-22.6s** and copies **34.2-39.4s**. Copying was the obvious answer and is
  70% slower, so it is the fallback for a filesystem boundary rather than the
  rule. `filesLinkedPerSandbox` did not move.

  It is safe only because `Overwrite` removes a path before writing a mutant.
  Removing a name leaves the original alone; a write in place would have
  corrupted the repository being measured.

  `WithSandboxStrategy` reaches the old behaviour for a measurement that wants
  it. `docs/experiments/the-sandbox-is-a-reference.md`.

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

[0.2.0]: https://github.com/Disble/ditto/releases/tag/v0.2.0
[0.1.0]: https://github.com/Disble/ditto/releases/tag/v0.1.0

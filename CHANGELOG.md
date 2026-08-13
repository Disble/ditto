# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

[0.1.0]: https://github.com/Disble/ditto/releases/tag/v0.1.0

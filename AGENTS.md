# AGENTS.md

Canonical context for anyone — human or agent — working on ditto.

## What ditto is for

A fork of [gtramontina/ooze](https://github.com/gtramontina/ooze), pushing on
two things in this order.

**Fast.** Mutation testing is normally a CI job: run once, over everything,
overnight. Ditto targets the other place — inside a TDD loop, on staged
changes, while the code is still being written. The number that matters is
latency on the laptop that is also running the editor, not throughput on a
build server.

**Well made.** A fast answer that is wrong is worse than no answer, because
somebody acts on it. A misleading verdict is a defect here, not a caveat.

## Never run a mutation run against a repository you care about

**Not this repository. Not the repository that consumes ditto. Not any
checkout with work in it.** Build a throwaway project for the run and point
ditto at that.

This is not caution, it is a measured rule. A mutation run spawns processes,
writes into temporary trees, and — in its own tests — creates fixture
repositories with `git init`, `git config` and `git commit`. Git exports
`GIT_DIR` and `GIT_INDEX_FILE` to hooks, and in a **linked worktree it exports
them as absolute paths**. Everything spawned from that hook inherits them, so a
command meant for a temporary directory addresses the real repository instead.

Measured twice, both times running the suite from a pre-commit hook inside a
worktree:

- once leaving a commit named `fixture` on a live branch, replacing the tree;
- once setting `core.bare=true` in the shared config, which broke the main
  working tree outright, and writing the fixture identity into `user.name` and
  `user.email`, so six later commits were attributed to a test fixture.

Both times **the command succeeded**. That is the trap: a misdirected Git
command does not fail, so the damage is silent and the test still passes.

Ditto's own process spawning strips `GIT_DIR`, `GIT_INDEX_FILE`,
`GIT_WORK_TREE`, `GIT_OBJECT_DIRECTORY` and `GIT_COMMON_DIR`, in
`internal/cmdtestrunner`. That closes the path we found. It does not make the
rule optional — the next such path will be found the same way this one was.

This paragraph claimed that before any code did it: the fix from the incident
landed in the consuming wrapper, and this file was written as though ditto had
absorbed it. A decoy test — `cannot reach a repository outside the sandbox`, in
`internal/cmdtestrunner` — reproduced the incident against the unchanged code on
its first run, and now fails if the stripping is ever removed. **A sentence in
this file is not a guard.** If something here describes a safety property, look
for the test that refuses without it.

### How to verify isolation without risking anything

Use a decoy. Create a throwaway repository, aim `GIT_DIR` and `GIT_INDEX_FILE`
at it, run whatever you are testing, and assert the decoy's `HEAD`, commit
count and local config are unchanged. If the thing under test can reach outside
its fixture, the decoy shows it and nothing real is lost.

A test that can reach outside its fixture is not isolated, whatever it asserts.

## What the score is, and is not

The mutation score measures the user's tests. **It is not how ditto's own quality
is judged**, and it is never a gate here. Four peer-reviewed results say a bare
ratio is uninterpretable — Google abandoned it in IEEE TSE 2021 as "neither
concrete nor actionable", fewer than 5% of mutants carry the signal (ISSTA 2016),
the correlation with real faults is weak once suite size is controlled (ICSE
2018), and below a high threshold the number is disconnected from fault
revelation altogether (ICSE 2017). Ditto's own gate runs at 0.5, which is below.

So the score ships **with its composition beside it** — generated, non-viable,
killed by reason, live — or it does not ship.

What replaces it is four metrics that measure whether ditto's ANSWER can be
believed, each naming the decision it drives and where its threshold comes from.
They live in `docs/metrics.md`, and the reviews that produced them are in
`docs/experiments/what-the-field-already-decided.md`.

This section changed direction on 2026-08-28, under the rule below: a measurement
contradicted what was written here, so the claim went with its evidence.

## Performance is the metric

For a library whose reason to exist is being cheap enough to run, "it got
slower" and "it stopped working" are the same sentence. So the cost is
recorded, not remembered.

- `perf/baseline.json` holds **exact counters**: files linked per mutant,
  source parses per release, AST walks per release, laboratory runs per
  release. Integers. Identical on every machine.
- `internal/perfbench` enforces them, and **ratchets in both directions**. A
  counter that grows is a regression. A counter that shrinks also fails, until
  the gain is written into the baseline — a gain nobody recorded is one that
  can be handed back later without anybody noticing.
- **Wall clock is measured and reported, never gated.** A development machine
  is always busy; that is normal, not a lab defect. The same workload has
  varied here by more than fifty percent between runs. A threshold tight
  enough to catch a real regression would fire on ambient load, and a gate
  that cries wolf is one people learn to ignore.

When a timing comparison is unavoidable, prefer a **ratio measured inside one
run** over two absolute numbers measured minutes apart. The ratio cancels the
load; the absolutes do not.

Beware of proxies. A counter that measures one thing and is documented as
measuring another holds only until the two diverge — which is exactly when you
need it. `sourceParsesPerRelease` counted AST-node visits and called them
parses; that was true only while every parse implied one walk, and it silently
stopped being true the moment parsing was shared. That one is written up in
full in [`docs/learning-log.md`](docs/learning-log.md), along with the
measurements that redirected this project's optimisation plan.

## Parallelism is not a direction

Running mutants in parallel is ruled out as a performance strategy. The target
machine is a laptop that is simultaneously running an editor and a language
server; saturating it makes the tool unusable in exactly the loop it exists
for. Speed comes from doing less work, not from doing the same work on more
cores.

The existing `Parallel()` option stays for callers who want it. It is not a
direction the project develops.

## Every hypothesis states what would kill it

Before measuring, write the prediction as a number and name the result that
would falsify it. Then measure. A hypothesis that cannot fail is a belief.

Always include a **control**. Three staged files producing 36 mutants where 12
were justified means nothing on its own; it becomes a finding when one staged
file produces 4 and 4.

When a measurement contradicts something already written down — including in
this file — the measurement wins, and the claim goes with its evidence.

The operational form of this section — the note written before the run, the
control, counters against wall clock, and the cases that earned each rule — is
[`.claude/skills/falsifiable-measurement`](.claude/skills/falsifiable-measurement/SKILL.md).
Load it before measuring anything. Its case studies are the measurements
themselves, three of which killed a claim already made out loud.

## Where the reasoning lives

- [`docs/learning-log.md`](docs/learning-log.md) — append-only, one dated line
  per lesson, newest at the bottom, never rewritten. Append to it when
  something non-obvious turns out to be true. It explains the _why_; it never
  replaces a check that enforces the _how_.
- [`docs/backlog.md`](docs/backlog.md) — observations worth acting on later. An
  entry whose evidence turns out to be a misreading goes away with the
  misreading.
- [`perf/baseline.json`](perf/baseline.json) — the recorded cost, and the only
  thing entitled to say whether a change made ditto faster.

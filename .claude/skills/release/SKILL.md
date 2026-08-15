---
name: release
description: "Trigger: release, cut a version, tag, publish, changelog, ship it, is this ready to release. Cutting a ditto release, and the four traps that make it not work the obvious way."
license: Apache-2.0
metadata:
  author: "disble"
  version: "1.0"
---

## Activation Contract

Load before cutting a ditto release, before answering whether one is ready, and
before trusting any local Git ref about what is published.

Nothing here is ceremony. Each rule is a trap that was hit, and the state it left
behind is written beside it.

## The four traps, in the order they bite

### 1. This repository is a fork, so CI does not run on push or pull request

**`Disble/ditto` is a fork of `gtramontina/ooze`.** GitHub disables event-driven
Actions on a fork. `push` and `pull_request` trigger nothing, whatever
`on:` says in the workflow, and `actions/permissions` still reports
`enabled: true`.

Measured 2026-08-14: pushing a 43-commit branch and opening a pull request
against `main` produced **zero** workflow runs. Every run in the repository's
history — three of them — has `event: workflow_dispatch`.

This is the trap that has already produced one wrong claim. "CI went green for
the first time" on 2026-08-13 meant *a manually dispatched run went green*.
Event-driven CI has never fired in this repository, and two releases were
published believing otherwise.

    # confirm before believing any green
    gh api "repos/Disble/ditto/actions/runs?per_page=10" \
      --jq '[.workflow_runs[].event] | unique'

    # the only way to get CI evidence for a branch
    gh workflow run ci.yml --repo Disble/ditto --ref <branch>
    gh run watch <id> --repo Disble/ditto --exit-status

**No event trigger works, not only `push` and `pull_request`.** `mutation.yml`
fires on `workflow_run`, chained after CI on `main` — a sound design, and dead
here for the same reason: a CI run that completed successfully on `main` produced
no mutation run.

Tested deliberately rather than inferred, because the API says nothing is wrong:
`actions/permissions` reports `enabled: true`, every workflow reports
`state: active`, and no workflow carries a `paths:` filter. The probe was a
commit pushed straight to `main` touching `.github/workflows/`, with two minutes
of polling after it. **Zero runs.** The kill line was simple — if events worked,
that push creates a run — and it held.

`mutation.yml` now carries `workflow_dispatch` so the gate is reachable while
events are broken. Adding the trigger was not enough on its own: the job gated on
`github.event.workflow_run.conclusion`, which is null on a dispatch, so the first
dispatched run **skipped its only job and reported `completed`**. Read the job's
conclusion, never the run's.

### 2. `origin/main` lies, and `git fetch` may not be able to correct it

Git here is configured for SSH, and an agent without the key gets
`Permission denied (publickey)` on fetch, push and `ls-remote`. A stale
`origin/main` then looks authoritative: it read `b98e4ff` while the real `main`
was `dfd8ed4`, which put the unreleased commit count at 60 when it was 43, and
made two published releases look unpublished.

Ask GitHub, not the local ref:

    gh api repos/Disble/ditto/commits/main --jq '.sha[0:7] + "  " + .commit.message'
    gh api repos/Disble/ditto/releases --jq '.[] | .tag_name + "  " + .published_at'

The CHANGELOG range is computed against **that** sha, never against
`origin/main`.

### 3. Pushing needs the token, because the key is not there

`gh` is authenticated with a token that carries `repo` and `workflow`, so the
push goes over HTTPS with it. Do not rewrite the remote — pass the URL once:

    TOKEN="$(gh auth token)"
    git push "https://x-access-token:${TOKEN}@github.com/Disble/ditto.git" <branch>:<branch>

Filter the token out of anything printed: `| sed "s|${TOKEN}|***|g"`.

### 4. `gh` follows the working directory, and the shell resets it

Every Bash call starts in the session's default directory, so a bare
`gh pr checks 1` reports **another repository's** PR #1 and looks like an answer.
It happened, and the output was briefly believed.

Always pass `--repo Disble/ditto`.

## The steps

1. **Establish what is actually published.** The remote `main` sha and the
   release list, through `gh api`. Everything after that sha is the release.
2. **Decide the version from the diff, not from habit.** A new exported option is
   a minor. A changed report label — `path → Virus` becoming
   `path:line:col → Virus` — breaks anyone matching subtest names and is called
   out under **Breaking**, even pre-1.0.
3. **Write the CHANGELOG against the real range.** Every user-facing commit gets
   an entry with the number that justified it. An entry with no measurement
   behind it is a feature description, and this project does not ship those.
4. **Commit through the hook.** `.githooks/pre-commit` runs `make lint
   test.failfast`. On Windows the target is `mingw32-make`; the hook looks for
   `make`, `mingw32-make` and `gmake` in that order. Never bypass it.
5. **Push, open the pull request, then dispatch CI by hand** — see trap 1. The
   pull request alone proves nothing here.
6. **Read the checks that do run.** SonarCloud is inherited from upstream and
   applies its own quality gate; it is not part of this project's own bar, and it
   is still red or green in public. Report which.
7. **Tag and publish only after a green dispatched run**, and say in the release
   notes which checks ran and which could not.

## What must never happen

- **Do not run `make test.mutation` against a checkout with work in it.**
  `ditto_mutation_test.go` calls `Release` with `WithRepositoryRoot(".")`, which
  is what `AGENTS.md` forbids — backlog entry 3. It is safe on a clean CI
  checkout and nowhere else.
- **Do not report a green suite as a green release.** The local gate is
  `make lint test.failfast`; the golden compares both paths byte for byte; and
  CI is a separate question that trap 1 answers.
- **Do not add AI attribution to commits.** Conventional commits only.

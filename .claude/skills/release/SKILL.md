---
name: release
description: "Trigger: release, cut a version, tag, publish, changelog, ship it, is this ready to release, why did no workflow run. Cutting a ditto release, and the traps that make it not work the obvious way."
license: Apache-2.0
metadata:
  author: "disble"
  version: "2.2"
---

## Activation Contract

Load before cutting a ditto release, before answering whether one is ready, and
before trusting any local Git ref about what is published.

Nothing here is ceremony. Each rule is a trap that was hit, and the state it left
behind is written beside it.

## The four traps, in the order they bite

### 1. A fork starts with every event trigger dead, and the API will not say so

**Resolved on 2026-08-15, and kept here because the symptom is unrecognisable
without it.** `Disble/ditto` is a fork, and GitHub ships forks with event-driven
Actions switched off. `push`, `pull_request` and `workflow_run` all did nothing,
while `actions/permissions` reported `enabled: true`, every workflow reported
`state: active`, and no workflow carried a `paths:` filter. Nothing in the
configuration explained the silence.

It was settled by a probe with a kill line rather than by inference: a commit
pushed straight to `main` touching `.github/workflows`, then two minutes of
polling. **Zero runs.** Ten runs of ten in the repository's history carried
`workflow_dispatch`.

The fix is a single click in the fork's **Actions** tab — *"I understand my
workflows, go ahead and enable them"* — and there is **no REST API for it**.
After it, the same push produced four runs: CI, CodeQL, OSV-Scanner and the
chained mutation gate.

Two things this cost, and both are the reason it is written down:

- **CodeQL and OSV-Scanner had never run once.** The fork had three security and
  quality gates that did not exist, and nobody could tell, because a workflow
  that never runs also never fails.
- **Two releases were published believing CI was green.** It was — manually
  dispatched. "CI went green for the first time" meant a `workflow_dispatch`.

Verify, rather than assume, whenever a run is missing:

    gh api "repos/Disble/ditto/actions/runs?per_page=50"       --jq '"total=" + (.total_count|tostring) + "  " + ([.workflow_runs[].event]|unique|join(","))'

If that list is only `workflow_dispatch`, event triggers are off whatever any
setting claims.

**A dispatched run can still do nothing.** `mutation.yml` chains off CI with
`workflow_run`, and its job gated on `github.event.workflow_run.conclusion`,
which is null on a dispatch — so the first dispatched run **skipped its only job
and reported `completed`**. Read the conclusion of the **job**, never of the run.

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
4. **Walk every surface that documents behaviour, not just the CHANGELOG.**
   A release is where documentation drift becomes public, and the CHANGELOG is
   the surface hardest to forget — which is exactly why the others get missed.
   Tick each one against the change being released:

   - [ ] **`readme.md`, where the feature is used.** The section a reader lands
         on from the top.
   - [ ] **`readme.md`, the `## Settings` section.** It is *the* place someone
         looks for "how do I configure this". A second way to configure ditto
         that is documented only elsewhere is invisible from here.
   - [ ] **The doc comment on the exported function.** This is what
         **pkg.go.dev** shows, and it is the only documentation a library
         consumer sees. Anything explained solely in an `internal` package or in
         the readme does not exist for them.
   - [ ] **`cmd/ditto` help text.** Two failure modes, both real:
         a default named in a flag's description that is no longer the default,
         and behaviour with **no flag at all** — `.ditto.json` — which a reader
         checking `-h` would otherwise conclude does not exist. `PrintDefaults`
         cannot mention what is not a flag, so `flags.Usage` has to.
   - [ ] **`docs/experiments/`.** A number quoted in the release notes should be
         traceable to the note that produced it.

   Measured, on 0.6.0: `.ditto.json` shipped documented in the readme and the
   CHANGELOG and in **none** of the other three, and `ditto staged -h` was still
   advertising `"link" (default)` two releases after the default became `copy`.
   A help text that misstates a default is worse than one that omits it.

5. **Commit through the hook.** `.githooks/pre-commit` runs `make lint
   test.failfast`. On Windows the target is `mingw32-make`; the hook looks for
   `make`, `mingw32-make` and `gmake` in that order. Never bypass it.
6. **Push and open the pull request.** Since the fork opt-in (trap 1) that
   triggers CI, CodeQL and OSV-Scanner on its own. If it does not, event
   triggers have been turned off again — check before dispatching by hand,
   because a dispatched green and an event green are not the same evidence.
7. **Read every check, including the ones the pull request does not show.**
   SonarCloud is inherited from upstream and applies its own quality gate; it is
   not part of this project's own bar, and it is still red or green in public.
   Report which.

   **The mutation gate is not on the pull request.** `mutation.yml` chains off CI
   with `workflow_run`, which only fires for `push`, so `gh pr checks` never
   lists it and a green pull request says nothing about it. Ask for it by sha:

       gh api "repos/Disble/ditto/actions/runs?per_page=15" \
         --jq '.workflow_runs[] | .head_sha[0:7] + "  " + .name + "  " + (.conclusion // "-")'

   Measured, and the reason this step exists: it was red on **every push from
   0.4.0 through 0.6.0** -- three releases -- with a panic in the first seconds
   naming the file it died on. Nobody opened it. The defect was real and it was
   in the sandbox: `docs/experiments/a-symlink-in-the-tree.md`.
8. **Tag and publish only after a green run on the exact commit being tagged.**
   A rebase merge rewrites SHAs, so the branch's green belongs to a commit that
   no longer exists on `main` — v0.3.0 nearly shipped that way. Re-run on the
   merged head first, and say in the release notes which checks ran and which
   could not.

## What must never happen

- **Do not run `make test.mutation` against a checkout with work in it.**
  `ditto_mutation_test.go` calls `Release` with `WithRepositoryRoot(".")`, which
  is what `AGENTS.md` forbids — backlog entry 3. It is safe on a clean CI
  checkout and nowhere else.
- **Do not report a green suite as a green release.** The local gate is
  `make lint test.failfast`; the golden compares both paths byte for byte; and
  CI is a separate question that trap 1 answers.
- **Do not report a perfect mutation score without reading what it cost.** The
  gate once printed 431 of 431 killed, a flawless 1.00, in 5.46 seconds, because
  `make` needed a git directory that a mutant sandbox does not have and ditto
  reads a failing command as a killed mutant. The real numbers were 329 killed
  and 102 survived. The laboratory now refuses to score against a red baseline,
  so this exact failure cannot return silently — but a score is still a number
  with a duration beside it, and the two have to agree.
- **Do not add AI attribution to commits.** Conventional commits only.

PATH := $(PWD)/.bin:$(PATH)
SHELL := /usr/bin/env bash -eu -o pipefail
CPUS ?= $(shell (nproc --all || sysctl -n hw.ncpu) 2>/dev/null || echo 1)
MAKEFLAGS += --warn-undefined-variables --output-sync=line --jobs $(CPUS)

# Asked for rather than assumed: in a linked worktree `.git` is a file, not a
# directory, so writing `.git/.hooks.log` fails there and every target that
# depends on it fails with it — including the pre-commit hook, which made the
# repository uncommittable from exactly the place this doctrine asks you to work.
#
# Asked for *quietly*, and skipped entirely when there is no answer. A mutant is
# tested in a sandbox, and a sandbox is a copy of the tree with no `.git` in it —
# so `git rev-parse` failed, `git-dir` came back empty, the target became
# `/.hooks.log`, and make died with Error 128 before running a single test.
#
# Ditto reads a failing test command as a killed mutant. So this line scored
# **431 of 431 mutants killed, a perfect 1.00, in 5.46 seconds** on ditto's own
# gate — a green that had compiled nothing. Measured in
# docs/experiments/false-perfect-score.md; the guard that makes it impossible to
# repeat is in internal/laboratory.
#
# Hooks belong to a checkout. Outside one there is nothing to install and nothing
# to record, which is why this is skipped rather than made to fail softly.
git-dir := $(shell git rev-parse --git-dir 2>/dev/null)

ifneq ($(git-dir),)
$(git-dir)/.hooks.log:
	@git config core.hooksPath .githooks
	@git config --get core.hooksPath > $@
pre-reqs += $(git-dir)/.hooks.log
endif

test: $(pre-reqs)
	@gotestsum --format-hide-empty-pkg -- -race -cover -timeout=60s -shuffle=on ./...
.PHONY: test

test.failfast: $(pre-reqs)
	@gotestsum --format-hide-empty-pkg --max-fails=1 -- -timeout=60s -failfast ./...
.PHONY: test.failfast

test.mutation: $(pre-reqs)
	@go test -timeout=30m -count=1 -v -tags=mutation
.PHONY: test.mutation

lint: $(pre-reqs)
	@golangci-lint -v run
.PHONY: lint

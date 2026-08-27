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

# The goldens run real releases — one test process per mutant — so the root
# package is the slow one and the timeout is about it rather than about the unit
# tests. Measured on 2026-08-27: 22s for the release golden, 24s for the staged
# one, 10s for the command-equals-library one, 67s for the package.
#
# 60s used to be the number and it was already marginal at 53s. A timeout that
# the suite grows into is one that fails for the wrong reason and gets raised in
# a hurry by whoever is unlucky; this one has room and a number beside it.
test-timeout := 300s

test: $(pre-reqs)
	@gotestsum --format-hide-empty-pkg -- -race -cover -timeout=$(test-timeout) -shuffle=on ./...
.PHONY: test

test.failfast: $(pre-reqs)
	@gotestsum --format-hide-empty-pkg --max-fails=1 -- -timeout=$(test-timeout) -failfast ./...
.PHONY: test.failfast

test.mutation: $(pre-reqs)
	@go test -timeout=30m -count=1 -v -tags=mutation
.PHONY: test.mutation

# golangci-lint embeds the Go it was built with, and refuses a config targeting a
# newer language version than that. A prebuilt binary therefore expires the day
# the floor moves. Measured on 2026-08-27: a 1.27 floor against a linter built
# with go1.26 refused every commit locally, and failed both legs of the CI matrix
# with `the Go language version (go1.26) used to build golangci-lint is lower
# than the targeted Go version (1.27)`.
#
# Building it here, with the toolchain this repository targets, is what keeps
# that from happening again: one pin, in one place, that cannot go stale against
# the floor. PATH already puts .bin first, so this is the golangci-lint every
# target sees, locally and in CI.
#
# The pin still had to move for 1.27, and rebuilding was not enough: 2.12.2
# vendors honnef.co/go/tools v0.7.0, which panics building IR for Go 1.27's
# internal/poll. That is a Linux-only crash — Windows has different sources for
# that package — so a green local gate said nothing about it.
golangci-lint-version := v2.13.1

.bin/.golangci-lint-$(golangci-lint-version):
	@mkdir -p .bin
	@GOBIN="$(CURDIR)/.bin" go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(golangci-lint-version)
	@touch $@

lint: $(pre-reqs) .bin/.golangci-lint-$(golangci-lint-version)
	@golangci-lint -v run
.PHONY: lint

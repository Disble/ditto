PATH := $(PWD)/.bin:$(PATH)
SHELL := /usr/bin/env bash -eu -o pipefail
CPUS ?= $(shell (nproc --all || sysctl -n hw.ncpu) 2>/dev/null || echo 1)
MAKEFLAGS += --warn-undefined-variables --output-sync=line --jobs $(CPUS)

# Asked for rather than assumed: in a linked worktree `.git` is a file, not a
# directory, so writing `.git/.hooks.log` fails there and every target that
# depends on it fails with it — including the pre-commit hook, which made the
# repository uncommittable from exactly the place this doctrine asks you to work.
git-dir := $(shell git rev-parse --git-dir)

$(git-dir)/.hooks.log:
	@git config core.hooksPath .githooks
	@git config --get core.hooksPath > $@
pre-reqs += $(git-dir)/.hooks.log

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

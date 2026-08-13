# Backlog

Things worth doing that were not worth derailing the change that found them.
Each entry records what was observed, not what was guessed. An entry whose
evidence turns out to be a misreading goes away with the misreading.

Nothing here is a promise.

---

## 1. A surviving mutant has no address

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

## 2. The logo still belongs to upstream

`.assets/logo.svg` is ooze's. The licence permits reuse, but the mark is the
one thing a fork should not inherit — it is upstream's identity, not its code.
Needs replacing before ditto is presented anywhere as its own project.

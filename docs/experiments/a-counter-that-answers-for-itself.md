# A counter that answers for itself

`mutantsPerReleaseOnThisRepository` was added to close a real gap: eight recorded
counters all measure a six-file synthetic fixture, and every one stayed green
while the repository's own cost went from 431 mutants to 660 and the gate blew
past its thirty-minute timeout.

It closed that gap and opened a worse one. This note is what a blind review found
and what measuring it cost to settle.

## The defect

`internal/perfbench/repository_test.go` asserted against the tree it was compiled
inside — `filepath.Abs("../..")`. Two facts turn that into a lie:

- ditto's sandbox carries every file but `.git`
  (`internal/fsrepository/fsrepository.go`), `_test.go` files included.
- each mutant's test command is `./...` (`makefile`, `test.failfast`).

So the counter ran **inside every mutant's sandbox and read the mutated tree**.
Any mutation that changes the number of mutable sites moves the count, the
assertion fails, `make` exits non-zero — and `internal/cmdtestrunner` recognises
a killed mutant by exactly that.

**A counter meant to watch the run became an answer to it.**

## Measured, with the control

The first attempt was wrong and is worth recording. Running ditto over
`internal/ignoredrepository/ignoredrepository.go` — 3 mutants — with and without
the counter gave **3 killed, score 1.00 both times**. The counter changed
nothing there, because those mutants break real tests anyway. The experiment
only discriminates on a mutant that would otherwise **survive**.

`internal/schemata/instrument.go:147:3 → Range Break` is one: it survived a
78-mutant run over that file made before the counter existed. Applying it by hand
and running the mutants' own command:

| tree | the mutants' test command | verdict |
| --- | --- | --- |
| no counter | `DONE 368 tests, 10 skipped in 104.951s` | **survived** — correct |
| counter, untagged | `REGRESSION mutantsPerReleaseOnThisRepository: 661, baseline 660 (+1)`, 7 failures, `Error 3` | **killed** — false |
| counter, tagged | `DONE 368 tests, 10 skipped in 210.867s` | **survived** — correct |

The other failures in the middle row are `--max-fails=1` cutting the run short;
only the counter failed on its own, in 0.41s.

**Population.** A census by virus over the repository's 660 mutants:

    Range Break  66      each inserts an *ast.BranchStmt: one more mutable site
    Loop Break   15      operates on existing ones; not verified to move the count

So up to **66 of 660 — a tenth of the run** — was answerable this way. How many
of those would otherwise survive is not measured; `instrument.go` alone held four
candidates.

## Which fix, measured

Three properties, all three required:

1. **Right** — reads 660, the real repository's count.
2. **Immune** — does not run inside a mutant's sandbox.
3. **Alive** — actually runs in the ordinary gate. A ratchet nobody runs is not
   a ratchet.

Same clone, same mutation, three exact observations each:

| candidate | reads 660 | immune | alive |
| --- | --- | --- | --- |
| `//go:build mutation` | yes | yes | **no** |
| frozen fixture | **no — 48** | yes | yes |
| a tag the gate passes and the mutants' command does not | yes | yes | yes |

`mutation` dies on a detail worth writing down: `make test.mutation` is
`go test -tags=mutation` over the **root package alone**, with no `./...`. A
mutation-tagged counter would never run anywhere.

A frozen fixture dies on the first property. The existing fixture counts 48, and
a frozen copy cannot follow a tree that changes — which is the whole job.

## What shipped

The tag is `livetree`, `make test.counters` runs it, and the pre-commit hook runs
that alongside `lint` and `test.failfast`.

## The part that generalises, and is not fixed

This is ditto's own counter, so ditto's makefile could have excluded
`internal/perfbench` and been done. That would protect ditto and nobody else.

**Any repository using ditto whose suite reads its own tree has this defect**, and
ditto neither detects it nor says anything. A test that asserts a property of the
source it is compiled inside becomes a false-killer under mutation. Ditto's own
census says the class currently has one member — `grep` finds no other test here
reading `../..`, and the other eight counters use the synthetic fixture — but
that is a fact about this repository, not about the tool.

## What the earlier note got wrong

`counting-the-real-repository.md` established determinism with four identical
runs of 660. Every one of them was on a tree **at rest**. It never asked whether
the counter is stable under the mutation the gate itself applies, which is the
one condition that mattered. Four agreeing runs proved the counter is not noisy;
they proved nothing about the only environment it was going to run in.

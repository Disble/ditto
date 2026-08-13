# Experiment — which viruses can be gated, and how much of a run they are

Written before reading any virus's implementation and before counting anything.

## Why this is being measured

The gate rewrite is proven for the comparison family: same verdicts, one
compilation instead of N. Ditto ships fourteen viruses, and the contract that
lets ditto build gets designed once. If every virus can be gated, the contract is
one path. If only some can, ditto needs **two** paths — gated where possible,
compile-per-mutant where not — and that is a different, larger contract.

So the shape of the contract depends on this answer, and writing the contract
first would mean writing it twice.

## The three categories

The gate is `(m == id && mutated) || (m != id && original)`. It is an expression
that yields a bool, so it can only stand where the original did.

| Category | What it is | Does the gate apply |
| --- | --- | --- |
| **A** | an expression yielding a **bool** | yes, unchanged |
| **B** | an expression yielding something else | no — Go has no conditional expression, so a non-bool site needs a function call or a statement rewrite |
| **C** | a **statement**, not an expression | no — nothing to substitute an expression into |

## H1 — the classification

Predicted from each virus's name and Go's semantics, without reading its code.
*Each row is falsified by its own implementation.*

| Virus | Predicted | Reasoning |
| --- | --- | --- |
| comparison | A | known: `BinaryExpr` operator swap, bool result |
| comparisoninvert | A | inverts a comparison, still a bool `BinaryExpr` |
| comparisonreplace | A | replaces the comparison with a bool constant |
| loopcondition | A | a loop condition is a bool expression |
| arithmetic | B | `+` to `-`: a number, not a bool |
| bitwise | B | `&` to `|`: a number |
| integerincrement | B | an integer literal |
| integerdecrement | B | an integer literal |
| floatincrement | B | a float literal |
| floatdecrement | B | a float literal |
| arithmeticassignment | C | `+=` is a statement |
| arithmeticassignmentinvert | C | same |
| loopbreak | C | inserts or changes a `break` |
| rangebreak | C | same |

Predicted totals: **A 4, B 6, C 4**.

## H2 — the share that decides the contract shape

Counted over real Go source, as exact integers.

**H2a — category A is a substantial share on its own.** A ≥ 35% of all mutants.
*Falsified below 35%.*

**H2b — the statement-level category is small.** C < 20% of all mutants.
*Falsified at 20% or above.*

H2b is the one that decides. If C is small, a mixed design — gate the
expressions, fall back to today's compile-per-mutant for the statements —
captures nearly all of the win while leaving every existing mutant working. If C
is large, gating cannot be the main path and the plan changes.

## Control

A counter that disagrees with ditto is measuring something else. The control is a
case already run through the real gate: dharness's `internal/report/human.go`,
with only the byte range of `if hidden > 0 {` in scope, produced exactly **4**
candidate mutants — Comparison, Comparison Invert, Integer Decrement and Integer
Increment.

The counter must reproduce that: 4 mutants, those 4 names, from that range. If it
does not, nothing below it counts.

## Fixture

The counter imports ditto's own virus packages and walks source with `go/ast`,
so it counts what ditto would infect rather than an imitation of it. It mirrors
`GoSourceFile.Incubate` exactly — parsed with `ParseComments|AllErrors`, every
node offered to every virus, `types.Info` nil — and skips `_test.go`, which is
what `fsrepository` does.

It was run against the real checkouts, not copies. The rule it does not break is
about **mutation runs**: this opens files read-only, writes nothing and starts no
process. Saying it used a copy would have been tidier and untrue.

## Results

Run 2026-08-13. Exact integers; nothing here is timed.

### The control passed first

`internal/report/human.go`, line 426 only: **4 mutants** — one Comparison, one
Comparison Invert, one Integer Decrement, one Integer Increment. That is the same
count and the same four names the real gate reported for that staged line. The
counter counts what ditto counts.

### H1 held, 14 of 14 — with one distinction it forced

| Codebase | mutants | A | B | C |
| --- | --- | --- | --- | --- |
| dharness `internal/` | 1293 | 615 (47.6%) | 520 (40.2%) | 158 (12.2%) |
| ditto `internal/` | 128 | 60 (46.9%) | 47 (36.7%) | 21 (16.4%) |
| stdlib `net/http` | 9551 | 4954 (51.9%) | 4099 (42.9%) | 498 (5.2%) |

Every predicted category survived contact with its implementation, but
`loopcondition` nearly took one down and the reason matters for the rewriter:
it matches `*ast.ForStmt`, a **statement**, and then mutates `statement.Cond`,
a **bool expression**, replacing it with `0 != 0`. The category is a property of
what gets mutated, not of the node the virus matches on. A rewriter that looks
for bool expressions and stops there would miss it.

`rangebreak` was checked the same way and is genuinely statement-level: it
prepends a `break` to `statement.Body`.

### H2a held: A ≥ 35%

Measured 46.9%, 47.6% and 51.9%. Gating only the bool-expression family covers
about half of every mutant ditto produces, on three unrelated codebases.

### H2b held: C < 20%

Measured 5.2%, 12.2% and 16.4%. The statement-level category that no expression
gate can reach is a small minority everywhere it was counted.

### What this decides

**The contract needs two paths, and that is affordable.** Gate what can be
gated, keep today's compile-per-mutant for the rest, and every existing mutant
goes on working — which is the constraint this whole line of work is under.
A is proven and is half the mutants. C is small and stays on the old path. B, at
37–43%, is the open question: those are expressions, but not bools, and Go has no
conditional expression, so the same trick does not transfer. Nothing here says
whether B can be gated.

### An aside worth keeping

Two of the fourteen viruses produce essentially nothing on real Go code:
`floatincrement` and `floatdecrement` found **0** mutants in dharness and ditto,
and **1** in the whole of `net/http`. `bitwise` found 1 and 1. Meanwhile
`comparisoninvert` alone is 400 of dharness's 1293 — 31% of every mutant from one
virus.

## What this does NOT establish

- **Nothing about category B.** Whether a non-bool expression can be gated at all
  is unmeasured, and it is 37–43% of the mutants.
- **Counting is not running.** These are the mutants ditto would *generate*.
  Whether a gated build reaches the same verdicts is only proven for the
  comparison family, on one package, in `schemata-feasibility.md`.
- **Three codebases are not a survey.** Two are the author's own and share a
  style. `net/http` is the only independent one, and its C share is the lowest of
  the three, which is the direction that flatters the conclusion.

# Experiment — how much of category B reaches the fast path

Written before the measurement.

## Why this is being measured

Category B — expressions that do not yield a bool — is 37–43% of every mutant
ditto produces. Category A is proven and is about half. If B can mostly be gated,
the fast path covers ~85% of a run and the two-path contract is a formality with
a small tail. If B mostly cannot, the fast path is half a run and the slow path
is a permanent first-class citizen. The contract is written once, so this is
worth one measurement.

## What is already settled without measuring

The boolean gate works because `(m == id && mutated) || (m != id && original)` is
an expression of the same type as what it replaces. For a non-bool site there is
no such form: Go has no conditional expression, so the only runtime selection
available is a **function call**.

And a function call is never a constant. `integerincrement` mutates a literal's
**text** — `42` becomes `43` — so the site stays an untyped constant that adapts
to its context. A call returns a typed value and adapts to nothing. **No gate can
preserve untyped-ness**; that is a fact about Go, not a hypothesis.

So the question is not whether some sites break. It is **how many**.

## H1 — a call gate fails on a substantial share of integer-literal sites

Rewrite one site at a time into a generated `func dittoLitN() int` and compile.
The compiler is the oracle; no type information has to be reconstructed.

*Prediction: the call gate compiles for fewer than 80% of integer-literal sites.*
*Falsified at 80% or above.*

The contexts expected to break are ordinary Go, not exotica: `const`
declarations, array lengths, `iota` blocks, and any site where the untyped
constant becomes something other than `int` — `5 * time.Second` is the common
one, where `5` must adopt `time.Duration` and an `int` will not.

## Control

The same one-at-a-time harness, run over the **comparison** sites with the
boolean gate that is already proven, must compile **all** of them. Those sites are
known to work — `schemata-feasibility.md` ran them and got matching verdicts — so
anything less than 100% means the harness is broken and H1's number means
nothing.

## Fixture

A throwaway copy of dharness, one package. Each rewrite is applied alone and the
original is restored before the next, so no two gates are ever in the file at
once.

## Results

Run 2026-08-13 over six files of a throwaway dharness copy, one site at a time.

| File | comparison sites (control) | literal sites | literals compiled |
| --- | --- | --- | --- |
| `internal/jsconfig/jsconfig.go` | 12 of 12 | 26 | 26 |
| `internal/report/human.go` | 12 of 12 | 50 | 47 |
| `internal/setup/steps.go` | 1 of 1 | 8 | 8 |
| `internal/tool/tool.go` | 1 of 1 | 3 | 3 |
| `internal/setup/files.go` | — | 3 | 3 |
| `internal/preset/preset.go` | — | 1 | 1 |
| **total** | **26 of 26** | **91** | **88 (96.7%)** |

**Control passed:** every comparison site compiled, as the proven gate must.

**Second control, which the first run did not have:** the harness had to be shown
capable of saying no. On the first package it reported 26 of 26 and that number
was worthless until `internal/report/human.go` refused three sites. A harness
that cannot fail is measuring nothing.

### H1 is dead

Predicted: the call gate compiles for fewer than 80% of integer-literal sites,
falsified at 80% or above. **Measured 96.7%.**

The reasoning was right and the number was badly wrong. Every refusal is one of
the contexts named in advance:

- `const wrapWidth = 70` — a constant declaration.
- `const declaredSideIndent = 15` — the same.
- `float64(ms)/1000` — the untyped `1000` has to become a `float64`, and a
  function returning `int` will not divide one. This is `5 * time.Second` with
  different names, exactly the mechanism predicted.

So untyped constants do break under a call gate, for precisely the stated reason.
They are simply rare: three sites in ninety-one. The claim that should be carried
forward is the mechanism, not my estimate of how often it fires.

### What it means for the contract

Category B's mutants are dominated by literals — in dharness, 456 of B's 520 are
`integerincrement` and `integerdecrement`, 88% of the category. At 96.7%, the
gated path would reach roughly:

    A                      615
    literals, 96.7% of 456  441
    ------------------------------
                           1056 of 1293  ≈ 82%

The slow path keeps the statement category, three literals in ninety-one, and
arithmetic `BinaryExpr` sites until those are measured. A tail, not a second
first-class path — which is a weaker conclusion than `virus-families.md` reached,
and this measurement is why.

## H2 — a literal gate that compiles also reaches today's verdicts

Asked because H1 answered the compiler and nothing else, and a gate that compiles
while returning the wrong value would have passed every check above.

`internal/jsconfig/jsconfig.go` is the file where all 26 literal sites compiled,
so all 26 are gated at once, the package is compiled once, and each mutant is
selected at runtime — against today's path of editing one literal's text and
running the suite, 26 times.

*Falsified by a single disagreement between the two verdict vectors, or by the
gated package with nothing selected behaving differently from the untouched one.*

No claim is made here about which unmeasured gap matters more. Arithmetic
`BinaryExpr` sites are still untested, and until both are run there is no basis
for ranking them.

**Result: holds.**

    baseline, untouched                    : suite passed
    instrumented, nothing selected         : suite passed

    verdicts, one edited literal (today)   : KKKKKKKKKKKKKKKKKKKKKKSKKK
    verdicts, instrumented and selected    : KKKKKKKKKKKKKKKKKKKKKKSKKK

    compilations : today 26, instrumented 1
    wall clock   : today 22.18 s, instrumented 3.51 s

Twenty-five killed, one survived, and the same one survived both ways. The
control was run first on the same file with the comparison family and reproduced
`KKKSKKKKSKKK` exactly, so the extension did not disturb the proven path.

A call gate for a literal reaches today's verdicts. The gap H1 left open —
compiling is not behaving — is closed for this family on this file.

*(The tool prints "comparison sites" for both families; the count above is of
literal sites. A cosmetic defect in the harness, recorded rather than quietly
fixed, because the number it labels is the one the reader has to trust.)*

## What this does NOT establish

- **Both families were gated separately, never together.** Each run rewrote one
  family in one file. A file carrying a boolean gate and twenty-six generated
  functions at once has not been compiled, and that is what a real run would do.
- **One file for H2.** The verdict equivalence covers `jsconfig`'s 26 literal
  sites. The three sites that refused to compile live in another file and were
  not part of it, so nothing here says what happens when a file contains both
  gateable and ungateable literals.
- **Arithmetic `BinaryExpr` sites are untested** — `a + b` mutated to `a - b`,
  63 of dharness's 520 B mutants. They need the operand's type, which a per-site
  generated function may or may not infer.
- **One repository, six files, ninety-one sites.** `net/http` was not run, and it
  is the only independent codebase available here.
- **The probe is one Go version on one platform.** Nothing here depends on that,
  but nothing here rules it out either.

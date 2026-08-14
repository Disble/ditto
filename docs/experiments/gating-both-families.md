# Experiment — gating two families in one file, and what arithmetic does

Written before the measurement.

## Why this is being measured

Both gates are proven, each alone, each in its own run. A real run gates
everything in a file at once, and that has never been compiled. Two things could
go wrong, and one of them would be a silent regression rather than a failure.

## H1 — a literal nested inside a comparison is lost

`if hidden > 0` is one comparison site and one literal site, and the literal is
**inside** the comparison. Sites are rewritten back to front, which is safe while
no site contains another — ordered comparisons cannot nest in one another,
because their result is a bool and a bool is not ordered. A literal inside a
comparison breaks that assumption: the comparison's replacement is built from the
**original** bytes, so it overwrites the literal's gate that was already applied.

If that happens, selecting the lost literal's id changes nothing, the code runs
unmutated, the suite passes, and the mutant reads as **survived** — while today
it is killed. A mutant that used to be tested and no longer is, reported as a
survivor rather than as an error.

*Prediction: every literal nested inside a comparison flips from killed to
survived under the combined gate. Falsified if the two verdict vectors match.*

The fixture must contain nesting for this to be able to fail, so the count of
nested sites is reported first. Zero nested sites would make the run vacuous, not
passing.

## H2 — an arithmetic mutant that cannot compile breaks the whole gated build

`arithmetic` turns `+` into `-`. Go's `+` also concatenates strings, and `-` does
not apply to them, so `name + suffix` mutates into code that does not compile.

Today that is harmless: the mutated file fails to build, the test command exits
non-zero, and ditto counts the mutant as **killed**. Under a shared compilation
it is not harmless at all — one such site makes the single build fail, and with
it every other mutant in the run.

*Prediction: dharness contains at least one arithmetic site whose mutated form
does not compile. Falsified if every arithmetic site's mutation compiles.*

If it holds, the gated path cannot simply take every arithmetic site; each one
has to be proven to compile before it is admitted, which is a rule the contract
has to carry.

## Control

The comparison family alone must still reproduce `KKKSKKKKSKKK` on
`internal/jsconfig/jsconfig.go`, as it has in every run so far. A harness that
has drifted measures nothing.

## Results

Run 2026-08-13 on a throwaway dharness copy.

**Control passed:** the comparison family alone reproduced `KKKSKKKKSKKK` on
`internal/jsconfig/jsconfig.go`, as in every run before this one.

### H1 — the mechanism held, the symptom did not

`jsconfig` has 38 sites with both families enabled, and **3 literals nested
inside a comparison**, so the fixture could fail.

Predicted: the nested literal is silently overwritten, its mutant runs unmutated,
and it reads as survived where today it is killed. **Measured: a syntax error.**

    jsconfig.go:263:103: unexpected name ittoLit22, expected semicolon or newline

The outer replacement is spliced using the original offsets into a buffer whose
length has already changed, so it does not cleanly overwrite the inner gate — it
decapitates the identifier. That is louder than predicted, and better. It is not
a guarantee: a different interleaving could still land on valid code, and nothing
here rules that out.

With nested sites excluded and the remaining ones applied strictly in descending
offset order:

    35 sites (12 comparison, 23 literal), 0 nested

    verdicts, one mutation at a time (today) : KKKSKKKKKKSKKKKKKKKKKKKSKKKKKKKKKKK
    verdicts, instrumented and selected      : KKKSKKKKKKSKKKKKKKKKKKKSKKKKKKKKKKK

    compilations : today 35, instrumented 1
    wall clock   : today 24.81 s, instrumented 3.89 s

**Both families gate together.** The price is exact: 3 of 38 sites, 7.9%, cannot
be gated and stay on the per-mutant path.

### H2 — held, and it makes a rule for the contract

`internal/tool/tool.go:44` is `return binary + "@latest"`, where `binary` is a
string. Mutated as `arithmetic` mutates it:

    invalid operation: operator - not defined on binary (variable of type string)

Checked by building that exact mutation directly, rather than inferred from the
180 ms the run took, because a fast result is not evidence of the reason it was
fast.

Today this costs nothing: the build fails, the test command exits non-zero, and
ditto counts the mutant as killed. Under one shared compilation it costs
everything — the single build fails and every other mutant in the run goes with
it. **No site may enter the gated path until its mutation is known to compile.**

### A defect in ditto today, found on the way

An uncompilable mutant is scored **killed**. Nothing discriminated it: the
compiler refused, the command failed, and ditto read that as a test catching the
mutation. Every such site inflates the mutation score with a kill no test earned.
This has nothing to do with gating and is true of ditto as it ships.

## What this does NOT establish

- **The arithmetic gate was never implemented.** To measure H2 quickly the
  harness reused the boolean gate for arithmetic sites, which is wrong by
  construction — `a + b` is not a bool — so the gated half of that run is
  meaningless and only the today half counts. What a real arithmetic gate would
  do is still unmeasured.
- **One file for the combined run.** 38 sites in `jsconfig`. The 7.9% nested
  share is that file's number, not a rate.
- **Nested detection is syntactic.** It finds a literal inside a comparison. Other
  containments between gateable sites have not been enumerated.

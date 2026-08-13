# Case studies

Measured cases from ditto and dharness. Each one is here because it changed a
decision, and three of them killed a claim the author had already made out loud.

## A number that was never an alternative

**Claim:** compiling with `go test -c` and running the binary is about a third of
the cost of `go test`, from first-wave figures — `go test -c` at 334 ms against
`go test` at 1100 ms.

**Prediction:** B/A ≈ 0.34, falsified above 0.70.

**Measured:** 1.16 to 1.36 in all six rounds. Compiling separately and then
starting the binary is *worse* than `go test`, by about 300 ms per mutant, on
both a trivial and a heavy suite.

**Why the claim was wrong:** the two figures compared things that are not
alternatives. `go test` compiles *and* runs; `-c` compiles, links to disk, and a
separate process load follows. The 334 ms was never the whole job.

**Lesson:** a ratio between two numbers is only an argument when both numbers
measure the same job.

## The residue that belonged to someone else

**Claim:** roughly 4.5 s of a run is ditto's own startup.

**How it was obtained:** total minus test commands. A subtraction, never an
observation.

**Measured**, by replacing the test command with a binary that exits immediately:
ditto's machinery is ~100 ms fixed plus ~7 ms per mutant, and 73–75% of what
remained was `go test` compiling and starting the binary ditto lives in. The
4.5 s had been measured on the *consuming wrapper*, not on ditto.

**Lesson:** a residue is not a measurement. Name what you subtracted from what,
or measure the thing directly.

## The control that saved a sound design

**Symptom:** the instrumented package's suite failed with no mutant selected —
precisely the result that would have killed the whole design, since instrumenting
is not allowed to change the program.

**Control:** build the *uninstrumented* package's test binary and run it the same
way. It failed identically.

**Cause:** `go test` runs tests from the package's own directory; the harness ran
the binary from the module root, so anything read by relative path broke.

**Lesson:** the control is not ceremony. It is the only thing that separates "the
design is broken" from "the harness is broken", and here it was the difference
between shipping and abandoning a design that worked.

## The infinite loop was mine

**Symptom:** a measurement ran for over an hour.

**Claim:** a mutant wrote an infinite loop — flipping `<` to `<=` in a loop
condition does exactly that — and `go test` only kills it at its ten-minute
default.

**Measured**, by timing each step separately: build 198 ms, delete 61 ms, copy
158 ms. The hang was inside the harness's own comment-skipping loop, which tested
the whole prefix from byte zero for a leading `//`, so one leading comment made
the condition true forever.

**Lesson:** when something is slow, time every step before explaining any of them.
The plausible story about the code under test cost an hour.

## A green suite proves the suite agrees with itself

`structured-reports` shipped four slices at 0.98–1.00 mutation score with the gate
green throughout, and eleven pieces of the approved report were missing —
including the entire body of the collision block, the thing the change existed
for. All eleven appeared the moment the binary was built and run.

**Lesson:** when the deliverable is output a human reads, running the binary is
part of the work, not a courtesy. Put it in the task list.

## A sentence is not a guard

`AGENTS.md` stated that ditto stripped `GIT_DIR`, `GIT_INDEX_FILE`,
`GIT_WORK_TREE`, `GIT_OBJECT_DIRECTORY` and `GIT_COMMON_DIR` from what it spawns.
The code never did — the fix had landed in the consuming wrapper.

A decoy repository, aimed at with those variables and left untouched by anything
under test, took the incident from one commit to two and rewrote its `user.name`
to the intruder's, on the first run against unchanged code.

**Lesson:** a documented safety property with no test that refuses without it is a
claim, not a guarantee. Look for the test.

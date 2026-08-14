# Case studies

Measured cases from ditto and dharness. Each one is here because it changed a
decision, and three of them killed a claim the author had already made out loud.

## 1. A number that was never an alternative

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

## 2. The residue that belonged to someone else

**Claim:** roughly 4.5 s of a run is ditto's own startup.

**How it was obtained:** total minus test commands. A subtraction, never an
observation.

**Measured**, by replacing the test command with a binary that exits immediately:
ditto's machinery is ~100 ms fixed plus ~7 ms per mutant, and 73–75% of what
remained was `go test` compiling and starting the binary ditto lives in. The
4.5 s had been measured on the *consuming wrapper*, not on ditto.

**Lesson:** a residue is not a measurement. Name what you subtracted from what,
or measure the thing directly.

## 3. The control that saved a sound design

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

## 4. The infinite loop was mine

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

## 5. A green suite proves the suite agrees with itself

`structured-reports` shipped four slices at 0.98–1.00 mutation score with the gate
green throughout, and eleven pieces of the approved report were missing —
including the entire body of the collision block, the thing the change existed
for. All eleven appeared the moment the binary was built and run.

**Lesson:** when the deliverable is output a human reads, running the binary is
part of the work, not a courtesy. Put it in the task list.

## 6. A sentence is not a guard

`AGENTS.md` stated that ditto stripped `GIT_DIR`, `GIT_INDEX_FILE`,
`GIT_WORK_TREE`, `GIT_OBJECT_DIRECTORY` and `GIT_COMMON_DIR` from what it spawns.
The code never did — the fix had landed in the consuming wrapper.

A decoy repository, aimed at with those variables and left untouched by anything
under test, took the incident from one commit to two and rewrote its `user.name`
to the intruder's, on the first run against unchanged code.

The same decoy answers a second question no inspection of the tree can. Isolation
has two sides: the fixture being measured, and the tool doing the measuring. A
run launched from a checkout with work in it inherits that checkout's addressing,
whatever it points at afterwards. Finding the tree clean says nothing happened
that time; it never says nothing could.

**Lesson:** a documented safety property with no test that refuses without it is a
claim, not a guarantee. Look for the test — and give it somewhere to reach, so
that not reaching it is a measurement rather than a story.

## 7. The green that could not go red

A golden test was extended to run the same fixture a second way, through a new
code path, against the same expected output. It passed on the first run.

Before believing it, a `panic` was put inside that new path. **The golden stayed
green** — so the second run was never reaching it. Two causes, found one after
the other: a decorator in the chain silently did not forward the batch, and the
edit that added the second run to the test file had never applied. A scripted
substitution had matched nothing and reported nothing; `rg -c` on the string
returned zero.

With both fixed, the panic fired, and removing it left the two paths printing the
same bytes.

**Lesson:** a check that has never been seen to fail is not evidence. Break what
it watches, watch it refuse, put it back. And grep for the edit you believe you
made — a substitution that matched nothing looks exactly like a change with no
effect.

## 8. The guard that could not fail

A helper was written to refuse a mutation that changed two places at once. It
rebuilt the file from the range it had computed and compared the result with the
input.

It could never fail. The range is derived from the two files, so rebuilding from
it reproduces the second one by construction. The test written to prove the guard
worked is what exposed it.

**Lesson:** ask whether an assertion could come out false given how its
expectation was built. If the answer is no, it reads like a guarantee and is not
one.

## 9. What the prototype could not see

Three separate prototypes measured a rewrite that gates one mutation per site,
and all three agreed. Wired to the real tool, the first thing that surfaced was
that `comparison`, `comparisoninvert` and `comparisonreplace` all infect the same
expression: one site carries three mutants, which no prototype had ever produced.

**Lesson:** a prototype differs from the shipped code in at least one way that
matters, and the difference is not visible from inside the prototype. Re-measure
through the real thing before carrying a result across.

## 10. The control that was written and not run

An experiment note declared two controls: that the counter must be shown to
count, and that the verdict comparison must be shown to refuse. The first was
run and immediately caught a harness bug. The second was never run — the results
were written and reported with it still sitting in the note.

Comparing 18/9/9 against 18/9/9 twice felt like checking. It is not: two numbers
agreeing says nothing until something shows they could have disagreed. Run
afterwards, with the path under test deliberately broken, killed fell from 9 to 3
and survived rose from 9 to 15.

**Lesson:** writing a control down reads like performing it, and the note is
exactly where that substitution hides. Tick each declared control off against its
evidence before reporting the result it was supposed to protect.

## 11. The set that could not come out empty

A note framed a choice between two fixes as two hypotheses:

    H1: the intervention removes at least N.   Falsified below N.
    H2: it removes fewer than N.               Falsified at N or more.

Exactly one survives, for every measurement that could be taken. The pair was
titled *mutually exclusive, and the measurement kills one* — which is true, and
hides that one is also **saved unconditionally**. H2 is H1's kill line promoted
to a hypothesis, and a threshold that selects an action is a decision rule rather
than a conjecture; it has its own section for that reason.

The same set carried a second entry that answered a different question — a number
needed to *design* the intervention, not a tentative answer to the question the
note existed to settle. It was measured, committed and reported as though
something had been settled.

Both defects have one origin: the note never stated a research question. With
nothing to answer, any statement qualifies as a hypothesis and a partition passes
for a set.

**Lesson:** write the question, then the answers actually considered possible,
then the outcome that would refute all of them. Hypotheses that differ only by a
threshold on one number are one measurement and a decision rule wearing two
hypotheses' clothes.

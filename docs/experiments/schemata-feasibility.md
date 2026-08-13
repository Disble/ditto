# Experiment — can a mutable site be gated without changing the program?

Written before the measurements, same discipline as the other two.

## Why this comes before the contract

The invocation experiment left one way to make a run cheap: compile once and
select the mutant at runtime. That needs ditto to build, which needs the contract
to open — and the contract is not worth opening for a technique that cannot be
made safe. So the question here is not "is it faster", it is **"does the
instrumented program still behave like the program"**.

Getting this wrong is the worst failure this tool has available. A mutation
tester that quietly alters the code it measures reports on a program nobody
wrote, and every verdict it gives is about that other program.

## The candidate rewrite

For a comparison site with mutant id `3` and selector `m`:

    a > b        becomes        (m == 3 && a >= b) || (m != 3 && a > b)

No function call, no generics, no new package-level machinery beyond `m`. It was
chosen over the obvious `gate(a > b, a >= b)` because that form evaluates both
comparisons, and over `gate(a, b)` because that one needs the operand type.

## H1 — the rewrite preserves behaviour and evaluation count

Three properties, each able to kill it:

1. unselected, it answers what `a > b` answered;
2. selected, it answers what `a >= b` would have;
3. **each operand is evaluated exactly once, either way.**

The third decides. An operand evaluated twice changes any expression whose
operands have side effects — `next() > limit()` would advance twice.

*Falsified if any operand is evaluated other than exactly once.*

**Result: holds.** Measured with counting operands across both selector states
and all three orderings. Short-circuiting does the work: unselected, `m == 3` is
false and the mutant comparison is never reached; selected, the mutant answers
and `m != 3` is false, so the original is never reached. Each side evaluates its
operands once and the other side evaluates nothing.

## H2 — almost no real site sits where a constant is required

The rewrite reads a variable, so wherever Go requires a constant expression it
does not compile. Those sites would have to be skipped — and a skipped site is a
mutant that used to be tested and is not any more, which is exactly the
regression this work is not allowed to cause.

*Prediction: zero such sites in ordinary Go. Falsified by any.*

**Result: holds.** Comparison sites sitting under a `const` declaration or an
array length:

| codebase | comparison sites | in constant context |
| --- | --- | --- |
| ditto | 410 | 0 |
| dharness | 1413 | 0 |
| the standard library's `net/http` | 5609 | 0 |

7432 sites, none blocked. The objection is real in principle and absent in
practice.

## What these two do NOT establish

**Only the comparison family.** `<`, `<=`, `>`, `>=`, `==`, `!=` are expressions
that yield a bool, which is why one expression can stand in for two. Ditto ships
fourteen viruses, and several are not that shape at all: `loopbreak` and
`rangebreak` insert statements, `arithmeticassignment` rewrites an assignment.
Each of those needs its own answer, and nothing here gives it.

**The const-context probe is syntactic.** It walks for `const` declarations and
array lengths rather than type-checking, because ditto is stdlib-only and
go/types over a whole module is a larger dependency than the question deserved.
A constant context it does not know about would not appear in these numbers.

**Compiling is not behaving.** Both results are about one expression's semantics
and one syntactic constraint. Whether a whole instrumented package still passes
its own suite unchanged was asked next, and is below.

## H3 — an instrumented package reaches the same verdicts as today

The two earlier results are about one expression. This one is about a real
package: dharness's `internal/jsconfig`, 12 comparison sites, in a throwaway copy
of the module.

Every site is run twice over. Once the way ditto works today — one operator
flipped, the suite run, repeat — and once against a single instrumented file
compiled one time, with the site selected through the environment.

*Falsified by a single disagreement between the two verdict vectors, or by the
instrumented package with nothing selected behaving differently from the
untouched one.*

**Result: holds.**

    baseline, untouched                    : suite passed
    instrumented, nothing selected         : suite passed

    verdicts, one flipped operator (today) : KKKSKKKKSKKK
    verdicts, instrumented and selected    : KKKSKKKKSKKK

    compilations : today 12, instrumented 1
    wall clock   : today 8.22 s, instrumented 1.77 s

Ten killed, two survived, and the same two survived both ways. The instrumented
build reaches today's verdicts, and it reaches them from one compilation instead
of twelve.

The 4.6× is this package's number and should not be carried anywhere else: the
toll removed per mutant is fixed, so a package whose suite does real work keeps
that work and shows a smaller ratio. What generalises is the counter — **one
compilation instead of N** — not the clock.

### Three defects found on the way, all in the harness

None of them were in the rewrite, and each was found by a control rather than by
reading the code:

- The tool hung for over an hour. Suspected an infinite loop in a mutant —
  flipping `<` to `<=` in a loop condition writes one — but timing each step
  separately showed the copy at 158 ms and the hang inside a comment-skipping
  loop of the harness's own. **The infinite loop was mine.**
- The instrumented file did not compile: the mutated text was taken by slicing a
  buffer whose length had changed, because `<` becomes `<=`, which quietly ate
  the expression's last character.
- The instrumented package's suite then failed with nothing selected, which
  looked exactly like the instrumentation having broken it. The control settled
  it: the **uninstrumented** binary failed the same way, because the harness ran
  it from the module root and `go test` runs from the package's directory.

The third is the one worth keeping. It produced the precise appearance of the
result that would have killed the design, and only a control that removed the
instrumentation could tell the difference.

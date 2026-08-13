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
its own suite unchanged is the golden test's question, and it has not been asked
of an instrumented build yet.

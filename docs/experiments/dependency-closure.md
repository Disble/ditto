# Experiment — can the observability closure replace the whole package scope?

Written before the measurement on 2026-09-18, at revision `2deb438`.

## The research question

**To what extent** does running only the package test binaries whose dependency closure contains the mutated package reduce package executions per selection, without changing a single verdict, over seven packages and four selections in a disposable module on 2026-09-18?

| Element | This question's |
| --- | --- |
| Interrogative phrase | To what extent |
| Variable, the counter that moves | Package test-binary executions per selection, and the observable-package count per package |
| Population, unit of analysis | Four selections against one mutated package; one closure table over all seven |
| Space and time | Revision `2deb438`, a disposable seven-package module, 2026-09-18 |

**FINER** — Feasible: `go list -deps -test -json ./...` reports each test binary's dependency set, and the runner already starts one binary per package per selection. · Interesting: the module path costs `selections × packages` and loses to the ordinary path above roughly sixteen packages, so whether that factor can be divided is the decision. · Novel: `gobuildrunner` runs every package and nothing computes which of them can observe the mutation. · Ethical: throwaway copies only. · Relevant: an exact reduction with unchanged verdicts selects the next core change; any verdict change rejects it.

**PICOT** — P: four selections against one mutated package. · I: run only the binaries whose dependency closure contains that package. · **C**: every package binary, which is what ships today. · **O**: package executions per selection, the ordered verdicts, and the observable-package count per package. · T: one run per mode, bound to the revision above.

## Why the reduction is provable rather than heuristic

A package test binary compiles its own package plus the transitive closure of what it imports. If the mutated package is not in that closure, no code in the binary can refer to anything the mutation changed, so no test in it can observe the mutation. That is a statement about what the compiler contains, not a guess about what a test probably covers.

The closure must include **test imports**. A package's external test file lives in a separate package that imports the subject, so a closure built from `Imports` alone would be too narrow. `go list -deps -test` is used for exactly that reason, and the fixture below is built so a test-only import would be missed if the flag were dropped.

**This is not the defect that was just fixed.** The package-only path ran the mutated package and nothing else, so a mutant only a dependent package could kill survived. This runs the mutated package **plus every package that can reach it** — strictly more than the old path, and exactly the set that can observe.

## Hypotheses, and what kills each one

**H1 — the closure excludes the packages that cannot observe the mutation.** Of seven test packages, exactly three — the mutated package, a dependent, and a dependent of the dependent — have the mutated package in their closure; four independent packages do not.
*Falsified if the count is anything other than three, or if an independent package appears in the closure.*

**H2 — the restricted mode lands every verdict the unrestricted mode lands.** Four selections produce identical ordered verdicts, including the sentinel that only a dependent package kills.
*Falsified if any verdict differs, and in particular if the sentinel survives.*

**H3 — the restriction cuts executions per selection by the ratio the closure predicts.** From three of seven: ten executions for four selections instead of twenty-two, counting the baseline as one selection.
*Falsified if the restricted count is not ten, or the unrestricted count is not twenty-two.*

**What would refute all of them:** the unrestricted mode does not reproduce the verdicts the module runner itself reports for the same fixture. That means the experiment is measuring its own harness rather than the runner, and no conclusion follows.

## Decision rule, fixed in advance

- All three corroborated → the closure is the next core change, implemented with the closure computed once per release and the sentinel kept as its guard.
- H2 refuted → the restriction is unsafe, the cost factor stays, and the module path is documented as a small-module optimization.
- H1 or H3 refuted but H2 corroborated → the mechanism is sound and the arithmetic is wrong; re-derive before implementing.

## Method

The fixture is a seven-package module: `base` holds the mutable comparison sites, `mid` imports `base`, `top` imports `mid`, and four `island` packages import nothing local. The signal travels through `DITTO_MUTANT`, which is the variable the runner already sets, so no instrumentation is needed to answer this question and the fixture stays legible.

Each package records its own execution by appending to a log file, which is what makes "the island did not run" observable rather than inferred from a total.

The closure is read from `go list -deps -test -json ./...` — one Go command, the toolchain's own answer, nothing re-implemented.

## Results

Measured 2026-09-18 from a complete `.git`-free disposable copy, Go 1.27, windows/amd64. The experiment test is tagged `experiment` and is not part of the default suite.

The observability table, read from `go list -deps -test -json ./...`:

| Package | Observes |
| --- | --- |
| base | — (imports nothing local) |
| mid | base |
| top | base, mid |
| island1 - island4 | — |

Packages whose binary can observe a mutation in `base`, the mutated package included: **base, mid, top — three of seven.**

| Mode | Executions | Verdicts |
| --- | ---: | --- |
| Unrestricted (seven packages) | 35 | killed, killed, killed, survived |
| Restricted (three packages) | 15 | killed, killed, killed, survived |

The three islands are excluded, and `top` is included even though it never names `base` — it reaches it through `mid`, which is the case a closure built from direct imports alone would miss.

### Controls

**The shipped runner agrees with the unrestricted mode — passed.** `gobuildrunner.NewModuleScope` started 35 package binaries over the same fixture and the same five runs, matching the experiment's unrestricted count exactly. Without that, the numbers above would describe the harness rather than the runner.

**The fixture is green before any mutant is selected — passed.** The unselected baseline runs clean in both modes, and the experiment fails rather than scoring if it does not.

**Every package records its own execution — passed.** Each selection asserts that exactly the packages in scope ran, by reading a log the packages write themselves, so "the island did not run" is observed rather than inferred from a total that could hide it.

**Two harness defects were found and fixed before the numbers were trusted.** The first run reported `base` failing on the unselected baseline, because the shared environment helper ignored its selector argument and hardcoded mutant `1`; the second omitted the mutated package from its own closure. Both were the instrument, not the design, and both are recorded because a harness that cannot report the wrong answer cannot report the right one either.

### H1 corroborated

Exactly three of seven packages have the mutated package in their closure, and no island appears in it.

### H2 corroborated

Identical ordered verdicts in both modes, and selection 2 — the sentinel that only `mid` kills — is killed under the restriction. A closure that dropped `mid` would have reported it survived, which is the package-only defect returning.

### H3 corroborated

Fifteen executions against a prediction of fifteen, and thirty-five against thirty-five. The restriction removed twenty of thirty-five executions, a 57% reduction on a fixture where four of seven packages are unreachable.

## Verdicts: 3 of 3

## Conclusion

The decision rule selects the first outcome. Running only the packages whose test binary compiles the mutated package is safe on this fixture and divides the execution count by the ratio the closure predicts. On the ten-package fixture that made the module path lose its gain, a mutated package observed by three of ten would cost about a third of what it costs today.

That projection is arithmetic from two measured facts — the closure ratio here and the ten-package ratio in `gated-through-the-binary.md` — and it is labelled a projection rather than a result. It is the number the implementation has to reproduce through the shipped binary before it is claimed.

## What this does NOT establish

The cost of computing the closure is not measured: it is one `go list` per release, not per mutant, and it is paid on top of the discovery the runner already does.

A repository whose packages form a single chain is not represented. There the closure is every package, the factor does not divide, and the module path keeps the shape that loses above sixteen packages.

And nothing here is implemented. This measures the ceiling, so the production shape follows from the number rather than from the plan — the same order that let the earlier experiments kill three ideas before any code changed.

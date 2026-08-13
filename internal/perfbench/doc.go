// Package perfbench holds ditto's performance contract.
//
// Performance is this library's primary metric: it exists to make mutation
// testing cheap enough to run inside a TDD loop, so a change that makes it
// slower has failed at the only thing it was for. That claim is worthless
// unless it can fail, so this package states it as numbers a test can reject.
//
// The contract is split by how trustworthy each kind of measurement is.
//
// Exact counters are the hard gate. How many files a mutant materializes, how
// many times a source file is parsed, how many mutants a scope produces — these
// are integers, identical on every machine, and a change in one is always
// meaningful. They live in perf/baseline.json and are enforced by ordinary
// tests, so the verdict arrives as an exit code and never as prose.
//
// Wall clock is not a gate. Measured on the machine this package was written
// on, the same workload varied by more than fifty percent between runs. A
// threshold tight enough to catch a real regression would fire constantly on
// noise, and a gate that cries wolf is one people learn to ignore. Benchmarks
// here report ns/op for humans to read and allocation counts for machines to
// compare, because allocations track work done without tracking the weather.
//
// The counters ratchet in both directions. A number that grows is a
// regression. A number that shrinks is an improvement that must be written
// down, so it cannot be silently given back later.
package perfbench

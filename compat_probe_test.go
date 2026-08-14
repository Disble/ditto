package ditto_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// docs/experiments/backward-compatibility.md.
//
// Every measurement so far compared two paths of the same revision. This one
// compares the revision before the branch with the revision after it, on the
// path that exists in both.

// baseRevision is what the branch was cut from. Gated() does not exist there,
// which is also control 2: a tree that has it is not the base.
const baseRevision = "dfd8ed4"

const compatSource = `package calc

func Grade(score int) string {
	if score > 50 {
		return "pass"
	}

	return "fail"
}

func Bonus(score int) int {
	if score >= 90 {
		return 10
	}

	return 0
}

func Label(kind int) string {
	switch kind {
	case 1:
		return "one"
	case 2:
		return "two"
	}

	return "other"
}

func Join(left, right string) string {
	return left + "-" + right
}

func First(values []string) string {
	return values[0]
}

func Report(values []string, limit int) string {
	found := len(values)

	if found > 0 && limit > 0 {
		return "some"
	}

	return "none"
}

func FirstNonEmpty(values []string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}
`

const compatTests = `package calc_test

import (
	"testing"

	"compat/calc"
)

func TestGrade(t *testing.T) {
	if calc.Grade(80) != "pass" {
		t.Fatal("80 should pass")
	}

	if calc.Grade(10) != "fail" {
		t.Fatal("10 should fail")
	}
}

func TestLabel(t *testing.T) {
	if calc.Label(1) != "one" {
		t.Fatal("1 should be one")
	}
}

func TestJoin(t *testing.T) {
	if calc.Join("a", "b") != "a-b" {
		t.Fatal("a and b should join")
	}
}
`

// compatMutation uses only options that exist on both revisions. Gated() is not
// among them, which is the point: this compares the path that already existed.
const compatMutation = `//go:build mutation

package compat_test

import (
	"testing"

	"github.com/Disble/ditto"
)

func TestMutation(t *testing.T) {
	ditto.Release(t,
		ditto.WithRepositoryRoot("."),
		ditto.WithTestCommand("go test -count=1 ./calc"),
		ditto.WithMinimumThreshold(0.0),
	)
}
`

func TestBackwardCompatibility(t *testing.T) {
	if os.Getenv("DITTO_PROBE") != "1" {
		t.Skip("set DITTO_PROBE=1 to run the probes; see docs/experiments/backward-compatibility.md")
	}

	base := materialiseBase(t)

	// Control 3: green before anything is mutated.
	if output, err := compatSuite(t); err != nil {
		t.Fatalf("the fixture is not green, so no verdict below means anything\n%s", output)
	}

	before := runAgainst(t, base, "before")
	after := runAgainst(t, moduleRoot(t), "after")

	// Control 2, the half that matters: each run proves which revision produced
	// it. Two revisions that are secretly the same agree about everything.
	if !strings.Contains(before.output, baseMarker) {
		t.Fatalf("the run that claims to be %s does not carry its marker, so it is "+
			"not the base and nothing below is a comparison\n%s", baseRevision, before.output)
	}

	if strings.Contains(after.output, baseMarker) {
		t.Fatalf("the run that claims to be HEAD carries the base's marker\n%s", after.output)
	}

	t.Logf("%s : total %d, killed %d, survived %d", baseRevision, before.total, before.killed, before.survived)
	t.Logf("HEAD    : total %d, killed %d, survived %d", after.total, after.killed, after.survived)

	if before.total != after.total {
		t.Fatalf("H1 is falsified: %s generates %d mutants and HEAD generates %d, "+
			"so no verdict comparison below would mean anything",
			baseRevision, before.total, after.total)
	}

	onlyBefore := missingFrom(survivorsOf(t, before), survivorsOf(t, after))
	onlyAfter := missingFrom(survivorsOf(t, after), survivorsOf(t, before))

	for _, label := range onlyBefore {
		t.Logf("  survived at %s and is killed at HEAD: %s", baseRevision, label)
	}

	for _, label := range onlyAfter {
		t.Logf("  killed at %s and survives at HEAD  : %s", baseRevision, label)
	}

	if len(onlyBefore)+len(onlyAfter) == 0 {
		t.Logf("H2 holds: no label changed column")

		return
	}

	t.Errorf("H2 is falsified: %d labels changed column between the revisions",
		len(onlyBefore)+len(onlyAfter))
}

// materialiseBase writes the base revision into a throwaway directory with
// `git archive`, which touches no checkout and creates no linked worktree.
func materialiseBase(t *testing.T) string {
	t.Helper()

	root := moduleRoot(t)
	into := t.TempDir()
	archive := filepath.Join(into, "base.tar")

	mustRun(t, root, "archiving "+baseRevision, "git", "archive", "--format=tar",
		"--output="+archive, baseRevision)

	// The name is relative on purpose: tar reads a leading `C:` as a remote host
	// and refuses, which on Windows is every absolute path there is.
	mustRun(t, into, "extracting "+baseRevision, "tar", "-xf", filepath.Base(archive))

	// Control 2: a tree carrying Gated is not the base.
	options, err := os.ReadFile(filepath.Join(into, "options.go"))
	if err != nil {
		t.Fatalf("reading the base tree's options.go: %v", err)
	}

	if strings.Contains(string(options), "func Gated") {
		t.Fatalf("the tree at %s already has Gated, so it is not the revision this "+
			"branch was cut from", baseRevision)
	}

	markBase(t, into)

	return into
}

// baseMarker is stamped into the base copy's banner so the run that claims to be
// the base has to prove it in its own output.
//
// Without it the probe rests on a `replace` line resolving the way it reads,
// and two revisions that silently turn out to be the same one agree about
// everything — which looks exactly like backward compatibility.
const baseMarker = "Releasing DITTO-AT-BASE…"

func markBase(t *testing.T, into string) {
	t.Helper()

	path := filepath.Join(into, "release.go")

	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the base tree's release.go: %v", err)
	}

	marked := strings.Replace(string(source), "Releasing Ditto…", baseMarker, 1)
	if marked == string(source) {
		t.Fatalf("the base tree's banner is not where this control expects it, "+
			"so nothing would prove which revision ran:\n%s", path)
	}

	// Written here rather than through the shared helper: the path comes from a
	// directory this test made, and threading it through writeFile makes every
	// other caller of that helper look tainted.
	//nolint:gosec // the path is t.TempDir() plus a fixed name
	if err := os.WriteFile(path, []byte(marked), 0o600); err != nil {
		t.Fatalf("stamping the base tree: %v", err)
	}
}

func compatSuite(t *testing.T) (string, error) {
	t.Helper()

	project := t.TempDir()

	writeFile(t, filepath.Join(project, "go.mod"), "module compat\n\ngo 1.25\n")
	writeFile(t, filepath.Join(project, "calc", "calc.go"), compatSource)
	writeFile(t, filepath.Join(project, "calc", "calc_test.go"), compatTests)

	output, err := command(t, project, "go", "test", "-count=1", "./calc").CombinedOutput()

	return string(output), err
}

func runAgainst(t *testing.T, dittoRoot, which string) release {
	t.Helper()

	project := t.TempDir()

	writeFile(t, filepath.Join(project, "go.mod"), "module compat\n\ngo 1.25\n\n"+
		"require github.com/Disble/ditto v0.0.0\n\n"+
		"replace github.com/Disble/ditto => "+filepath.ToSlash(dittoRoot)+"\n")
	writeFile(t, filepath.Join(project, "doc.go"), "// Package compat is a fixture.\npackage compat\n")
	writeFile(t, filepath.Join(project, "mutation_test.go"), compatMutation)
	writeFile(t, filepath.Join(project, "calc", "calc.go"), compatSource)
	writeFile(t, filepath.Join(project, "calc", "calc_test.go"), compatTests)

	mustRun(t, project, which+": resolving dependencies", "go", "mod", "tidy")

	binary := filepath.Join(project, "mutation.test")
	mustRun(t, project, which+": building the release", "go", "test", "-c", "-tags=mutation", "-o", binary, ".")

	output, _ := command(t, project, binary, "-test.run", "TestMutation", "-test.count=1").CombinedOutput()

	return parseRelease(t, string(output))
}

// repositoryMutation scopes a release to one file of a real repository, the way
// dharness's wrapper does, so the comparison runs over the code the spoiled kill
// was actually seen in rather than over a fixture built to be convenient.
const repositoryMutation = `//go:build mutation

package dharness_test

import (
	"testing"

	"github.com/Disble/ditto"
)

func TestWriterMutation(t *testing.T) {
	ditto.Release(t,
		ditto.WithRepositoryRoot("."),
		ditto.WithTestCommand("go test -count=1 ./internal/setup"),
		ditto.WithMinimumThreshold(0.0),
		ditto.WithChangedRanges(map[string][]ditto.Range{
			"internal/setup/writer.go": {},
		}),
	)
}
`

// TestBackwardCompatibilityOnTheRepository runs the same comparison over the
// file where the verdict was seen to move: internal/setup/writer.go, the case
// gated-gain-slow.md opened this whole line of work with.
//
// Point DITTO_COMPAT_REPO at a repository. It is copied, never mutated in place.
func TestBackwardCompatibilityOnTheRepository(t *testing.T) {
	if os.Getenv("DITTO_PROBE") != "1" {
		t.Skip("set DITTO_PROBE=1 to run the probes; see docs/experiments/backward-compatibility.md")
	}

	repository := os.Getenv("DITTO_COMPAT_REPO")
	if repository == "" {
		t.Skip("set DITTO_COMPAT_REPO to a repository to copy and mutate")
	}

	base := materialiseBase(t)

	before := runRepositoryAgainst(t, repository, base, "before")
	after := runRepositoryAgainst(t, repository, moduleRoot(t), "after")

	if !strings.Contains(before.output, baseMarker) {
		t.Fatalf("the run that claims to be %s does not carry its marker\n%s",
			baseRevision, before.output)
	}

	t.Logf("%s : total %d, killed %d, survived %d", baseRevision, before.total, before.killed, before.survived)
	t.Logf("HEAD    : total %d, killed %d, survived %d", after.total, after.killed, after.survived)

	if before.total != after.total {
		t.Fatalf("the population moved: %d against %d", before.total, after.total)
	}

	onlyBefore := missingFrom(survivorsOf(t, before), survivorsOf(t, after))
	onlyAfter := missingFrom(survivorsOf(t, after), survivorsOf(t, before))

	for _, label := range onlyBefore {
		t.Logf("  survived at %s and is killed at HEAD: %s", baseRevision, label)
	}

	for _, label := range onlyAfter {
		t.Logf("  killed at %s and survives at HEAD  : %s", baseRevision, label)
	}

	if len(onlyBefore)+len(onlyAfter) != 0 {
		t.Errorf("%d labels changed column between the revisions",
			len(onlyBefore)+len(onlyAfter))
	}
}

func runRepositoryAgainst(t *testing.T, repository, dittoRoot, which string) release {
	t.Helper()

	project := t.TempDir()

	copyRepository(t, repository, project)

	manifest, err := os.ReadFile(filepath.Join(project, "go.mod"))
	if err != nil {
		t.Fatalf("%s: reading the copy's go.mod: %v", which, err)
	}

	//nolint:gosec // the same directory
	err = os.WriteFile(filepath.Join(project, "go.mod"),
		append(manifest, []byte("\nreplace github.com/Disble/ditto => "+
			filepath.ToSlash(dittoRoot)+"\n")...), 0o600)
	if err != nil {
		t.Fatalf("%s: pointing the copy at ditto: %v", which, err)
	}

	writeFile(t, filepath.Join(project, "writer_mutation_test.go"), repositoryMutation)

	mustRun(t, project, which+": resolving dependencies", "go", "mod", "tidy")

	binary := filepath.Join(project, "mutation.test")
	mustRun(t, project, which+": building the release", "go", "test", "-c", "-tags=mutation", "-o", binary, ".")

	output, _ := command(t, project, binary, "-test.run", "TestWriterMutation", "-test.count=1").CombinedOutput()

	return parseRelease(t, string(output))
}

// copyRepository mirrors a tree through tar rather than through the golden
// test's walker, which taint analysis then flags for every other caller. Both
// names are relative because tar reads a leading `C:` as a remote host.
func copyRepository(t *testing.T, from, to string) {
	t.Helper()

	archive := filepath.Join(to, "repo.tar")

	mustRun(t, from, "archiving the repository", "git", "archive", "--format=tar",
		"--output="+archive, "HEAD")

	// Extracted from inside the destination with a relative name: tar reads a
	// leading `C:` as a remote host, in `-C` as well as in the archive's name.
	mustRun(t, to, "extracting the repository", "tar", "-xf", "repo.tar")

	if err := os.Remove(archive); err != nil {
		t.Fatalf("removing the staging archive: %v", err)
	}
}

// gainCounter is appended to the fixture package so every run of the test
// command leaves a line behind. Invocations are counted from outside ditto, not
// read out of its own bookkeeping, which is what makes the number evidence
// rather than a restatement.
const gainCounter = `package jsconfig_test

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if log := os.Getenv("DITTO_RUN_LOG"); log != "" {
		file, err := os.OpenFile(log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err == nil {
			_, _ = file.WriteString("run\n")
			_ = file.Close()
		}
	}

	os.Exit(m.Run())
}
`

const gainMutation = `//go:build mutation

package dharness_test

import (
	"os"
	"testing"

	"github.com/Disble/ditto"
)

func TestGainMutation(t *testing.T) {
	options := []ditto.Option{
		ditto.WithRepositoryRoot("."),
		ditto.WithTestCommand("go test -count=1 ./internal/jsconfig"),
		ditto.WithMinimumThreshold(0.0),
		ditto.WithChangedRanges(map[string][]ditto.Range{
			"internal/jsconfig/jsconfig.go": {},
		}),
	}

	if os.Getenv("GAIN_GATED") == "1" {
		options = append(options, ditto.Gated())
	}

	ditto.Release(t, options...)
}
`

type gain struct {
	release
	invocations int
	wall        time.Duration
}

// TestGainAfterTheFixes is docs/experiments/gain-after-the-fixes.md.
//
// Point DITTO_COMPAT_REPO at a repository holding internal/jsconfig.
func TestGainAfterTheFixes(t *testing.T) {
	if os.Getenv("DITTO_PROBE") != "1" {
		t.Skip("set DITTO_PROBE=1 to run the probes; see docs/experiments/gain-after-the-fixes.md")
	}

	repository := os.Getenv("DITTO_COMPAT_REPO")
	if repository == "" {
		t.Skip("set DITTO_COMPAT_REPO to a repository to copy and mutate")
	}

	// One warm-up discarded, then three rounds with the two paths rotated.
	runGain(t, repository, false, "warm-up")

	for round := 1; round <= 3; round++ {
		gatedFirst := round%2 == 0

		first := runGain(t, repository, gatedFirst, "round")
		second := runGain(t, repository, !gatedFirst, "round")

		report(t, round, first, gatedFirst)
		report(t, round, second, !gatedFirst)
	}
}

func report(t *testing.T, round int, got gain, gated bool) {
	t.Helper()

	path := "ordinary"
	if gated {
		path = "gated"
	}

	t.Logf("round %d | %-8s | total %3d | killed %3d | survived %2d | invocations %3d | %v",
		round, path, got.total, got.killed, got.survived, got.invocations, got.wall.Round(time.Millisecond))
}

func runGain(t *testing.T, repository string, gated bool, which string) gain {
	t.Helper()

	project := t.TempDir()

	copyRepository(t, repository, project)
	pointAtDitto(t, project, moduleRoot(t), which)

	writeFile(t, filepath.Join(project, "internal", "jsconfig", "zz_gain_counter_test.go"), gainCounter)
	writeFile(t, filepath.Join(project, "gain_mutation_test.go"), gainMutation)

	mustRun(t, project, which+": resolving dependencies", "go", "mod", "tidy")

	binary := filepath.Join(project, "mutation.test")
	mustRun(t, project, which+": building the release", "go", "test", "-c", "-tags=mutation", "-o", binary, ".")

	log := filepath.Join(project, "invocations.log")

	run := command(t, project, binary, "-test.run", "TestGainMutation", "-test.count=1")
	run.Env = append(run.Env, "DITTO_RUN_LOG="+log)

	if gated {
		run.Env = append(run.Env, "GAIN_GATED=1")
	}

	started := time.Now()
	output, _ := run.CombinedOutput()
	elapsed := time.Since(started)

	return gain{
		release:     parseRelease(t, string(output)),
		invocations: linesIn(t, log),
		wall:        elapsed,
	}
}

func linesIn(t *testing.T, path string) int {
	t.Helper()

	//nolint:gosec // a file inside a directory this test made
	content, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	return strings.Count(string(content), "\n")
}

func pointAtDitto(t *testing.T, project, dittoRoot, which string) {
	t.Helper()

	//nolint:gosec // a directory this test just made
	manifest, err := os.ReadFile(filepath.Join(project, "go.mod"))
	if err != nil {
		t.Fatalf("%s: reading the copy's go.mod: %v", which, err)
	}

	//nolint:gosec // the same directory
	err = os.WriteFile(filepath.Join(project, "go.mod"),
		append(manifest, []byte("\nreplace github.com/Disble/ditto => "+
			filepath.ToSlash(dittoRoot)+"\n")...), 0o600)
	if err != nil {
		t.Fatalf("%s: pointing the copy at ditto: %v", which, err)
	}
}

package ditto_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// H1 and H2 of docs/experiments/changed-scope.md, measured through a whole
// release rather than through a seam.
//
// Each variant is a scratch module written into t.TempDir() and destroyed with
// it. Nothing is copied from a checkout and nothing survives the run, which is
// AGENTS.md's rule applied to ditto's own suite.

const scratchSource = `package calc

func Grade(score int) string {
	if score > 50 {
		return "pass"
	}

	return "fail"
}
`

// scratchIndexed adds a literal index, which decrementing turns negative and the
// compiler refuses. It is what variant C needs and nothing else uses.
const scratchIndexed = `
func First(values []string) string {
	return values[0]
}
`

const scratchTests = `package calc_test

import (
	"testing"

	"scratch/calc"
)

func TestGradeAbove(t *testing.T) {
	if calc.Grade(80) != "pass" {
		t.Fatal("80 should pass")
	}
}

func TestGradeBelow(t *testing.T) {
	if calc.Grade(10) != "fail" {
		t.Fatal("10 should fail")
	}
}
`

const scratchRedTest = `
func TestAlreadyRed(t *testing.T) {
	t.Fatal("this suite fails before anything is mutated")
}
`

const scratchMutation = `//go:build mutation

package scratch_test

import (
	"os"
	"testing"

	"github.com/Disble/ditto"
)

func TestMutation(t *testing.T) {
	options := []ditto.Option{
		ditto.WithRepositoryRoot("."),
		ditto.WithTestCommand("go test -count=1 ./calc"),
		ditto.WithMinimumThreshold(0.0),
	}

	if os.Getenv("SCRATCH_GATED") == "1" {
		options = append(options, ditto.Gated())
	}

	ditto.Release(t, options...)
}
`

type release struct {
	total    int
	killed   int
	survived int
	output   string
}

// TestChangedScope is a probe, not a guard, and it is skipped without being
// asked for by name.
//
// Two reasons, and the second matters more. It builds scratch modules and runs
// whole releases, which does not fit the gate's budget. And its assertions
// record that a defect is present: the day the baseline is read and the run
// refuses, this turns red for having been fixed. A measurement frozen as a
// regression test fails in the wrong direction.
func TestChangedScope(t *testing.T) {
	if os.Getenv("DITTO_PROBE") != "1" {
		t.Skip("set DITTO_PROBE=1 to run the probes; see docs/experiments/changed-scope.md")
	}

	t.Run("control: a green suite reports both outcomes", func(t *testing.T) {
		got := runScratch(t, scratchSource, scratchTests, true)

		if got.killed == 0 || got.survived == 0 {
			t.Fatalf("the probe cannot tell outcomes apart: %d killed, %d survived\n%s",
				got.killed, got.survived, got.output)
		}

		t.Logf("gated over a green suite: total %d, killed %d, survived %d",
			got.total, got.killed, got.survived)
	})

	t.Run("H1: a red baseline is scored as a perfect run", func(t *testing.T) {
		source, tests := scratchSource, scratchTests+scratchRedTest

		// Control 2: the fault is confirmed present before anything is blamed on
		// it. The suite must already fail with nothing mutated.
		if output, err := scratchSuite(t, source, tests); err == nil {
			t.Fatalf("the red fixture is not red, so nothing below means anything\n%s", output)
		}

		got := runScratch(t, source, tests, true)

		t.Logf("gated over a RED suite: total %d, killed %d, survived %d",
			got.total, got.killed, got.survived)

		if strings.Contains(strings.ToLower(got.output), "baseline") {
			t.Errorf("H1 is falsified: the output names the baseline\n%s", got.output)
		}

		if got.survived != 0 {
			t.Errorf("H1 is falsified: %d survived on a suite that fails unmutated",
				got.survived)
		}
	})

	t.Run("H2: the fallback reports the same false kill", func(t *testing.T) {
		source := scratchSource + scratchIndexed

		// Control 2 again: the mutation this variant exists for must really fail
		// to compile. Written by hand rather than assumed from a virus's name.
		decremented := strings.Replace(source, "values[0]", "values[-1]", 1)
		if output, err := scratchSuite(t, decremented, scratchTests); err == nil {
			t.Fatalf("the non-compiling mutant compiles, so this variant tests nothing\n%s", output)
		}

		gated := runScratch(t, source, scratchTests, true)
		ordinary := runScratch(t, source, scratchTests, false)

		t.Logf("gated   : total %d, killed %d, survived %d", gated.total, gated.killed, gated.survived)
		t.Logf("ordinary: total %d, killed %d, survived %d", ordinary.total, ordinary.killed, ordinary.survived)

		if gated.killed != ordinary.killed {
			t.Errorf("H2 is falsified: gated killed %d, ordinary killed %d",
				gated.killed, ordinary.killed)
		}
	})
}

// scratchSuite runs the fixture's own suite with nothing mutated, which is what
// turns "this variant carries the fault" from an assumption into evidence.
func scratchSuite(t *testing.T, source, tests string) (string, error) {
	t.Helper()

	project := t.TempDir()

	writeFile(t, filepath.Join(project, "go.mod"), "module scratch\n\ngo 1.25\n")
	writeFile(t, filepath.Join(project, "calc", "calc.go"), source)
	writeFile(t, filepath.Join(project, "calc", "calc_test.go"), tests)

	output, err := command(t, project, "go", "test", "-count=1", "./calc").CombinedOutput()

	return string(output), err
}

func runScratch(t *testing.T, source, tests string, gated bool) release {
	t.Helper()

	project := t.TempDir()
	root := moduleRoot(t)

	writeFile(t, filepath.Join(project, "go.mod"), "module scratch\n\ngo 1.25\n\n"+
		"require github.com/Disble/ditto v0.0.0\n\n"+
		"replace github.com/Disble/ditto => "+filepath.ToSlash(root)+"\n")
	writeFile(t, filepath.Join(project, "doc.go"), "// Package scratch is a fixture.\npackage scratch\n")
	writeFile(t, filepath.Join(project, "mutation_test.go"), scratchMutation)
	writeFile(t, filepath.Join(project, "calc", "calc.go"), source)
	writeFile(t, filepath.Join(project, "calc", "calc_test.go"), tests)

	mustRun(t, project, "resolving dependencies", "go", "mod", "tidy")

	binary := filepath.Join(project, "mutation.test")
	mustRun(t, project, "building the release", "go", "test", "-c", "-tags=mutation", "-o", binary, ".")

	run := command(t, project, binary, "-test.run", "TestMutation", "-test.count=1", "-test.v")
	if gated {
		run.Env = append(run.Env, "SCRATCH_GATED=1")
	}

	output, _ := run.CombinedOutput()

	return parseRelease(t, string(output))
}

func mustRun(t *testing.T, project, what string, name string, args ...string) {
	t.Helper()

	output, err := command(t, project, name, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("%s: %v\n%s", what, err, output)
	}
}

var releaseCounts = regexp.MustCompile(`(Total|Killed|Survived)[^0-9]+(\d+)`)

func parseRelease(t *testing.T, output string) release {
	t.Helper()

	parsed := release{output: output}

	for _, match := range releaseCounts.FindAllStringSubmatch(output, -1) {
		count, err := strconv.Atoi(match[2])
		if err != nil {
			continue
		}

		switch match[1] {
		case "Total":
			parsed.total = count
		case "Killed":
			parsed.killed = count
		case "Survived":
			parsed.survived = count
		}
	}

	if parsed.total == 0 {
		t.Fatalf("no report in the output:\n%s", output)
	}

	return parsed
}

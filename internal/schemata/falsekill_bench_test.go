package schemata_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/gosourcefile"
	"github.com/Disble/ditto/viruses"
	"github.com/Disble/ditto/viruses/arithmetic"
	"github.com/Disble/ditto/viruses/arithmeticassignment"
	"github.com/Disble/ditto/viruses/arithmeticassignmentinvert"
	"github.com/Disble/ditto/viruses/bitwise"
	"github.com/Disble/ditto/viruses/comparison"
	"github.com/Disble/ditto/viruses/comparisoninvert"
	"github.com/Disble/ditto/viruses/comparisonreplace"
	"github.com/Disble/ditto/viruses/floatdecrement"
	"github.com/Disble/ditto/viruses/floatincrement"
	"github.com/Disble/ditto/viruses/integerdecrement"
	"github.com/Disble/ditto/viruses/integerincrement"
	"github.com/Disble/ditto/viruses/loopbreak"
	"github.com/Disble/ditto/viruses/loopcondition"
	"github.com/Disble/ditto/viruses/rangebreak"
	"github.com/stretchr/testify/require"
)

// TestCountsMutantsThatCannotBuild measures how many of the mutants ditto scores
// as killed were never run at all.
//
// A mutation that does not compile makes the test command exit non-zero, and a
// failing command is how ditto recognises a mutant a test caught. Every one of
// those inflates the score, which is the number people act on. See
// docs/experiments/false-kills.md.
//
// Point DITTO_FALSEKILL_ROOT at a THROWAWAY COPY. This rewrites files in place.
func TestCountsMutantsThatCannotBuild(t *testing.T) {
	root := os.Getenv("DITTO_FALSEKILL_ROOT")
	if root == "" {
		t.Skip("set DITTO_FALSEKILL_ROOT to a throwaway copy of a repository")
	}

	mutants, broken := 0, 0
	counted := &tallies{
		reasons:  map[string]int{},
		pairs:    map[failure]int{},
		examples: map[failure]string{},
		produced: map[string]int{},
		broke:    map[string]int{},
	}
	perFile := map[string]string{}

	//nolint:gosec // rewriting the tree the caller named is what this probe is for
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !mutable(entry.Name()) {
			return err
		}

		// DITTO_FALSEKILL_ONLY narrows the walk, so a control can be run over one
		// file whose mutant count is already known from a real release.
		if only := os.Getenv("DITTO_FALSEKILL_ONLY"); only != "" &&
			!strings.Contains(filepath.ToSlash(path), only) {
			return nil
		}

		here, brokenHere := countOne(t, root, path, counted)
		mutants += here
		broken += brokenHere

		if here > 0 {
			perFile[strings.TrimPrefix(filepath.ToSlash(path), filepath.ToSlash(root)+"/")] = fmt.Sprintf("%d of %d", brokenHere, here)
		}

		return nil
	})
	require.NoError(t, err)

	report(t, mutants, broken, counted, perFile)
}

// failure is one compiler complaint attributed to the virus that produced it.
//
// The message on its own is what this probe recorded first, and a plan was built
// on reading the virus out of the message by eye — 42 index failures were called
// integerdecrement's without anything ever counting them. This pairing is what
// removes the reading.
type failure struct {
	virus string
	cause string
}

// tallies are exact integer counters and nothing else. produced counts every
// mutant a virus wrote; broke counts the ones that then failed to build, so a
// refusal can be weighed against what it would also throw away.
type tallies struct {
	reasons  map[string]int
	pairs    map[failure]int
	examples map[failure]string
	produced map[string]int
	broke    map[string]int
}

// separator is what GoMutatedFile.Label puts between the path and the infection.
const separator = " → "

// virusOf reads the infection's name off the label, which is the only place a
// mutant carries it. Adding an accessor to the product so a probe can measure
// would let the probe decide the product's API.
//
// It refuses rather than trims. A TrimPrefix that does not match returns the
// whole label, so every count would be attributed to a virus named
// "internal/x/y.go → comparison" and the table would look populated and be
// wrong. An attribution that cannot fail is not an attribution.
func virusOf(t *testing.T, mutant *gomutatedfile.GoMutatedFile) string {
	t.Helper()

	name, found := strings.CutPrefix(mutant.Label(), mutant.Path()+separator)
	require.True(t, found,
		"label %q does not begin with path %q, so the virus cannot be read off it",
		mutant.Label(), mutant.Path())

	return name
}

// report prints the totals, the causes ordered by how often they fired, and the
// files that carry them. One cause repeated a hundred times and a hundred causes
// are different problems, so the messages are counted rather than listed.
func report(t *testing.T, mutants, broken int, counted *tallies, perFile map[string]string) {
	t.Helper()

	t.Logf("mutants %d, of which do not build %d (%.1f%%)",
		mutants, broken, 100*float64(broken)/float64(mutants))

	messages := make([]string, 0, len(counted.reasons))
	for message := range counted.reasons {
		messages = append(messages, message)
	}

	sort.Slice(messages, func(i, j int) bool {
		return counted.reasons[messages[i]] > counted.reasons[messages[j]]
	})

	for _, message := range messages {
		t.Logf("  %4d  %s", counted.reasons[message], message)
	}

	reportViruses(t, counted)

	for file, count := range perFile {
		if !strings.HasPrefix(count, "0 ") {
			t.Logf("  %s: %s", count, file)
		}
	}
}

// reportViruses answers the question the message column cannot: which virus
// wrote the mutant that would not build, and how much of that virus's output
// refusing it would cost. A virus whose failures are a tenth of what it produces
// cannot simply be switched off.
func reportViruses(t *testing.T, counted *tallies) {
	t.Helper()

	viruses := make([]string, 0, len(counted.broke))
	for virus := range counted.broke {
		viruses = append(viruses, virus)
	}

	sort.Slice(viruses, func(i, j int) bool {
		return counted.broke[viruses[i]] > counted.broke[viruses[j]]
	})

	t.Logf("viruses that produced a mutant that does not build: %d of 14", len(viruses))

	for _, virus := range viruses {
		t.Logf("  %4d of %4d produced  %s", counted.broke[virus], counted.produced[virus], virus)
	}

	pairs := make([]failure, 0, len(counted.pairs))
	for pair := range counted.pairs {
		pairs = append(pairs, pair)
	}

	sort.Slice(pairs, func(i, j int) bool { return counted.pairs[pairs[i]] > counted.pairs[pairs[j]] })

	for _, pair := range pairs {
		t.Logf("  %4d  %s  →  %s", counted.pairs[pair], pair.virus, pair.cause)
		t.Logf("        e.g. %s", counted.examples[pair])
	}
}

func countOne(t *testing.T, root, path string, counted *tallies) (int, int) {
	t.Helper()

	original, err := os.ReadFile(path)
	if err != nil {
		return 0, 0
	}

	infected := gosourcefile.New(path, original).Incubate(everyVirus()...)
	broken := 0

	for _, one := range infected {
		mutant := one.Mutate()
		virus := virusOf(t, mutant)
		counted.produced[virus]++

		require.NoError(t, os.WriteFile(path, mutant.Mutated(), 0o600)) //nolint:gosec

		if ok, message := buildsWithMessage(root, filepath.Dir(path)); !ok {
			broken++
			counted.reasons[message]++
			counted.broke[virus]++

			pair := failure{virus: virus, cause: message}
			counted.pairs[pair]++

			if _, seen := counted.examples[pair]; !seen {
				counted.examples[pair] = mutant.Label()
			}
		}
	}

	require.NoError(t, os.WriteFile(path, original, 0o600)) //nolint:gosec

	return len(infected), broken
}

// compilerMessage keeps the kind of complaint and drops the file, line and
// identifier, so a hundred instances of one cause count as one cause.
var compilerMessage = regexp.MustCompile(`^.*?\.go:\d+:\d+: `)

func buildsWithMessage(root, packageDir string) (bool, string) {
	relative, err := filepath.Rel(root, packageDir)
	if err != nil {
		return false, err.Error()
	}

	//nolint:gosec,noctx // the argument is a path inside the tree the caller named
	command := exec.Command("go", "build", "./"+filepath.ToSlash(relative))
	command.Dir = root

	output, err := command.CombinedOutput()
	if err == nil {
		return true, ""
	}

	for line := range strings.SplitSeq(string(output), "\n") {
		if compilerMessage.MatchString(line) {
			return false, strings.TrimSpace(compilerMessage.ReplaceAllString(line, ""))
		}
	}

	return false, "unrecognised build failure"
}

func everyVirus() []viruses.Virus {
	return []viruses.Virus{
		arithmetic.New(), arithmeticassignment.New(), arithmeticassignmentinvert.New(),
		bitwise.New(), comparison.New(), comparisoninvert.New(), comparisonreplace.New(),
		floatdecrement.New(), floatincrement.New(), integerdecrement.New(),
		integerincrement.New(), loopbreak.New(), loopcondition.New(), rangebreak.New(),
	}
}

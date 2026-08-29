package gomutatedfile

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/Disble/ditto/internal/schemata"
)

type Repository interface {
	Overwrite(filePath string, data []byte)
}

type Differ interface {
	Diff(a, b string, aData, bData []byte) string
}

type GoMutatedFile struct {
	infectionName     string
	relativePath      string
	rawSourceContent  []byte
	rawMutatedContent []byte

	address string
	change  string
}

func New(infectionName, relativePath string, rawSourceContent, rawMutatedContent []byte) *GoMutatedFile {
	address, change := locate(relativePath, rawSourceContent, rawMutatedContent)

	return &GoMutatedFile{
		infectionName:     infectionName,
		relativePath:      relativePath,
		rawSourceContent:  rawSourceContent,
		rawMutatedContent: rawMutatedContent,
		address:           address,
		change:            change,
	}
}

// locate reads where a mutant struck, and what it wrote there, off the one
// difference between the file before it and the file after it.
//
// It asks schemata.Difference rather than asking the fourteen viruses to report
// their own site. The answer already exists — it is the same difference the
// gated path builds its gates from — and teaching a second part of the tree what
// each virus replaces would be a copy that goes stale without anything failing.
// It also costs no parse: the bytes are already in hand.
//
// Measured over 135 mutants of four gofmt'd files, in
// docs/experiments/mutant-address.md: the address falls inside the mutated node
// for every one of them, and the tuple printed beside it is unique for every one
// of them — against 129 of those 135 that could not be told apart from another
// mutant by what ditto printed before this.
//
// A mutant with no difference to read — identical bytes, or none at all — keeps
// the path as its address. A report without a line number is worse than one
// with; one that panicked would be worse than both.
func locate(relativePath string, source, mutated []byte) (string, string) {
	difference, found := schemata.Difference(source, mutated)
	if !found {
		return relativePath, ""
	}

	line, column := lineColumn(source, difference.Start)

	return relativePath + ":" + strconv.Itoa(line) + ":" + strconv.Itoa(column),
		describe(difference.Original, difference.Mutated)
}

// describe says what the mutation did, in the fewest words that stay true.
//
// A difference is the smallest stretch of bytes that changed, so one side of it
// is empty whenever the mutation added or removed rather than swapped —
// widening `>` to `>=` inserts one byte and replaces nothing. Rendering that as
// an arrow with an empty left side reads as a rendering fault, so the operation
// is named instead. Seen in the golden's own output before it was fixed:
// `Comparison ((nothing) → =)`.
func describe(original, mutated string) string {
	before, after := summarize(original), summarize(mutated)

	switch {
	case before == "" && after == "":
		return ""
	case before == "":
		return "inserts " + after
	case after == "":
		return "deletes " + before
	default:
		return before + " → " + after
	}
}

// lineColumn converts a byte offset into the one-based line and column an editor
// jumps to.
func lineColumn(source []byte, offset int) (int, int) {
	offset = min(max(offset, 0), len(source))
	before := source[:offset]

	return 1 + bytes.Count(before, []byte("\n")), offset - (bytes.LastIndexByte(before, '\n') + 1) + 1
}

// summaryWidth bounds one side of the change. A difference is the smallest
// stretch of bytes that differs, and for an insertion that stretch can drag part
// of the following statement along with it; on one summary line that pushes the
// address off the screen, which is the thing the line exists to show.
const summaryWidth = 48

// summarize renders one side of a replacement onto a single line. The rendered
// diff below the summary is where the original layout still lives; this line is
// an index into it.
func summarize(text string) string {
	collapsed := strings.Join(strings.Fields(text), " ")

	// Runes rather than bytes: a Go identifier may be non-ASCII, and cutting one
	// in half would print a replacement character where the mutation should be.
	if runes := []rune(collapsed); len(runes) > summaryWidth {
		return string(runes[:summaryWidth]) + "…"
	}

	return collapsed
}

func (f *GoMutatedFile) WriteTo(repository Repository) {
	repository.Overwrite(f.relativePath, f.rawMutatedContent)
}

// RestoreIn puts the original bytes back where WriteTo put the mutated ones.
//
// It is what lets one temporary repository serve every mutant instead of being
// rebuilt for each: a sandbox that is handed back carrying the previous
// mutation would silently test two mutations at once.
func (f *GoMutatedFile) RestoreIn(repository Repository) {
	repository.Overwrite(f.relativePath, f.rawSourceContent)
}

// Source is the file before this mutant touched it, and Path is where it lives.
//
// The gated path needs both: it plans one instrumented file from the original
// and every mutant of it, and writes that file back where the original was.
func (f *GoMutatedFile) Source() []byte { return f.rawSourceContent }

func (f *GoMutatedFile) Path() string { return f.relativePath }

// Mutated is the file as this mutant leaves it.
//
// The gated path needs the bytes rather than the effect of writing them: it
// reads what a virus replaced off the difference between these and the original,
// instead of asking each of the fourteen viruses what it does.
func (f *GoMutatedFile) Mutated() []byte {
	return f.rawMutatedContent
}

func (f *GoMutatedFile) String() string {
	return f.relativePath
}

// Address is where a reader has to look: `path:line:col`, or the path alone
// when no difference could be read.
func (f *GoMutatedFile) Address() string { return f.address }

// Change is the replacement itself, `original → mutated`, on one line. It is
// empty when there is no difference to describe.
func (f *GoMutatedFile) Change() string { return f.change }

// Label names one mutant. It is the subtest's name and the survivor's headline,
// so it carries the address: `go test` separates repeated subtest names with
// #01 and #02, which told a reader that two mutants differed and nothing about
// where either of them was.
func (f *GoMutatedFile) Label() string {
	return f.address + " → " + f.infectionName
}

func (f *GoMutatedFile) Diff(differ Differ) string {
	return differ.Diff(
		f.relativePath+" (original)",
		f.relativePath+" (mutated with '"+f.infectionName+"')",
		f.rawSourceContent,
		f.rawMutatedContent,
	)
}

// Virus is the mutation operator that produced this file.
//
// It is what somebody fixes when a mutant does not compile: the rate has an
// external benchmark -- Major 1.8%, PIT 0% -- and a rate alone names no work.
// docs/metrics.md metric 1.
func (f *GoMutatedFile) Virus() string { return f.infectionName }

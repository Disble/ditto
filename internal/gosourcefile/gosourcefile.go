package gosourcefile

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/Disble/ditto/internal/goinfectedfile"
	"github.com/Disble/ditto/viruses"
)

// Range is a half-open byte range within one file.
type Range struct {
	Start int
	End   int
}

type GoSourceFile struct {
	relativePath string
	rawContent   []byte
	// ranges limits mutation to part of this file. Empty means the whole file.
	ranges []Range
}

func New(relativePath string, rawContent []byte) *GoSourceFile {
	return &GoSourceFile{
		relativePath: relativePath,
		rawContent:   rawContent,
	}
}

// Restrict returns the same file mutated only within the given byte ranges.
//
// The ranges belong to the file rather than to the run, and that is the whole
// point. Offsets are only meaningful against the file they were measured in —
// every file is parsed with its own token.FileSet, so every file's positions
// start from the same base — and a scope held anywhere other than beside its
// file cannot tell them apart. Merging several files' ranges into one set makes
// each file answer to all of them, which mutates code no change touched and
// grows the work as the square of the number of files.
func (f *GoSourceFile) Restrict(ranges []Range) *GoSourceFile {
	return &GoSourceFile{
		relativePath: f.relativePath,
		rawContent:   f.rawContent,
		ranges:       ranges,
	}
}

// inScope reports whether a node position falls inside this file's ranges.
//
// token.Pos is one past the byte offset when the file was parsed on its own,
// which is how GoSourceFile parses. A node with no position carries no scope
// information, so it is let through rather than silently dropped.
func (f *GoSourceFile) inScope(node ast.Node) bool {
	if len(f.ranges) == 0 || node == nil || node.Pos() == token.NoPos {
		return true
	}

	offset := int(node.Pos()) - 1
	for _, span := range f.ranges {
		if offset >= span.Start && offset < span.End {
			return true
		}
	}

	return false
}

// Incubate parses the file once and offers it to every mutator given.
//
// It takes the whole set rather than one mutator because the parse is the
// expensive part and it does not depend on which mutator is asking. Called once
// per mutator, as it was, a source file was parsed fourteen times per release
// with the default set — thirteen of them producing an identical tree.
//
// Each mutator still walks the tree in its own pass, so the mutants come out in
// exactly the order they did before. Only the parsing is shared.
func (f *GoSourceFile) Incubate(viri ...viruses.Virus) []*goinfectedfile.GoInfectedFile {
	fileSet := token.NewFileSet()

	fileTree, err := parser.ParseFile(fileSet, f.relativePath, f.rawContent, parser.ParseComments|parser.AllErrors)
	if err != nil {
		panic(fmt.Errorf("failed parsing file '%s': %w", f.relativePath, err))
	}

	var infectedFiles []*goinfectedfile.GoInfectedFile

	for _, virus := range viri {
		ast.Inspect(fileTree, func(node ast.Node) bool {
			if !f.inScope(node) {
				// Keep descending: a node outside the ranges can still hold
				// children inside them.
				return true
			}

			for _, infection := range virus.Incubate(node, nil) {
				infectedFiles = append(infectedFiles, goinfectedfile.New(f.relativePath, f.rawContent, infection, fileSet, fileTree))
			}

			return true
		})
	}

	return infectedFiles
}

func (f *GoSourceFile) String() string {
	return f.relativePath
}

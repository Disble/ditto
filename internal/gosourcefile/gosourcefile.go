package gosourcefile

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/Disble/ditto/internal/goinfectedfile"
	"github.com/Disble/ditto/viruses"
)

type GoSourceFile struct {
	relativePath string
	rawContent   []byte
}

func New(relativePath string, rawContent []byte) *GoSourceFile {
	return &GoSourceFile{
		relativePath: relativePath,
		rawContent:   rawContent,
	}
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

package gomutatedfile

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
}

func New(infectionName, relativePath string, rawSourceContent, rawMutatedContent []byte) *GoMutatedFile {
	return &GoMutatedFile{
		infectionName:     infectionName,
		relativePath:      relativePath,
		rawSourceContent:  rawSourceContent,
		rawMutatedContent: rawMutatedContent,
	}
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

func (f *GoMutatedFile) String() string {
	return f.relativePath
}

func (f *GoMutatedFile) Label() string {
	return f.relativePath + " → " + f.infectionName
}

func (f *GoMutatedFile) Diff(differ Differ) string {
	return differ.Diff(
		f.relativePath+" (original)",
		f.relativePath+" (mutated with '"+f.infectionName+"')",
		f.rawSourceContent,
		f.rawMutatedContent,
	)
}

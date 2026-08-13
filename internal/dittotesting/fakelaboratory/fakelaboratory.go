package fakelaboratory

import (
	"reflect"

	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/future"
	"github.com/Disble/ditto/internal/gomutatedfile"
	"github.com/Disble/ditto/internal/result"
)

type FakeLaboratory struct {
	always  result.Result[string]
	results []*Result
}

type Result struct {
	expectedRepository  ditto.Repository
	expectedMutatedFile *gomutatedfile.GoMutatedFile
	output              result.Result[string]
}

func NewResult(
	expectedRepository ditto.Repository,
	expectedMutatedFile *gomutatedfile.GoMutatedFile,
	output result.Result[string],
) *Result {
	return &Result{
		expectedRepository:  expectedRepository,
		expectedMutatedFile: expectedMutatedFile,
		output:              output,
	}
}

func New(tuples ...*Result) *FakeLaboratory {
	return &FakeLaboratory{
		always:  nil,
		results: tuples,
	}
}

func NewAlways(output result.Result[string]) *FakeLaboratory {
	return &FakeLaboratory{
		always:  output,
		results: []*Result{},
	}
}

func (l *FakeLaboratory) Test(
	repository ditto.Repository,
	file *gomutatedfile.GoMutatedFile,
) future.Future[result.Result[string]] {
	if l.always != nil {
		return future.Resolved(l.always)
	}

	for _, res := range l.results {
		sameRepository := repository == res.expectedRepository
		sameMutatedFile := reflect.DeepEqual(file, res.expectedMutatedFile)

		if sameRepository && sameMutatedFile {
			return future.Resolved(res.output)
		}
	}

	panic("unexpected mutated file")
}

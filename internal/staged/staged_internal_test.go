package staged

import (
	"reflect"
	"testing"
)

func TestChangedLinesReadsEveryHunk(t *testing.T) {
	t.Parallel()

	diff := "--- a/x.go\n+++ b/x.go\n@@ -1,2 +1,3 @@\n context\n@@ -20 +30,4 @@\n more\n"

	spans, err := changedLines(diff)
	if err != nil {
		t.Fatalf("changedLines: %v", err)
	}

	want := []lineSpan{{first: 1, last: 3}, {first: 30, last: 33}}
	if !reflect.DeepEqual(spans, want) {
		t.Fatalf("spans = %+v, want %+v", spans, want)
	}
}

// A hunk with a count of zero is a pure deletion: nothing was added at that
// line, so the bytes worth mutating are the ones the deletion joined.
func TestChangedLinesCoversAPureDeletion(t *testing.T) {
	t.Parallel()

	spans, err := changedLines("@@ -4,3 +3,0 @@\n")
	if err != nil {
		t.Fatalf("changedLines: %v", err)
	}

	want := []lineSpan{{first: 3, last: 4}}
	if !reflect.DeepEqual(spans, want) {
		t.Fatalf("spans = %+v, want %+v", spans, want)
	}
}

func TestChangedLinesRejectsAHunkItCannotRead(t *testing.T) {
	t.Parallel()

	spans, err := changedLines("@@ -1 +x @@\n")
	if err != nil {
		t.Fatalf("changedLines: %v", err)
	}

	if len(spans) != 0 {
		t.Fatalf("spans = %+v, want none: an unreadable header names no line", spans)
	}
}

func TestOffsetsOfUseTheContentsOwnLineStarts(t *testing.T) {
	t.Parallel()

	content := []byte("package a\nfunc B() {}\nfunc C() {}\n")

	offsets := offsetsOf(content, []lineSpan{{first: 2, last: 2}})

	want := []Range{{Start: 10, End: 22}}
	if !reflect.DeepEqual(offsets, want) {
		t.Fatalf("offsets = %+v, want %+v", offsets, want)
	}
}

// A span past the end of the file names bytes that are not there. Dropping it is
// what makes the scope a fact about the content rather than about the diff.
func TestOffsetsOfDropASpanPastTheEnd(t *testing.T) {
	t.Parallel()

	offsets := offsetsOf([]byte("package a\n"), []lineSpan{{first: 9, last: 9}})
	if len(offsets) != 0 {
		t.Fatalf("offsets = %+v, want none", offsets)
	}
}

func TestMergeJoinsOverlappingRanges(t *testing.T) {
	t.Parallel()

	merged := merge([]Range{{Start: 40, End: 50}, {Start: 0, End: 10}, {Start: 8, End: 20}})

	want := []Range{{Start: 0, End: 20}, {Start: 40, End: 50}}
	if !reflect.DeepEqual(merged, want) {
		t.Fatalf("merged = %+v, want %+v", merged, want)
	}
}

func TestSelectMutableKeepsProductionSourcesOnly(t *testing.T) {
	t.Parallel()

	files := []string{
		"internal\\calc\\calc.go",
		"internal/calc/calc_test.go",
		"tools/thing/main.go",
		"README.md",
		"",
	}

	selected := selectMutable(files, []string{"tools/"})

	want := []string{"internal/calc/calc.go"}
	if !reflect.DeepEqual(selected, want) {
		t.Fatalf("selected = %v, want %v", selected, want)
	}
}

// An empty prefix would exclude every path, because every path starts with the
// empty string. Ignoring it is what lets a caller pass a list it built.
func TestSelectMutableIgnoresAnEmptyPrefix(t *testing.T) {
	t.Parallel()

	selected := selectMutable([]string{"a.go"}, []string{""})
	if len(selected) != 1 {
		t.Fatalf("selected = %v, want a.go kept", selected)
	}
}

func TestWithoutGitEnvironmentRemovesTheInheritedAddressing(t *testing.T) {
	t.Parallel()

	kept := withoutGitEnvironment([]string{"PATH=/bin", "GIT_DIR=/somewhere/.git", "HOME=/home/x", "GIT_INDEX_FILE=/i"})

	want := []string{"PATH=/bin", "HOME=/home/x"}
	if !reflect.DeepEqual(kept, want) {
		t.Fatalf("kept = %v, want %v", kept, want)
	}
}

// failOpen maps every file to no ranges, which is what WithChangedRanges reads
// as "mutate this one whole". A file left out of the map is not mutated at all,
// so the difference is the whole meaning of failing open.
func TestFailOpenNamesEveryFileWithNoRanges(t *testing.T) {
	t.Parallel()

	scope := failOpen([]string{"a.go", "b.go"}, "because")

	if scope.Derived || scope.Reason != "because" {
		t.Fatalf("scope = %+v, want a fallback that says why", scope)
	}

	for _, file := range []string{"a.go", "b.go"} {
		ranges, named := scope.Ranges[file]
		if !named || len(ranges) != 0 {
			t.Fatalf("%s maps to %+v, want a named file with no ranges", file, ranges)
		}
	}
}

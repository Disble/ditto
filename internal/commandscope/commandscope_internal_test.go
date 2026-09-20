package commandscope

import (
	"reflect"
	"testing"
)

// A stream names every package the command executes, and a package with no test
// files is one of them: go test reports it as a skip rather than staying silent.
// Measured on go1.27 — this is the case that decides whether a package nobody
// wrote tests for is read as executed or as absent.
func TestExecutedPackagesReadsAPackageWithNoTestFiles(t *testing.T) {
	t.Parallel()

	stream := `{"Action":"start","Package":"probe/internal/noTests"}
{"Action":"output","Package":"probe/internal/noTests","Output":"?   \tprobe/internal/noTests\t[no test files]\n"}
{"Action":"skip","Package":"probe/internal/noTests","Elapsed":0}
{"Action":"start","Package":"probe/internal/withTests"}
{"Action":"run","Package":"probe/internal/withTests","Test":"TestSum"}
{"Action":"pass","Package":"probe/internal/withTests","Elapsed":0.3}
`

	got := executedPackages(stream)
	want := []string{"probe/internal/noTests", "probe/internal/withTests"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("executedPackages = %v, want %v", got, want)
	}
}

// Compiler output, prose and mangled JSON share the stream, and none of them is
// an error: the scope is an answer a reader acts on, and refusing it over one
// unreadable line would trade it for nothing.
func TestExecutedPackagesIgnoresWhatIsNotAnEvent(t *testing.T) {
	t.Parallel()

	stream := "?   \tprobe/internal/x\t[no test files]\n" +
		"# probe/internal/x\n" +
		"x.go:3:2: undefined: Missing\n" +
		"{\"Action\":\"start\"}\n" +
		"{\"Package\":\"probe/internal/x\"}\n" +
		"{not json}\n" +
		"{\"Action\":\"start\",\"Package\":\"probe/internal/x\"}\n"

	got := executedPackages(stream)
	want := []string{"probe/internal/x"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("executedPackages = %v, want %v", got, want)
	}
}

// A command that is not `go test -json` prints none of this, and the answer has
// to be "nothing was named" rather than a guess: make, gotestsum and a wrapper
// script all read as unknown here.
func TestExecutedPackagesIsEmptyForACommandThatNamedNothing(t *testing.T) {
	t.Parallel()

	for _, stream := range []string{"", "ok  \tprobe/internal/x\t0.2s\n", "BUILD FAILED\n"} {
		if got := executedPackages(stream); len(got) != 0 {
			t.Fatalf("executedPackages(%q) = %v, want nothing", stream, got)
		}
	}
}

func TestRelativeDirectoryRefusesWhatIsOutsideTheRoot(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		root      string
		directory string
		want      string
	}{
		{name: "inside", root: "/tree", directory: "/tree/internal/desktop", want: "internal/desktop"},
		{name: "the root itself", root: "/tree", directory: "/tree", want: "."},
		{name: "a sibling with the same prefix", root: "/tree", directory: "/treehouse/x", want: ""},
		{name: "elsewhere", root: "/tree", directory: "/other/x", want: ""},
		{name: "the standard library", root: "/tree", directory: "/usr/local/go/src/fmt", want: ""},
		{name: "no directory at all", root: "/tree", directory: "", want: ""},
	}

	for _, testcase := range cases {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			if got := relativeDirectory(testcase.root, testcase.directory, "linux"); got != testcase.want {
				t.Fatalf("relativeDirectory(%q, %q) = %q, want %q",
					testcase.root, testcase.directory, got, testcase.want)
			}
		})
	}
}

// The case-fold is the half CI cannot run, so it is exercised here with the
// target named. Windows spells one directory two ways and they are one package;
// reporting them as two fails to clear a mutant, which is a wrong accusation
// rather than a missing one.
func TestNormalizePathFoldsOnlyOnWindows(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		goos string
		path string
		want string
	}{
		{name: "windows folds case", goos: "windows", path: `Internal\Desktop`, want: "internal/desktop"},
		{name: "windows folds the drive letter", goos: "windows", path: `C:\Tree\Pkg`, want: "c:/tree/pkg"},
		{name: "windows drops a trailing separator", goos: "windows", path: `C:\Tree\Pkg\`, want: "c:/tree/pkg"},
		{name: "linux folds nothing", goos: "linux", path: "Internal/Desktop", want: "Internal/Desktop"},
	}

	for _, testcase := range cases {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			if got := normalizePath(testcase.path, testcase.goos); got != testcase.want {
				t.Fatalf("normalizePath(%q, %q) = %q, want %q", testcase.path, testcase.goos, got, testcase.want)
			}
		})
	}
}

// And the two spellings of one Windows directory have to reach the same key, or
// the fold is decorative: that is the failure this test exists for.
func TestTheTwoSpellingsOfOneDirectoryAgree(t *testing.T) {
	t.Parallel()

	root := `C:\Users\Dev\Temp\ditto-1`
	fromToolchain := normalizePath(relativeDirectory(root, root+`\internal\desktop`, "windows"), "windows")
	fromDitto := normalizePath("internal/desktop", "windows")

	if fromToolchain != fromDitto {
		t.Fatalf("one directory produced two keys: %q and %q", fromToolchain, fromDitto)
	}
}

// The inherited git addressing is removed from every process ditto starts, and
// a list is not a guard: this is the half that would notice the list being
// dropped from the command above.
func TestWithoutGitEnvironmentKeepsEverythingElse(t *testing.T) {
	t.Parallel()

	kept := withoutGitEnvironment([]string{
		"PATH=/bin", "GIT_DIR=/somewhere/.git", "HOME=/home/x",
		"GIT_INDEX_FILE=/i", "GIT_WORK_TREE=/w", "GIT_OBJECT_DIRECTORY=/o", "GIT_COMMON_DIR=/c",
	})
	want := []string{"PATH=/bin", "HOME=/home/x"}

	if !reflect.DeepEqual(kept, want) {
		t.Fatalf("withoutGitEnvironment = %v, want %v", kept, want)
	}
}

package gobuildrunner

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/Disble/ditto/internal/cmdtestrunner"
	"github.com/Disble/ditto/internal/ditto"
	"github.com/Disble/ditto/internal/result"
)

// ModuleScopeRunner compiles the default module test scope once and executes
// every resulting package test binary for each selected mutant.
type ModuleScopeRunner struct {
	mutant int

	toolchain string
	packages  []modulePackage
	output    string
	failure   string

	discovered   bool
	buildAttempt bool
	built        bool

	discoveries     int
	toolchainStarts int
	compilations    int
	selections      int
	packageRuns     int
	converterStarts int
	skippedPackages int
	scopedDir       string
}

type modulePackage struct {
	importPath  string
	directory   string
	relativeDir string
	binary      string
	hasTests    bool

	// observes is the relative directories whose code is compiled into this
	// package's test binary. A mutation outside it cannot be referred to by
	// anything in the binary, so running it can only cost time.
	observes map[string]bool
}

// The field names are Go's own, capitalised, which is what `go list -json`
// emits; tagliatelle wants camelCase and is overruled because this struct
// decodes somebody else's format rather than defining one. Same shape and same
// reason as the event struct in internal/verdict.
type goListPackage struct {
	ImportPath   string   `json:"ImportPath"`   //nolint:tagliatelle // go list -json emits these names
	Dir          string   `json:"Dir"`          //nolint:tagliatelle // go list -json emits these names
	TestGoFiles  []string `json:"TestGoFiles"`  //nolint:tagliatelle // go list -json emits these names
	XTestGoFiles []string `json:"XTestGoFiles"` //nolint:tagliatelle // go list -json emits these names
	Deps         []string `json:"Deps"`         //nolint:tagliatelle // go list -json emits these names
}

// errBinaryNameCollision is what a module scope reports when two packages would
// write the same test binary. `go test -c -o <directory> ./...` refuses such a
// tree outright, so a scope that reached the build would fail there with a
// message about the output directory rather than about the packages.
var errBinaryNameCollision = errors.New("ditto: module test binary name collision")

// NewModuleScope returns a runner for the exact default ./... Go test scope.
func NewModuleScope() *ModuleScopeRunner {
	return &ModuleScopeRunner{toolchain: goToolchain()}
}

// Toolchain is the resolved Go executable, or empty when none could be found.
func (r *ModuleScopeRunner) Toolchain() string { return r.toolchain }

// Select chooses the mutant test binaries receive through Selected.
func (r *ModuleScopeRunner) Select(mutant int) { r.mutant = mutant }

// Discoveries counts structured package-layout discovery attempts.
func (r *ModuleScopeRunner) Discoveries() int { return r.discoveries }

// ToolchainStarts counts go list and go test process starts.
func (r *ModuleScopeRunner) ToolchainStarts() int { return r.toolchainStarts }

// Compilations counts complete-scope go test -c invocations.
func (r *ModuleScopeRunner) Compilations() int { return r.compilations }

// Selections counts baseline and mutant selections answered by Test.
func (r *ModuleScopeRunner) Selections() int { return r.selections }

// PackageRuns counts started package test binaries.
func (r *ModuleScopeRunner) PackageRuns() int { return r.packageRuns }

// ConverterStarts counts conversions of a failing package's output into the
// stream internal/verdict reads a reason from. A green selection starts none,
// because a reason is only ever asked of a kill.
func (r *ModuleScopeRunner) ConverterStarts() int { return r.converterStarts }

// SkippedPackages counts package test binaries not started because the mutated
// package is not in what they compile. It is the counter the scoping change is
// judged on, and it says nothing unless ScopeTo was called.
func (r *ModuleScopeRunner) SkippedPackages() int { return r.skippedPackages }

// SetCompilationDirectory writes this runner's test binaries into a directory
// chosen by the caller, so the batches of one release reuse one instead of each
// building its own.
//
// It only pays together with a sandbox that is reused too. A fresh directory per
// batch throws away the toolchain's up-to-date check; a fresh sandbox per batch
// defeats it anyway, because Go's build IDs cover the package directories.
// Measured both ways: the shared directory alone bought nothing at all.
func (r *ModuleScopeRunner) SetCompilationDirectory(directory string) {
	r.output = directory
}

// ScopeTo declares the repository-relative directory of the package whose
// mutation this batch selects, so only the test binaries that can observe it
// are started.
//
// It is optional on purpose. A runner with no scope runs every package, which is
// what this did before the closure was measured, so a caller that never declares
// one loses nothing but time.
func (r *ModuleScopeRunner) ScopeTo(directory string) {
	r.scopedDir = path.Clean(filepath.ToSlash(directory))
}

// Built is true only after discovery, layout validation, compilation, and every
// expected test-binary check has succeeded.
func (r *ModuleScopeRunner) Built() bool { return r.built }

// Test preserves the package runner's result contract: Ok means a command or
// test failed, while Err means every package test passed.
func (r *ModuleScopeRunner) Test(repository ditto.TemporaryRepository) result.Result[string] {
	r.selections++

	if !r.built {
		if !r.buildAttempt {
			r.buildAttempt = true
			r.failure = r.prepare(repository.Root())
		}

		if !r.built {
			return result.Ok(r.failure)
		}
	}

	return r.run()
}

func (r *ModuleScopeRunner) prepare(root string) string {
	// A directory given by the caller outlives this runner, so its up-to-date
	// state survives into the next batch. One made here does not.
	if r.output == "" {
		output, err := os.MkdirTemp(root, "ditto-module-tests-")
		if err != nil {
			return fmt.Sprintf("ditto: create module test output directory: %v", err)
		}

		r.output = output
	}

	output := r.output

	if err := r.discover(root); err != nil {
		return err.Error()
	}

	r.compilations++
	r.toolchainStarts++
	command := exec.Command(r.toolchain, "test", "-c", "-o", output, "./...") //nolint:noctx,gosec // resolved to an absolute path in goToolchain
	command.Dir = root
	command.Env = environment(r.mutant)

	buildOutput, err := command.CombinedOutput()
	if err != nil {
		return string(buildOutput)
	}

	for _, pkg := range r.packages {
		if !pkg.hasTests {
			continue
		}

		info, err := os.Stat(pkg.binary)
		if err != nil || info.IsDir() {
			return fmt.Sprintf("ditto: expected test binary for %s at %s", pkg.importPath, pkg.binary)
		}
	}

	r.built = true

	return ""
}

func (r *ModuleScopeRunner) discover(root string) error {
	r.discovered = true

	r.discoveries++
	if r.toolchain == "" {
		return errNoToolchain
	}

	r.toolchainStarts++

	// -deps and -test are what make the observability closure available: a test
	// main package's Deps are exactly what its binary compiles in, test imports
	// included. A closure built from Imports alone would be too narrow, because a
	// package's external test file is a separate package that imports the
	// subject.
	command := exec.Command(r.toolchain, "list", "-deps", "-test", "-json", "./...") //nolint:noctx,gosec // resolved to an absolute path in goToolchain
	command.Dir = root
	command.Env = environment(r.mutant)

	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ditto: discover module packages: %w\n%s", err, strings.TrimSpace(string(output)))
	}

	decoder := json.NewDecoder(strings.NewReader(string(output)))

	byImportPath, depsByPackage, err := decodeLayout(root, decoder)
	if err != nil {
		return err
	}

	packages := make([]modulePackage, 0, len(byImportPath))

	seenBinaries := make(map[string]string)

	for importPath, pkg := range byImportPath {
		// A package's own test binary compiles the package, so it observes itself
		// whatever the toolchain reports about its dependencies.
		pkg.observes = map[string]bool{pkg.relativeDir: true}

		for _, dep := range depsByPackage[importPath] {
			observed, known := byImportPath[depWithoutVariant(dep)]
			if known && observed.relativeDir != "" {
				pkg.observes[observed.relativeDir] = true
			}
		}

		if pkg.hasTests {
			name := moduleTestBinaryName(importPath, runtime.GOOS)
			if other, exists := seenBinaries[name]; exists {
				return fmt.Errorf("%w: %s and %s both produce %s", errBinaryNameCollision, other, importPath, name)
			}

			seenBinaries[name] = importPath
			pkg.binary = filepath.Join(r.output, name)
		}

		packages = append(packages, pkg)
	}

	sort.Slice(packages, func(i, j int) bool {
		return packages[i].importPath < packages[j].importPath
	})
	r.packages = packages

	return nil
}

// decodeLayout reads one `go list` stream into the module's own packages and
// what each test main compiles in.
//
// Only packages inside the module are kept. With -deps the stream also carries
// the standard library and every other dependency, and none of those is part of
// the configured `./...` scope or a candidate for a test binary of it.
func decodeLayout(root string, decoder *json.Decoder) (map[string]modulePackage, map[string][]string, error) {
	byImportPath := make(map[string]modulePackage)
	depsByPackage := make(map[string][]string)

	for decoder.More() {
		var listed goListPackage
		if err := decoder.Decode(&listed); err != nil {
			return nil, nil, fmt.Errorf("ditto: decode module package layout: %w", err)
		}

		// A test variant is reported as `pkg [pkg.test]`. It is the same
		// directory seen a second time, so it is skipped rather than counted.
		if strings.Contains(listed.ImportPath, " [") {
			continue
		}

		if before, ok := strings.CutSuffix(listed.ImportPath, ".test"); ok {
			depsByPackage[before] = listed.Deps

			continue
		}

		relative, relErr := filepath.Rel(root, listed.Dir)
		if listed.Dir == "" || relErr != nil || strings.HasPrefix(filepath.ToSlash(relative), "..") {
			continue
		}

		byImportPath[listed.ImportPath] = modulePackage{
			importPath:  listed.ImportPath,
			directory:   listed.Dir,
			relativeDir: path.Clean(filepath.ToSlash(relative)),
			hasTests:    len(listed.TestGoFiles)+len(listed.XTestGoFiles) > 0,
		}
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, nil, fmt.Errorf("ditto: decode module package layout: %w", err)
	}

	return byImportPath, depsByPackage, nil
}

// depWithoutVariant strips the ` [pkg.test]` suffix go list -test puts on the
// test-built variant of a package, so a name in a closure matches the package it
// names rather than the form it arrived in.
func depWithoutVariant(importPath string) string {
	if before, _, ok := strings.Cut(importPath, " ["); ok {
		return before
	}

	return importPath
}

func (r *ModuleScopeRunner) run() result.Result[string] {
	filtering := r.scopedDir != "" && r.anyObservers()

	var output strings.Builder

	failed := false

	for _, pkg := range r.packages {
		if !pkg.hasTests {
			continue
		}

		if filtering && !pkg.observes[r.scopedDir] {
			r.skippedPackages++

			continue
		}

		r.packageRuns++

		binaryOutput, err := r.runPackage(pkg)
		if err != nil {
			failed = true

			// Converted only when the package failed, because a reason is only
			// ever asked of a kill: a selection where everything passed starts
			// no converter and pays nothing for a reason nobody reads.
			binaryOutput = r.readable(pkg, binaryOutput)
		}

		output.Write(binaryOutput)
	}

	if failed {
		return result.Ok(output.String())
	}

	return result.Err[string](output.String())
}

// anyObservers reports whether any test binary claims to observe the declared
// scope.
//
// It is the fail-open rule. The closure is a saving and never a licence to run
// less than the caller asked for, so a declared scope that nothing resolves — a
// discovery that returned no dependencies, a layout this cannot read — runs
// every package instead of none.
func (r *ModuleScopeRunner) anyObservers() bool {
	for _, pkg := range r.packages {
		if pkg.hasTests && pkg.observes[r.scopedDir] {
			return true
		}
	}

	return false
}

// runPackage starts one package's test binary from that package's own directory,
// which is what `go test` does and what a suite reading a relative path depends
// on.
func (r *ModuleScopeRunner) runPackage(pkg modulePackage) ([]byte, error) {
	// -test.timeout is passed because a test binary invoked directly takes 0 --
	// timeout disabled -- and only the `go test` driver injects the 10 minute
	// default. Without it a mutant that loops never returns and the release
	// never ends; loopcondition, loopbreak and rangebreak are all in the default
	// virus set.
	command := exec.Command(pkg.binary, "-test.count=1", //nolint:gosec,noctx // binary was verified after this runner built it
		"-test.timeout="+cmdtestrunner.DefaultDeadline.String())
	command.Dir = pkg.directory
	command.Env = environment(r.mutant)

	output, err := command.CombinedOutput()
	if err != nil {
		// The caller only needs to know the package failed; the tool's own words
		// are already in output, which is what the report prints. Wrapped so the
		// exit status is not lost if anything ever asks for it.
		return output, fmt.Errorf("ditto: %s failed: %w", pkg.importPath, err)
	}

	return output, nil
}

// readable turns a failing package's own output into the stream ditto reads a
// verdict reason from.
//
// A package test binary cannot emit `go test -json`: that flag belongs to the
// driver that starts the binary, not to the binary. So a kill arrived as plain
// text, internal/verdict saw no stream, and every module-path kill reported
// Unknown. Measured against the ordinary command over the same fixture and the
// same mutant: `unknown` against `assertion`,
// docs/experiments/module-path-verdict-reason.md.
//
// That is not a cosmetic loss. internal/confirminglaboratory re-runs a kill only
// when its reason is Assertion, so on the gated path `--confirm-kills` silently
// never fired and a flaky suite's false kill could not be caught.
//
// `go tool test2json` is the toolchain's own conversion, so nothing is
// re-implemented here. The verdict is still the binary's own exit status and
// never the converter's: a converter run over captured text reported to us has
// no test status of its own to report. Output that cannot be converted is
// returned untouched, which degrades to the previous behaviour rather than to a
// wrong reason.
func (r *ModuleScopeRunner) readable(pkg modulePackage, binaryOutput []byte) []byte {
	if len(binaryOutput) == 0 {
		return binaryOutput
	}

	r.converterStarts++

	command := exec.Command(r.toolchain, "tool", "test2json", "-t", "-p", pkg.importPath) //nolint:noctx,gosec // resolved to an absolute path in goToolchain
	command.Dir = pkg.directory
	command.Env = environment(r.mutant)
	command.Stdin = bytes.NewReader(binaryOutput)

	converted, err := command.CombinedOutput()
	if err != nil {
		return binaryOutput
	}

	return converted
}

// moduleTestBinaryName is the test-binary name go test -c -o <directory>
// assigns to one package. Keep the platform suffix decision explicit: package
// metadata always uses slash-separated import paths, while output paths use
// filepath.Join at the call site.
func moduleTestBinaryName(importPath, goos string) string {
	name := path.Base(importPath) + ".test"
	if goos == "windows" {
		return name + ".exe"
	}

	return name
}

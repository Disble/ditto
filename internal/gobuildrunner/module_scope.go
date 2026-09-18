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
}

type modulePackage struct {
	importPath string
	directory  string
	binary     string
	hasTests   bool
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
	output, err := os.MkdirTemp(root, "ditto-module-tests-")
	if err != nil {
		return fmt.Sprintf("ditto: create module test output directory: %v", err)
	}

	r.output = output

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
	command := exec.Command(r.toolchain, "list", "-json", "./...") //nolint:noctx,gosec // resolved to an absolute path in goToolchain
	command.Dir = root
	command.Env = environment(r.mutant)

	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ditto: discover module packages: %w\n%s", err, strings.TrimSpace(string(output)))
	}

	decoder := json.NewDecoder(strings.NewReader(string(output)))

	var packages []modulePackage

	seenBinaries := make(map[string]string)

	for decoder.More() {
		var listed goListPackage
		if err := decoder.Decode(&listed); err != nil {
			return fmt.Errorf("ditto: decode module package layout: %w", err)
		}

		hasTests := len(listed.TestGoFiles)+len(listed.XTestGoFiles) > 0

		pkg := modulePackage{
			importPath: listed.ImportPath,
			directory:  listed.Dir,
			hasTests:   hasTests,
		}
		if hasTests {
			name := moduleTestBinaryName(listed.ImportPath, runtime.GOOS)
			if other, exists := seenBinaries[name]; exists {
				return fmt.Errorf("%w: %s and %s both produce %s", errBinaryNameCollision, other, listed.ImportPath, name)
			}

			seenBinaries[name] = listed.ImportPath
			pkg.binary = filepath.Join(r.output, name)
		}

		packages = append(packages, pkg)
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("ditto: decode module package layout: %w", err)
	}

	sort.Slice(packages, func(i, j int) bool {
		return packages[i].importPath < packages[j].importPath
	})
	r.packages = packages

	return nil
}

func (r *ModuleScopeRunner) run() result.Result[string] {
	var output strings.Builder

	failed := false

	for _, pkg := range r.packages {
		if !pkg.hasTests {
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

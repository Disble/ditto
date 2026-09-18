package ditto

import "testing"

func TestDefaultGoModuleScopeAdmission(t *testing.T) {
	t.Run("defaults to the complete Go module scope", func(t *testing.T) {
		if defaultOptions.commandScope != moduleScope {
			t.Fatalf("default scope = %v, want module scope", defaultOptions.commandScope)
		}
	})

	for _, tt := range []struct {
		name    string
		command string
		want    commandScope
	}{
		{name: "documented default", command: "go test -count=1 ./...", want: moduleScope},
		{name: "built-in JSON default", command: "go test -count=1 -json ./...", want: moduleScope},
		{name: "JSON flags reversed", command: "go test -json -count=1 ./...", want: moduleScope},
		{name: "make target", command: "make test", want: unsupportedScope},
		{name: "package local", command: "go test -count=1 ./internal/gatedlaboratory", want: unsupportedScope},
		{name: "count omitted", command: "go test ./...", want: unsupportedScope},
		{name: "race flag", command: "go test -count=1 -race ./...", want: unsupportedScope},
		{name: "tags flag", command: "go test -count=1 -tags=integration ./...", want: unsupportedScope},
		{name: "absolute executable", command: "/usr/local/go/bin/go test -count=1 ./...", want: unsupportedScope},
		{name: "alternate executable", command: "gotip test -count=1 ./...", want: unsupportedScope},
		{name: "extra spacing", command: "go  test -count=1 ./...", want: unsupportedScope},
		{name: "malformed token", command: "go test -count 1 ./...", want: unsupportedScope},
	} {
		t.Run(tt.name, func(t *testing.T) {
			options := WithTestCommand(tt.command)(defaultOptions)
			if options.commandScope != tt.want {
				t.Fatalf("scope for %q = %v, want %v", tt.command, options.commandScope, tt.want)
			}
		})
	}
}

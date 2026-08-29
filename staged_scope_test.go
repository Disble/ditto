package ditto_test

import (
	"strings"
	"testing"

	"github.com/Disble/ditto"
)

// A staged run that could not derive byte ranges falls back to whole files --
// and widens EVERY file, not only the one whose diff it could not read. It then
// reports survivors at addresses the change never touched, which is a lie about
// scope: measured on perfbench's own fixture, a scope that does not keep each
// range beside its file strays to 8 such addresses.
//
// PlanStaged has carried Derived and Reason since the beginning and RunStaged
// never printed them; only --dry did. docs/metrics.md metric 3.
func TestRunStagedAnnouncesAWidenedScope(t *testing.T) {
	t.Parallel()

	widened := ditto.StagedPlan{Derived: false, Reason: "the diff could not be read"}
	printed := widened.ScopeNotice()

	if !strings.Contains(printed, "never touched") {
		t.Fatalf("a widened scope did not warn about untouched lines: %q", printed)
	}

	if !strings.Contains(printed, "the diff could not be read") {
		t.Fatalf("the announcement does not carry why: %q", printed)
	}
}

// And it says nothing when the scope held, because a line printed on every run
// is a line people stop reading.
func TestRunStagedIsQuietWhenTheScopeHeld(t *testing.T) {
	t.Parallel()

	plan := ditto.StagedPlan{Derived: true}
	if printed := plan.ScopeNotice(); printed != "" {
		t.Fatalf("a scope that held was announced anyway: %q", printed)
	}
}

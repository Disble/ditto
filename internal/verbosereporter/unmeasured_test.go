package verbosereporter_test

import (
	"testing"

	"github.com/Disble/ditto/internal/dittotesting/fakelogger"
	"github.com/Disble/ditto/internal/dittotesting/fakereporter"
	"github.com/Disble/ditto/internal/verbosereporter"
	"github.com/stretchr/testify/assert"
)

// This lives here rather than beside the reporter it decorates, for the reason
// total_test.go does: the stack the release actually reads through is this one.

func TestUnmeasuredIsForwarded(t *testing.T) {
	assert.Equal(t, 6, verbosereporter.New(fakelogger.New(), fakereporter.Counting{Outside: 6}).Unmeasured(),
		"a decorator that drops a capability is refused by nothing")
}

// An unreadable count is UNKNOWN, not zero. Zero is the answer that says the
// whole scope was measured, so a decorator that cannot forward this must not be
// able to say that: the release would pass a run it could not judge.
func TestAnUnreadableUnmeasuredCountIsNotZero(t *testing.T) {
	assert.Equal(t, -1, verbosereporter.New(fakelogger.New(), fakereporter.Countless{}).Unmeasured())
}

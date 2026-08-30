package verbosereporter_test

import (
	"testing"

	"github.com/Disble/ditto/internal/dittotesting/fakelogger"
	"github.com/Disble/ditto/internal/dittotesting/fakereporter"
	"github.com/Disble/ditto/internal/verbosereporter"
	"github.com/stretchr/testify/assert"
)

// This lives here rather than beside the reporter it decorates, and the reason is
// a property of the gated path worth knowing: a gated mutant is compiled and run
// with the tests of ITS OWN package. The first version of these assertions sat in
// internal/consolereporter, every one of them passed, and the gate still reported
// four survivors -- the mutants in this file were never judged by them.

func TestTotalIsForwarded(t *testing.T) {
	assert.Equal(t, 47, verbosereporter.New(fakelogger.New(), fakereporter.Counting{Scored: 47}).Total(), "a decorator that drops a capability is refused by nothing")
}

// A delegate that cannot answer must read as UNKNOWN, not as zero. Zero means
// "the scope produced nothing to judge", which RunChanged treats as nothing to
// do -- so answering it for "I could not tell" would turn a genuinely failing
// run into a silent success. The sentinel is negative because a count cannot be.
func TestAnUnreadableCountIsNotZero(t *testing.T) {
	assert.Equal(t, -1, verbosereporter.New(fakelogger.New(), fakereporter.Countless{}).Total())
}

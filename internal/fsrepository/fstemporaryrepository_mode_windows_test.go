//go:build windows

package fsrepository_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// assertOverwrittenMode checks what Windows can actually promise.
//
// There is no umask here, and Go does not report POSIX permission bits: a
// writable file comes back as 0666 and a read-only one as 0444. Asserting an
// exact mode would pin an implementation detail of that mapping, so the claim
// made here is the one that matters — Overwrite left the file writable, and a
// later mutant can replace it.
func assertOverwrittenMode(t *testing.T, stat os.FileInfo) {
	t.Helper()

	assert.NotZero(t, stat.Mode().Perm()&0o200, "Overwrite must leave the file writable")
}

//go:build !windows

package fsrepository_test

import (
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

// assertOverwrittenMode checks the permission bits Overwrite leaves behind.
//
// Overwrite writes with os.ModePerm, so what reaches the disk is that minus
// the process umask. Umask is a POSIX notion with no Windows equivalent, which
// is why this assertion is split by platform rather than written once.
func assertOverwrittenMode(t *testing.T, stat os.FileInfo) {
	t.Helper()

	mask := syscall.Umask(0)
	defer syscall.Umask(mask)

	assert.Equal(t, os.ModePerm^os.FileMode(mask), stat.Mode()) //nolint:gosec
}

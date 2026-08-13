package fstemporarydir

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// removalAttempts and retryDelay bound how long a directory is waited on before
// it is left for the next run to reclaim.
//
// Windows keeps a handle on a directory for a moment after the process that was
// working in it exits, so an unguarded RemoveAll there fails with a busy or
// permission error on a sandbox that is genuinely finished with.
const (
	removalAttempts = 5
	retryDelay      = 200 * time.Millisecond
)

// FSTemporaryDir hands out sandboxes inside a single parent directory owned by
// this process.
//
// One parent rather than one directory per sandbox is what makes cleanup a
// single removal, and what makes an interrupted run leave one identifiable
// directory instead of a scattering of them. The owning process id is in the
// name so a later run can tell an abandoned parent from one that is still in
// use.
type FSTemporaryDir struct {
	prefix string

	mutex  sync.Mutex
	parent string
	next   int
}

func New(prefix string) *FSTemporaryDir {
	return &FSTemporaryDir{prefix: prefix}
}

// New returns a fresh directory inside this process's parent, creating the
// parent on first use.
func (f *FSTemporaryDir) New() string {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.parent == "" {
		parent, err := os.MkdirTemp("", fmt.Sprintf("%s%d-", f.prefix, os.Getpid()))
		if err != nil {
			panic(err)
		}

		f.parent = parent
	}

	child := filepath.Join(f.parent, strconv.Itoa(f.next))
	f.next++

	if err := os.MkdirAll(child, os.ModePerm); err != nil {
		panic(err)
	}

	return child
}

// RemoveAll deletes every sandbox this process handed out, and the parent.
//
// It reports the failure rather than panicking. This runs as cleanup, often
// while a test is already failing for its own reasons, and a panic here would
// replace that diagnosis with one about a temporary directory. A parent that
// survives is named after a process id, so it is reclaimable rather than lost.
func (f *FSTemporaryDir) RemoveAll() error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.parent == "" {
		return nil
	}

	parent := f.parent
	f.parent = ""
	f.next = 0

	return remove(parent)
}

func remove(path string) error {
	var err error

	for attempt := range removalAttempts {
		err = os.RemoveAll(path)
		if err == nil || errors.Is(err, fs.ErrNotExist) {
			return nil
		}

		if !errors.Is(err, fs.ErrPermission) {
			return fmt.Errorf("removing '%s': %w", path, err)
		}

		if attempt < removalAttempts-1 {
			time.Sleep(retryDelay)
		}
	}

	return fmt.Errorf("removing '%s' after %d attempts: %w", path, removalAttempts, err)
}

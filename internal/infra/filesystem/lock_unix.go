//go:build !windows

package filesystem

import (
	"errors"
	"os"
	"syscall"
)

// tryAcquireLock attempts to acquire an exclusive lock using flock (Unix) in non-blocking mode.
// Returns ErrLockBusy if the lock is already held.
func tryAcquireLock(file *os.File) error {
	err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == nil {
		return nil
	}

	if errors.Is(err, syscall.EWOULDBLOCK) {
		return ErrLockBusy
	}

	return err
}

// releaseLock releases the flock (Unix).
func releaseLock(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}

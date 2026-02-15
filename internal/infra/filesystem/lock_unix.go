//go:build !windows

package filesystem

import (
	"errors"
	"syscall"
	"time"
)

// lockRetryInterval is the interval between lock acquisition attempts.
const lockRetryInterval = 10 * time.Millisecond

// acquireLockPlatform acquires an exclusive lock using flock (Unix) with non-blocking retry.
func (fl *FileLock) acquireLockPlatform() error {
	for {
		// Try non-blocking lock acquisition
		err := syscall.Flock(int(fl.file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return nil
		}

		// If error is not EWOULDBLOCK, return the error
		if !errors.Is(err, syscall.EWOULDBLOCK) {
			return err
		}

		// Lock is busy, wait a bit and retry
		time.Sleep(lockRetryInterval)
	}
}

// releaseLockPlatform releases the flock (Unix).
func (fl *FileLock) releaseLockPlatform() error {
	return syscall.Flock(int(fl.file.Fd()), syscall.LOCK_UN)
}

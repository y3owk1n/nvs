//go:build windows

package filesystem

import (
	"errors"
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = kernel32.NewProc("LockFileEx")
	procUnlockFileEx = kernel32.NewProc("UnlockFileEx")
)

// Windows constants for LockFileEx.
const (
	lockFileExclusiveLock   = 0x00000002
	lockFileFailImmediately = 0x00000001
	maxDWORD                = 0xFFFFFFFF
	// errLockViolation is ERROR_LOCK_VIOLATION = 33.
	errLockViolation = 33
)

// tryAcquireLock attempts to acquire an exclusive lock using LockFileEx (Windows) in non-blocking mode.
// Returns ErrLockBusy if the lock is already held.
func tryAcquireLock(file *os.File) error {
	handle := file.Fd()

	var overlapped syscall.Overlapped

	ret, _, err := procLockFileEx.Call(
		handle,
		uintptr(lockFileExclusiveLock|lockFileFailImmediately),
		uintptr(0),
		uintptr(maxDWORD),
		uintptr(maxDWORD),
		uintptr(unsafe.Pointer(&overlapped)),
	)

	if ret == 0 {
		// Check if the error is ERROR_LOCK_VIOLATION (lock is busy)
		var errno syscall.Errno
		if errors.As(err, &errno) && errno == errLockViolation {
			return ErrLockBusy
		}

		return err
	}

	return nil
}

// releaseLock releases the LockFileEx lock (Windows).
func releaseLock(file *os.File) error {
	handle := file.Fd()

	var overlapped syscall.Overlapped

	ret, _, err := procUnlockFileEx.Call(
		handle,
		uintptr(0),
		uintptr(maxDWORD),
		uintptr(maxDWORD),
		uintptr(unsafe.Pointer(&overlapped)),
	)

	if ret == 0 {
		return err
	}

	return nil
}

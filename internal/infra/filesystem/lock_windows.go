//go:build windows

package filesystem

import (
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
)

// acquireLockPlatform acquires an exclusive lock using LockFileEx (Windows).
func (fl *FileLock) acquireLockPlatform() error {
	handle := fl.file.Fd()

	var overlapped syscall.Overlapped

	ret, _, err := procLockFileEx.Call(
		handle,
		uintptr(lockFileExclusiveLock),
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

// releaseLockPlatform releases the LockFileEx lock (Windows).
func (fl *FileLock) releaseLockPlatform() error {
	handle := fl.file.Fd()

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

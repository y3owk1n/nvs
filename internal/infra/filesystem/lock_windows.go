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

const (
	LOCKFILE_EXCLUSIVE_LOCK   = 0x00000002
	LOCKFILE_FAIL_IMMEDIATELY = 0x00000001
)

// acquireLockPlatform acquires an exclusive lock using LockFileEx (Windows).
func (fl *FileLock) acquireLockPlatform() error {
	handle := fl.file.Fd()

	var overlapped syscall.Overlapped

	ret, _, err := procLockFileEx.Call(
		uintptr(handle),
		uintptr(LOCKFILE_EXCLUSIVE_LOCK),
		uintptr(0),
		uintptr(0xFFFFFFFF),
		uintptr(0xFFFFFFFF),
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
		uintptr(handle),
		uintptr(0),
		uintptr(0xFFFFFFFF),
		uintptr(0xFFFFFFFF),
		uintptr(unsafe.Pointer(&overlapped)),
	)

	if ret == 0 {
		return err
	}
	return nil
}

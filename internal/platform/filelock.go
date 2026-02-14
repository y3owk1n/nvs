package platform

import (
	"os"
)

// FileLock provides cross-platform file locking.
type FileLock struct {
	file *os.File
	path string
}

// FilePerm is the default permission for lock files.
const FilePerm = 0o644

// NewFileLock creates a new file lock for the given path.
func NewFileLock(path string) (*FileLock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, FilePerm)
	if err != nil {
		return nil, err
	}

	return &FileLock{file: f, path: path}, nil
}

// Lock acquires an exclusive lock on the file.
func (fl *FileLock) Lock() error {
	return lockFile(fl.file)
}

// Unlock releases the lock on the file.
func (fl *FileLock) Unlock() error {
	return unlockFile(fl.file)
}

// Close closes the underlying file.
func (fl *FileLock) Close() error {
	return fl.file.Close()
}

// Remove removes the lock file from disk.
func (fl *FileLock) Remove() error {
	err := fl.file.Close()
	if err != nil {
		return err
	}

	return os.Remove(fl.path)
}

// Fd returns the file descriptor.
func (fl *FileLock) Fd() uintptr {
	return fl.file.Fd()
}

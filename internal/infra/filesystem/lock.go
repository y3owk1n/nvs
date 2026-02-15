package filesystem

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// FileLock provides production-grade file-based locking.
type FileLock struct {
	path string
	file *os.File
}

// NewFileLock creates a new file lock at the specified path.
func NewFileLock(path string) *FileLock {
	return &FileLock{path: path}
}

// ErrLockTimeout is returned when unable to acquire lock within timeout.
var ErrLockTimeout = errors.New("timeout waiting for file lock")

// ErrLockFailed is returned when lock acquisition fails.
var ErrLockFailed = errors.New("failed to acquire file lock")

const (
	// defaultDirPerms are permissions for creating lock directory.
	defaultDirPerms = 0o755
	// defaultFilePerms are permissions for creating lock file.
	defaultFilePerms = 0o644
)

// Lock attempts to acquire an exclusive lock with a timeout.
// The lock is automatically released when the process exits or Unlock is called.
func (fl *FileLock) Lock(ctx context.Context) error {
	// Create lock file directory if needed
	dir := filepath.Dir(fl.path)

	err := os.MkdirAll(dir, defaultDirPerms)
	if err != nil {
		return fmt.Errorf("failed to create lock directory: %w", err)
	}

	// Open or create lock file
	file, err := os.OpenFile(fl.path, os.O_CREATE|os.O_RDWR, defaultFilePerms)
	if err != nil {
		return fmt.Errorf("failed to open lock file: %w", err)
	}

	fl.file = file

	// Try to acquire lock with timeout
	done := make(chan error, 1)
	go func() {
		done <- fl.acquireLock()
	}()

	select {
	case err := <-done:
		if err != nil {
			closeErr := fl.file.Close()
			if closeErr != nil {
				logrus.Warnf("failed to close lock file after lock error: %v", closeErr)
			}

			fl.file = nil

			return err
		}

		return nil
	case <-ctx.Done():
		// Close file descriptor to interrupt flock syscall
		// This will cause the goroutine to return with an error
		closeErr := fl.file.Close()
		if closeErr != nil {
			logrus.Warnf("failed to close lock file after timeout: %v", closeErr)
		}

		// Wait for goroutine to finish
		<-done

		fl.file = nil

		return fmt.Errorf("%w: %w", ErrLockTimeout, ctx.Err())
	}
}

// Unlock releases the lock and closes the file.
func (fl *FileLock) Unlock() error {
	if fl.file == nil {
		return nil
	}

	var unlockErr, closeErr error

	// Release the lock
	unlockErr = fl.releaseLock()

	// Close the file
	closeErr = fl.file.Close()
	fl.file = nil

	// Remove lock file (best effort)
	_ = os.Remove(fl.path)

	if unlockErr != nil {
		return fmt.Errorf("failed to release lock: %w", unlockErr)
	}

	if closeErr != nil {
		return fmt.Errorf("failed to close lock file: %w", closeErr)
	}

	return nil
}

// WithLock executes the given function while holding the lock.
// The lock is automatically released after the function returns, even on panic.
func (fl *FileLock) WithLock(ctx context.Context, operation func() error) error {
	err := fl.Lock(ctx)
	if err != nil {
		return err
	}

	defer func() {
		unlockErr := fl.Unlock()
		if unlockErr != nil {
			logrus.Warnf("failed to unlock: %v", unlockErr)
		}
	}()

	return operation()
}

// DefaultLockTimeout is the default timeout for lock acquisition.
const DefaultLockTimeout = 30 * time.Second

// LockWithDefaultTimeout acquires lock with default timeout.
func (fl *FileLock) LockWithDefaultTimeout() error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultLockTimeout)
	defer cancel()

	return fl.Lock(ctx)
}

// acquireLock performs the actual platform-specific lock acquisition.
// This must be implemented in platform-specific files.
func (fl *FileLock) acquireLock() error {
	return fl.acquireLockPlatform()
}

// releaseLock performs the actual platform-specific lock release.
// This must be implemented in platform-specific files.
func (fl *FileLock) releaseLock() error {
	return fl.releaseLockPlatform()
}

package filesystem

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// FileLock provides production-grade file-based locking.
type FileLock struct {
	path string
	file *os.File
	mu   sync.Mutex
}

// NewFileLock creates a new file lock at the specified path.
func NewFileLock(path string) *FileLock {
	return &FileLock{path: path}
}

// ErrLockTimeout is returned when unable to acquire lock within timeout.
var ErrLockTimeout = errors.New("timeout waiting for file lock")

// ErrLockFailed is returned when lock acquisition fails.
var ErrLockFailed = errors.New("failed to acquire file lock")

// ErrLockHeld is returned when attempting to acquire a lock already held by the same process.
var ErrLockHeld = errors.New("lock already held")

const (
	// defaultDirPerms are permissions for creating lock directory.
	defaultDirPerms = 0o755
	// defaultFilePerms are permissions for creating lock file.
	defaultFilePerms = 0o644
	// lockPollInterval is the interval between lock acquisition attempts.
	lockPollInterval = 10 * time.Millisecond
)

// Lock attempts to acquire an exclusive lock with a timeout.
// The lock is automatically released when the process exits or Unlock is called.
func (fl *FileLock) Lock(ctx context.Context) error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	if fl.file != nil {
		return ErrLockHeld
	}

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

	// Try to acquire lock with timeout using polling
	ticker := time.NewTicker(lockPollInterval)
	defer ticker.Stop()

	for {
		err = tryAcquireLock(file)
		if err == nil {
			fl.file = file

			return nil
		}

		if !errors.Is(err, ErrLockBusy) {
			closeErr := file.Close()
			if closeErr != nil {
				logrus.Warnf("failed to close lock file: %v", closeErr)
			}

			return err
		}

		// Lock is busy, wait for ticker or context cancellation
		select {
		case <-ticker.C:
			// Try again
		case <-ctx.Done():
			closeErr := file.Close()
			if closeErr != nil {
				logrus.Warnf("failed to close lock file: %v", closeErr)
			}

			return fmt.Errorf("%w: %w", ErrLockTimeout, ctx.Err())
		}
	}
}

// ErrLockBusy indicates the lock is currently held by another process.
var ErrLockBusy = errors.New("lock is busy")

// Unlock releases the lock and closes the file.
func (fl *FileLock) Unlock() error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	if fl.file == nil {
		return nil
	}

	var unlockErr, closeErr error

	// Release the lock
	unlockErr = releaseLock(fl.file)

	// Close the file
	closeErr = fl.file.Close()
	fl.file = nil

	// Note: we intentionally do NOT remove the lock file here.
	// Removing it would create a race condition where another process
	// could open+lock a new file at the same path (different inode)
	// while a third process still holds a lock on the old (deleted) inode.

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

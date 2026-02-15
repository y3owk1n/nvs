//nolint:testpackage
package filesystem

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestFileLock_BasicLockUnlock(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")
	lock := NewFileLock(lockPath)

	// Lock should succeed
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := lock.Lock(ctx)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	// Unlock should succeed
	err = lock.Unlock()
	if err != nil {
		t.Fatalf("Failed to release lock: %v", err)
	}

	// Lock file should be cleaned up
	_, err = os.Stat(lockPath)
	if !os.IsNotExist(err) {
		t.Error("Lock file should be removed after unlock")
	}
}

func TestFileLock_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")

	var (
		counter   int
		waitGroup sync.WaitGroup
	)

	numGoroutines := 10

	// Run multiple goroutines that all try to increment a counter
	for range numGoroutines {
		waitGroup.Go(func() {
			lock := NewFileLock(lockPath)

			err := lock.LockWithDefaultTimeout()
			if err != nil {
				t.Errorf("Failed to acquire lock: %v", err)

				return
			}

			defer func() {
				unlockErr := lock.Unlock()
				if unlockErr != nil {
					t.Logf("Failed to unlock: %v", unlockErr)
				}
			}()

			// Critical section: increment counter
			current := counter

			time.Sleep(10 * time.Millisecond) // Simulate some work

			counter = current + 1
		})
	}

	waitGroup.Wait()

	if counter != numGoroutines {
		t.Errorf("Expected counter to be %d, got %d", numGoroutines, counter)
	}
}

func TestFileLock_Timeout(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")

	// First, acquire a lock and hold it
	lock1 := NewFileLock(lockPath)

	err := lock1.LockWithDefaultTimeout()
	if err != nil {
		t.Fatalf("Failed to acquire first lock: %v", err)
	}

	defer func() {
		unlockErr := lock1.Unlock()
		if unlockErr != nil {
			t.Logf("Failed to unlock first lock: %v", unlockErr)
		}
	}()

	// Try to acquire the same lock with a short timeout
	lock2 := NewFileLock(lockPath)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = lock2.Lock(ctx)
	if err == nil {
		t.Error("Expected timeout error when trying to acquire held lock")

		unlockErr := lock2.Unlock()
		if unlockErr != nil {
			t.Logf("Failed to unlock second lock: %v", unlockErr)
		}
	}

	if !errors.Is(err, ErrLockTimeout) {
		t.Logf("Got error: %v (type: %T)", err, err)
	}
}

func TestFileLock_WithLock(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")
	lock := NewFileLock(lockPath)

	var executed bool

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := lock.WithLock(ctx, func() error {
		executed = true

		return nil
	})
	if err != nil {
		t.Fatalf("WithLock failed: %v", err)
	}

	if !executed {
		t.Error("Function inside WithLock was not executed")
	}

	// Lock file should be cleaned up
	_, err = os.Stat(lockPath)
	if !os.IsNotExist(err) {
		t.Error("Lock file should be removed after WithLock")
	}
}

func TestFileLock_UnlockWithoutLock(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")
	lock := NewFileLock(lockPath)

	// Unlocking without locking should not panic or error
	err := lock.Unlock()
	if err != nil {
		t.Errorf("Unlock without lock should succeed, got: %v", err)
	}
}

func TestFileLock_MultipleLocksSameProcess(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")

	// Same process can acquire the same lock multiple times (flock behavior)
	// This is actually platform-dependent, but on Unix it should succeed
	lock1 := NewFileLock(lockPath)

	err := lock1.LockWithDefaultTimeout()
	if err != nil {
		t.Fatalf("Failed to acquire first lock: %v", err)
	}

	// Try to acquire again (should succeed on Unix with flock)
	lock2 := NewFileLock(lockPath)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = lock2.Lock(ctx)
	if err == nil {
		// If we got the lock, release it
		unlockErr := lock2.Unlock()
		if unlockErr != nil {
			t.Logf("Failed to unlock second lock: %v", unlockErr)
		}
	}

	// Clean up
	unlockErr := lock1.Unlock()
	if unlockErr != nil {
		t.Logf("Failed to unlock first lock: %v", unlockErr)
	}
}

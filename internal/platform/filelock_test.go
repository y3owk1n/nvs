package platform_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	platform "github.com/y3owk1n/nvs/internal/platform"
)

func TestFileLock_BasicLock(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")

	lock, err := platform.NewFileLock(lockPath)
	if err != nil {
		t.Fatalf("failed to create lock: %v", err)
	}

	err = lock.Lock()
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	err = lock.Unlock()
	if err != nil {
		t.Fatalf("failed to release lock: %v", err)
	}

	err = lock.Close()
	if err != nil {
		t.Fatalf("failed to close lock: %v", err)
	}
}

func TestFileLock_ReacquireAfterRelease(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")

	lock, err := platform.NewFileLock(lockPath)
	if err != nil {
		t.Fatalf("failed to create lock: %v", err)
	}

	defer func() {
		_ = lock.Close()
	}()

	err = lock.Lock()
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	err = lock.Unlock()
	if err != nil {
		t.Fatalf("failed to release lock: %v", err)
	}

	err = lock.Lock()
	if err != nil {
		t.Fatalf("failed to reacquire lock: %v", err)
	}

	err = lock.Unlock()
	if err != nil {
		t.Fatalf("failed to release lock: %v", err)
	}
}

func TestFileLock_Concurrent(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")

	var waitGroup sync.WaitGroup

	acquired := make(chan struct{})
	secondAcquired := make(chan struct{})

	waitGroup.Add(2)

	go func() {
		defer waitGroup.Done()

		lock1, err := platform.NewFileLock(lockPath)
		if err != nil {
			t.Errorf("failed to create lock: %v", err)

			return
		}

		defer func() {
			_ = lock1.Close()
		}()

		err = lock1.Lock()
		if err != nil {
			t.Errorf("failed to acquire lock: %v", err)

			return
		}

		close(acquired)

		err = lock1.Unlock()
		if err != nil {
			t.Errorf("failed to release lock: %v", err)
		}
	}()

	go func() {
		defer waitGroup.Done()

		<-acquired

		lock2, err := platform.NewFileLock(lockPath)
		if err != nil {
			t.Errorf("failed to create lock: %v", err)

			return
		}

		defer func() {
			_ = lock2.Close()
		}()

		err = lock2.Lock()
		if err != nil {
			t.Errorf("failed to acquire lock: %v", err)

			return
		}

		close(secondAcquired)

		err = lock2.Unlock()
		if err != nil {
			t.Errorf("failed to release lock: %v", err)
		}
	}()

	waitGroup.Wait()
}

func TestFileLock_InvalidPath(t *testing.T) {
	lockPath := "/nonexistent/path/test.lock"

	_, err := platform.NewFileLock(lockPath)
	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
}

func TestFileLock_LockFilePersists(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")

	lock, err := platform.NewFileLock(lockPath)
	if err != nil {
		t.Fatalf("failed to create lock: %v", err)
	}

	err = lock.Lock()
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	err = lock.Close()
	if err != nil {
		t.Fatalf("failed to close lock: %v", err)
	}

	_, err = os.Stat(lockPath)
	if err != nil {
		t.Errorf("lock file should exist: %v", err)
	}
}

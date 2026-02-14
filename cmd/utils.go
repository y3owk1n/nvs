package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/internal/platform"
)

// copyDir recursively copies a directory from src to dst.
// Note: Relative symlinks may point to incorrect locations if the target is outside the src tree.
// For atomicity, uses a temporary directory and renames on success to avoid partial copies.
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	dstDir := filepath.Dir(dst)

	tempDst, err := os.MkdirTemp(dstDir, "copy-temp-")
	if err != nil {
		return err
	}

	defer func() {
		err := os.RemoveAll(tempDst)
		if err != nil {
			logrus.Warnf("Failed to clean up temp directory %s: %v", tempDst, err)
		}
	}()

	err = os.MkdirAll(tempDst, srcInfo.Mode())
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(tempDst, entry.Name())

		if entry.Type()&os.ModeSymlink != 0 {
			link, err := os.Readlink(srcPath)
			if err != nil {
				return err
			}

			err = os.Symlink(link, dstPath)
			if err != nil {
				return err
			}

			continue
		}

		if entry.IsDir() {
			err := copyDir(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			err := copyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}

	return os.Rename(tempDst, dst)
}

// copyDirLocked copies a directory with file locking for thread-safe operation.
func copyDirLocked(src, dst string) error {
	lockFile := src + ".lock"

	lockFd, lockErr := platform.NewFileLock(lockFile)
	if lockErr != nil {
		return fmt.Errorf("failed to open lock file: %w", lockErr)
	}

	defer func() {
		_ = lockFd.Unlock()
		_ = lockFd.Close()
	}()

	lockErr = lockFd.Lock()
	if lockErr != nil {
		return fmt.Errorf("failed to acquire lock: %w", lockErr)
	}

	return copyDir(src, dst)
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}

	defer func() {
		closeErr := srcFile.Close()
		if closeErr != nil {
			logrus.Warnf("Failed to close source file: %v", closeErr)
		}
	}()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}

	defer func() {
		closeErr := dstFile.Close()
		if closeErr != nil {
			logrus.Warnf("Failed to close destination file: %v", closeErr)
		}
	}()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	return dstFile.Sync()
}

// outputJSON marshals the given data to indented JSON and prints it to stdout.
// Returns an error if marshaling or writing fails.
func outputJSON(data any) error {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %w", err)
	}

	_, err = os.Stdout.WriteString(string(jsonBytes) + "\n")
	if err != nil {
		return fmt.Errorf("error writing JSON output: %w", err)
	}

	return nil
}

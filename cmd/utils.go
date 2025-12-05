package cmd

import (
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// copyDir recursively copies a directory from src to dst.
// Note: Relative symlinks may point to incorrect locations if the target is outside the src tree.
// For atomicity, uses a temporary directory and renames on success to avoid partial copies.
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create temp directory next to dst for atomic copy
	dstDir := filepath.Dir(dst)
	tempDst, err := os.MkdirTemp(dstDir, "copy-temp-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDst) // Clean up temp on failure

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

		// Handle symlinks
		// Note: Relative symlinks may break if target is outside src tree
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

	// Atomic rename on success
	return os.Rename(tempDst, dst)
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

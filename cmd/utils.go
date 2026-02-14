package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
)

// Windows error code for missing privilege to create symbolic links.
const errorPrivilegeNotHeld = 1314 // ERROR_PRIVILEGE_NOT_HELD

// copyDir recursively copies a directory from src to dst.
// Handles relative symlinks by adjusting paths when targets are outside the src tree.
// On Windows, falls back to copying target content if symlink creation fails due to permissions.
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

		// Handle symlinks
		// For relative symlinks pointing outside src tree, adjust the target path
		if entry.Type()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(srcPath)
			if err != nil {
				return err
			}

			// If relative symlink, resolve it and recompute relative path from dst
			if !filepath.IsAbs(linkTarget) {
				// Resolve symlink target relative to the symlink's directory, not CWD
				// Use srcPath's directory to properly resolve the target
				symlinkDir := filepath.Dir(srcPath)

				absTarget, err := filepath.Abs(filepath.Join(symlinkDir, linkTarget))
				if err != nil {
					return err
				}

				// Ensure src is absolute for reliable comparison
				absSrc, err := filepath.Abs(src)
				if err != nil {
					return err
				}

				// Check if target is outside source tree
				relToSrc, err := filepath.Rel(absSrc, absTarget)
				if err != nil {
					return err
				}

				// If outside src tree (starts with ".."), recompute relative path from dst
				if relToSrc == ".." ||
					strings.HasPrefix(relToSrc, ".."+string(filepath.Separator)) {
					dstDir := filepath.Dir(dstPath)

					linkTarget, err = filepath.Rel(dstDir, absTarget)
					if err != nil {
						return err
					}
				}
			}

			err = os.Symlink(linkTarget, dstPath)
			if err != nil {
				// On Windows, symlink creation requires admin privileges (ERROR_PRIVILEGE_NOT_HELD = 1314)
				// Fall back to copying the target content
				isWinPermError := errors.Is(err, os.ErrPermission)
				if runtime.GOOS == "windows" {
					var errno syscall.Errno
					if errors.As(err, &errno) {
						isWinPermError = isWinPermError || errno == errorPrivilegeNotHeld
					}
				}

				if runtime.GOOS == "windows" && isWinPermError {
					logrus.Warnf(
						"Cannot create symlink on Windows without admin rights, copying target instead: %s",
						srcPath,
					)

					// Resolve the symlink target
					resolvedPath, statErr := os.Stat(srcPath)
					if statErr != nil {
						return statErr
					}

					// If it's a directory, recurse; otherwise copy the file
					if resolvedPath.IsDir() {
						err = copyDir(srcPath, dstPath)
					} else {
						err = copyFile(srcPath, dstPath)
					}

					if err != nil {
						return err
					}

					continue
				}

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

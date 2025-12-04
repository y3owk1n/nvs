// Package filesystem provides filesystem utilities.
package filesystem

import (
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

const (
	DirPerm = 0o755
)

// CopyFile copies the content of the file from src to dst,
// sets the destination file's permissions to 0755, and returns an error if any step fails.
func CopyFile(src, dst string) error {
	var err error

	inputFile, err := os.Open(src)
	if err != nil {
		return err
	}

	defer func() {
		cerr := inputFile.Close()
		if cerr != nil {
			logrus.Warnf("Failed to close source file %s: %v", src, cerr)
		}
	}()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()

	_, err = io.Copy(out, inputFile)
	if err != nil {
		return err
	}

	err = os.Chmod(dst, DirPerm)
	if err != nil {
		return err
	}

	return nil
}

// ClearDirectory removes all contents within the specified directory.
// It returns an error if any removal fails.
func ClearDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())

		err := os.RemoveAll(path)
		if err != nil {
			return err
		}
	}

	return nil
}

package filesystem

import (
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

const (
	// DirPerm is the permission for directories.
	DirPerm = 0o755
)

// CopyFile copies the content of the file from src to dst,
// sets the destination file's permissions to the specified mode, and returns an error if any step fails.
func CopyFile(src, dst string, mode os.FileMode) (err error) {
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

	err = os.Chmod(dst, mode)
	if err != nil {
		return err
	}

	return err
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

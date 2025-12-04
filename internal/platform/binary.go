package platform

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

// FindNvimBinary walks through the given directory to find a Neovim binary.
// On Windows it searches for "nvim.exe" (or prefixed names with .exe),
// while on other OSes it looks for "nvim" (or prefixed names) that are executable.
// The function returns the full path to the binary or an empty string if not found.
func FindNvimBinary(dir string) string {
	var binaryPath string

	err := filepath.WalkDir(dir, func(path string, dirEntry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !dirEntry.IsDir() {
			name := dirEntry.Name()
			if runtime.GOOS == WindowsOS {
				if strings.EqualFold(name, "nvim.exe") ||
					(strings.HasPrefix(strings.ToLower(name), "nvim-") && filepath.Ext(name) == ".exe") {
					// Go two levels up: ../../nvim-win64
					binaryPath = filepath.Dir(filepath.Dir(path))

					return io.EOF // break early
				}
			} else {
				if name == "nvim" || strings.HasPrefix(name, "nvim-") {
					info, err := dirEntry.Info()
					if err == nil && info.Mode()&0o111 != 0 {
						binaryPath = path

						return io.EOF // break early
					}
				}
			}
		}

		return nil
	})
	if err != nil && !errors.Is(err, io.EOF) {
		logrus.Warnf("Failed to walk through nvim directory: %v", err)
	}

	return binaryPath
}

package utils

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// IsInstalled checks if a version is installed by verifying the existence of the version directory.
func IsInstalled(versionsDir, version string) bool {
	_, err := os.Stat(filepath.Join(versionsDir, version))
	return !os.IsNotExist(err)
}

// ListInstalledVersions returns a list of installed version directories.
func ListInstalledVersions(versionsDir string) ([]string, error) {
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		return nil, err
	}
	var versions []string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "current" {
			versions = append(versions, entry.Name())
		}
	}
	return versions, nil
}

// UpdateSymlink updates a symlink to point to the target.
func UpdateSymlink(target, link string) error {
	if _, err := os.Lstat(link); err == nil {
		if err := os.Remove(link); err != nil {
			return err
		}
	}
	return os.Symlink(target, link)
}

// GetCurrentVersion returns the currently active version by reading the "current" symlink.
func GetCurrentVersion(versionsDir string) (string, error) {
	link := filepath.Join(versionsDir, "current")
	target, err := os.Readlink(link)
	if err != nil {
		return "", err
	}
	return filepath.Base(target), nil
}

// FindNvimBinary searches for an executable named "nvim" or starting with "nvim-".
func FindNvimBinary(dir string) string {
	var binaryPath string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip error
		}
		if !d.IsDir() {
			name := d.Name()
			if runtime.GOOS == "windows" {
				// Check for nvim.exe or files starting with nvim- (with .exe extension)
				if name == "nvim.exe" || (strings.HasPrefix(name, "nvim-") && filepath.Ext(name) == ".exe") {
					binaryPath = path
					return io.EOF // break early
				}
			} else {
				// Unix-like systems: check for "nvim" or names starting with "nvim-"
				if name == "nvim" || strings.HasPrefix(name, "nvim-") {
					info, err := d.Info()
					if err == nil && info.Mode()&0111 != 0 {
						binaryPath = path
						return io.EOF // break early
					}
				}
			}
		}
		return nil
	})
	if err != nil && err != io.EOF {
		logrus.Fatalf("Failed to walk through nvim directory: %v", err)
	}

	return binaryPath
}

// GetInstalledReleaseIdentifier reads the version.txt file from an installed version.
func GetInstalledReleaseIdentifier(versionsDir, alias string) (string, error) {
	versionFile := filepath.Join(versionsDir, alias, "version.txt")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func LaunchNvimWithConfig(configName string) {
	home, err := os.UserHomeDir()
	if err != nil {
		logrus.Fatalf("Failed to get home directory: %v", err)
	}
	configDir := filepath.Join(home, ".config", configName)

	info, err := os.Stat(configDir)
	if os.IsNotExist(err) || !info.IsDir() {
		fmt.Printf("Error: configuration '%s' does not exist in ~/.config\n", configName)
		return
	}

	os.Setenv("NVIM_APPNAME", configName)
	fmt.Printf("Switched NVIM_APPNAME to %s\n", configName)

	nvimExec, err := exec.LookPath("nvim")
	if err != nil {
		logrus.Fatalf("nvim not found in PATH: %v", err)
	}
	launch := exec.Command(nvimExec)
	launch.Env = append(os.Environ(), "NVIM_APPNAME="+configName)
	launch.Stdin = os.Stdin
	launch.Stdout = os.Stdout
	launch.Stderr = os.Stderr
	if err := launch.Run(); err != nil {
		logrus.Fatalf("Failed to launch nvim: %v", err)
	}
}

func ClearDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}
	return nil
}

func TimeFormat(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	return t.Format("2006-01-02")
}

func ColorizeRow(row []string, color, reset string) []string {
	colored := make([]string, len(row))
	for i, cell := range row {
		colored[i] = fmt.Sprintf("%s%s%s", color, cell, reset)
	}
	return colored
}

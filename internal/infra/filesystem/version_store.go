// Package filesystem provides version storage operations on the filesystem.
package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/internal/domain/version"
)

const (
	dirPerm   = 0o755
	windowsOS = "windows"
)

// VersionStore implements version.Manager for filesystem-based storage.
type VersionStore struct{}

// New creates a new VersionStore instance.
func New() *VersionStore {
	return &VersionStore{}
}

// List returns all installed versions.
func (s *VersionStore) List(versionsDir string) ([]version.Version, error) {
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read versions directory: %w", err)
	}

	var versions []version.Version

	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "current" {
			// Read version.txt to get full info
			versionFile := filepath.Join(versionsDir, entry.Name(), "version.txt")
			data, err := os.ReadFile(versionFile)

			var commitHash string
			if err == nil {
				commitHash = strings.TrimSpace(string(data))
			}

			// Determine version type
			vType := determineVersionType(entry.Name())

			versions = append(versions, version.New(
				entry.Name(),
				vType,
				entry.Name(),
				commitHash,
			))
		}
	}

	return versions, nil
}

// Current returns the currently active version.
func (s *VersionStore) Current(versionsDir string) (version.Version, error) {
	link := filepath.Join(versionsDir, "current")

	info, err := os.Lstat(link)
	if err != nil {
		return version.Version{}, fmt.Errorf("failed to lstat current: %w", err)
	}

	var targetName string

	// Handle symlink
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(link)
		if err != nil {
			return version.Version{}, fmt.Errorf("failed to read symlink: %w", err)
		}
		targetName = filepath.Base(target)
	} else if info.IsDir() {
		// Windows junction
		targetName = filepath.Base(link)
	} else {
		return version.Version{}, version.ErrNoCurrentVersion
	}

	// Read version info
	versionFile := filepath.Join(versionsDir, targetName, "version.txt")
	data, err := os.ReadFile(versionFile)

	var commitHash string
	if err == nil {
		commitHash = strings.TrimSpace(string(data))
	}

	vType := determineVersionType(targetName)

	return version.New(targetName, vType, targetName, commitHash), nil
}

// Switch activates a specific version.
func (s *VersionStore) Switch(v version.Version, versionsDir, binDir string) error {
	versionPath := filepath.Join(versionsDir, v.Name())
	currentLink := filepath.Join(versionsDir, "current")

	// Update current symlink
	if err := updateSymlink(versionPath, currentLink, true); err != nil {
		return fmt.Errorf("failed to update current symlink: %w", err)
	}

	// Find nvim binary
	nvimExec := findNvimBinary(versionPath)
	if nvimExec == "" {
		return fmt.Errorf("%w in %s", ErrBinaryNotFound, versionPath)
	}

	// Update global binary link
	targetBin := filepath.Join(binDir, "nvim")

	// Remove existing link
	if _, err := os.Lstat(targetBin); err == nil {
		if err := os.Remove(targetBin); err != nil {
			logrus.Warnf("Failed to remove existing global bin: %v", err)
		}
	}

	// Create new link
	isDir := runtime.GOOS == windowsOS
	if err := updateSymlink(nvimExec, targetBin, isDir); err != nil {
		return fmt.Errorf("failed to create global nvim link: %w", err)
	}

	logrus.Debugf("Switched to version: %s", v.Name())

	return nil
}

// IsInstalled checks if a version is installed.
func (s *VersionStore) IsInstalled(v version.Version, versionsDir string) bool {
	_, err := os.Stat(filepath.Join(versionsDir, v.Name()))
	return !os.IsNotExist(err)
}

// Uninstall removes an installed version.
func (s *VersionStore) Uninstall(v version.Version, versionsDir string, force bool) error {
	// Check if version is current
	if !force {
		current, err := s.Current(versionsDir)
		if err == nil && current.Name() == v.Name() {
			return version.ErrVersionInUse
		}
	}

	versionPath := filepath.Join(versionsDir, v.Name())

	if err := os.RemoveAll(versionPath); err != nil {
		return fmt.Errorf("failed to remove version directory: %w", err)
	}

	return nil
}

// GetInstalledReleaseIdentifier returns the release identifier (e.g. commit hash) for an installed version.
func (s *VersionStore) GetInstalledReleaseIdentifier(versionName, versionsDir string) (string, error) {
	versionFile := filepath.Join(versionsDir, versionName, "version.txt")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return "", fmt.Errorf("failed to read version file: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// updateSymlink creates or updates a symlink.
func updateSymlink(target, link string, isDir bool) error {
	// Remove old link if exists
	if _, err := os.Lstat(link); err == nil {
		if err := os.Remove(link); err != nil {
			return err
		}
	}

	// Try normal symlink
	if err := os.Symlink(target, link); err == nil {
		return nil
	} else if runtime.GOOS != windowsOS {
		return err
	}

	// Windows fallback
	var cmd *exec.Cmd
	if isDir {
		cmd = exec.CommandContext(context.Background(), "cmd", "/C", "mklink", "/J", link, target)
	} else {
		cmd = exec.CommandContext(context.Background(), "cmd", "/C", "mklink", "/H", link, target)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create Windows link: %w", err)
	}

	return nil
}

// findNvimBinary searches for the nvim binary in a directory.
func findNvimBinary(dir string) string {
	var binaryPath string

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			name := d.Name()
			if runtime.GOOS == windowsOS {
				if strings.EqualFold(name, "nvim.exe") ||
					(strings.HasPrefix(strings.ToLower(name), "nvim-") && filepath.Ext(name) == ".exe") {
					binaryPath = filepath.Dir(filepath.Dir(path))
					return io.EOF
				}
			} else {
				if name == "nvim" || strings.HasPrefix(name, "nvim-") {
					info, err := d.Info()
					if err == nil && info.Mode()&0o111 != 0 {
						binaryPath = path
						return io.EOF
					}
				}
			}
		}

		return nil
	})

	if err != nil && !errors.Is(err, io.EOF) {
		logrus.Warnf("Error walking directory: %v", err)
	}

	return binaryPath
}

// determineVersionType determines the version type from the name.
func determineVersionType(name string) version.Type {
	switch {
	case name == "stable":
		return version.TypeStable
	case name == "nightly" || strings.HasPrefix(name, "nightly-"):
		return version.TypeNightly
	case len(name) == 7 || len(name) == 40:
		// Likely a commit hash
		return version.TypeCommit
	default:
		return version.TypeTag
	}
}

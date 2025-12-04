// Package filesystem provides version storage operations on the filesystem.
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
	domainversion "github.com/y3owk1n/nvs/internal/domain/version"
)

const (
	dirPerm   = 0o755
	windowsOS = "windows"
)

// VersionStore implements domainversion.Manager for filesystem-based storage.
type VersionStore struct {
	config *Config
}

// Config holds configuration for the version store.
type Config struct {
	VersionsDir  string
	GlobalBinDir string
}

// New creates a new VersionStore.
func New(config *Config) *VersionStore {
	return &VersionStore{
		config: config,
	}
}

// List returns all installed versions.
func (s *VersionStore) List() ([]domainversion.Version, error) {
	entries, err := os.ReadDir(s.config.VersionsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read versions directory: %w", err)
	}

	var versions []domainversion.Version

	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "current" {
			// Read version.txt to get full info
			versionFile := filepath.Join(s.config.VersionsDir, entry.Name(), "version.txt")
			data, err := os.ReadFile(versionFile)

			var commitHash string
			if err == nil {
				commitHash = strings.TrimSpace(string(data))
			}

			// Determine version type
			vType := determineVersionType(entry.Name())

			versions = append(versions, domainversion.New(
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
func (s *VersionStore) Current() (domainversion.Version, error) {
	link := filepath.Join(s.config.VersionsDir, "current")

	info, err := os.Lstat(link)
	if err != nil {
		return domainversion.Version{}, fmt.Errorf("failed to lstat current: %w", err)
	}

	var targetName string

	// Handle symlink
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		target, err := os.Readlink(link)
		if err != nil {
			return domainversion.Version{}, fmt.Errorf("failed to read symlink: %w", err)
		}

		targetName = filepath.Base(target)
	case info.IsDir():
		// Windows junction - resolve using EvalSymlinks
		resolved, err := filepath.EvalSymlinks(link)
		if err != nil {
			return domainversion.Version{}, fmt.Errorf("failed to resolve junction: %w", err)
		}

		targetName = filepath.Base(resolved)
	default:
		return domainversion.Version{}, domainversion.ErrNoCurrentVersion
	}

	// Read version info
	versionFile := filepath.Join(s.config.VersionsDir, targetName, "version.txt")
	data, err := os.ReadFile(versionFile)

	var commitHash string
	if err == nil {
		commitHash = strings.TrimSpace(string(data))
	}

	vType := determineVersionType(targetName)

	return domainversion.New(targetName, vType, targetName, commitHash), nil
}

// Switch activates a specific version.
func (s *VersionStore) Switch(version domainversion.Version) error {
	versionPath := filepath.Join(s.config.VersionsDir, version.Name())
	currentLink := filepath.Join(s.config.VersionsDir, "current")

	// Update current symlink
	err := updateSymlink(versionPath, currentLink, true)
	if err != nil {
		return fmt.Errorf("failed to update current symlink: %w", err)
	}

	// Find nvim link target
	nvimExec := findNvimLinkTarget(versionPath)
	if nvimExec == "" {
		return fmt.Errorf("%w in %s", ErrBinaryNotFound, versionPath)
	}

	// Update global binary link
	targetBin := filepath.Join(s.config.GlobalBinDir, "nvim")

	// Remove existing link
	_, err = os.Lstat(targetBin)
	if err == nil {
		err = os.Remove(targetBin)
		if err != nil {
			logrus.Warnf("Failed to remove existing global bin: %v", err)
		}
	}

	// Create new link
	isDir := runtime.GOOS == windowsOS

	err = updateSymlink(nvimExec, targetBin, isDir)
	if err != nil {
		return fmt.Errorf("failed to create global nvim link: %w", err)
	}

	logrus.Debugf("Switched to version: %s", version.Name())

	return nil
}

// IsInstalled checks if a version is installed.
func (s *VersionStore) IsInstalled(v domainversion.Version) bool {
	_, err := os.Stat(filepath.Join(s.config.VersionsDir, v.Name()))

	return !os.IsNotExist(err)
}

// Uninstall removes an installed version.
func (s *VersionStore) Uninstall(version domainversion.Version, force bool) error {
	// Check if version is current
	if !force {
		current, err := s.Current()
		if err == nil && current.Name() == version.Name() {
			return domainversion.ErrVersionInUse
		}
	}

	versionPath := filepath.Join(s.config.VersionsDir, version.Name())

	err := os.RemoveAll(versionPath)
	if err != nil {
		return fmt.Errorf("failed to remove version directory: %w", err)
	}

	return nil
}

// GetInstalledReleaseIdentifier returns the release identifier (e.g. commit hash) for an installed version.
func (s *VersionStore) GetInstalledReleaseIdentifier(versionName string) (string, error) {
	versionFile := filepath.Join(s.config.VersionsDir, versionName, "version.txt")

	data, err := os.ReadFile(versionFile)
	if err != nil {
		return "", fmt.Errorf("failed to read version file: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

// updateSymlink creates or updates a symlink.
func updateSymlink(target, link string, isDir bool) error {
	// Remove old link if exists
	_, statErr := os.Lstat(link)
	if statErr == nil {
		statErr = os.Remove(link)
		if statErr != nil {
			return statErr
		}
	}

	// Try normal symlink
	err := os.Symlink(target, link)
	if err == nil {
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

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to create Windows link: %w", err)
	}

	return nil
}

// findNvimLinkTarget searches for the nvim binary and returns the appropriate link target.
// On Unix, returns the binary path. On Windows, returns the version directory for junction creation.
func findNvimLinkTarget(dir string) string {
	var binaryPath string

	err := filepath.WalkDir(dir, func(path string, dirEntry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !dirEntry.IsDir() {
			name := dirEntry.Name()
			if runtime.GOOS == windowsOS {
				if strings.EqualFold(name, "nvim.exe") ||
					(strings.HasPrefix(strings.ToLower(name), "nvim-") && filepath.Ext(name) == ".exe") {
					binaryPath = filepath.Dir(filepath.Dir(path))

					return io.EOF
				}
			} else {
				if name == "nvim" || strings.HasPrefix(name, "nvim-") {
					info, err := dirEntry.Info()
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
func determineVersionType(name string) domainversion.Type {
	switch {
	case name == "stable":
		return domainversion.TypeStable
	case strings.HasPrefix(strings.ToLower(name), "nightly"):
		return domainversion.TypeNightly
	case domainversion.IsCommitHash(name):
		return domainversion.TypeCommit
	default:
		return domainversion.TypeTag
	}
}

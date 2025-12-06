// Package config provides the application service for Neovim configuration management.
package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/platform"
)

// Service handles Neovim configuration operations.
type Service struct{}

// New creates a new config Service.
func New() *Service {
	return &Service{}
}

// List returns all available Neovim configurations.
func (s *Service) List() ([]string, error) {
	configDir, err := platform.GetNvimConfigBaseDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config base dir: %w", err)
	}

	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	var configs []string

	for _, entry := range entries {
		entryPath := filepath.Join(configDir, entry.Name())

		info, err := os.Lstat(entryPath)
		if err != nil {
			logrus.Warnf("Failed to lstat %s: %v", entryPath, err)

			continue
		}

		var isDir bool

		// Handle symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			resolvedPath, err := os.Readlink(entryPath)
			if err != nil {
				logrus.Warnf("Failed to resolve symlink %s: %v", entry.Name(), err)

				continue
			}

			// Make relative paths absolute based on the symlink's directory
			if !filepath.IsAbs(resolvedPath) {
				resolvedPath = filepath.Clean(filepath.Join(filepath.Dir(entryPath), resolvedPath))
			}

			targetInfo, err := os.Stat(resolvedPath)
			if err != nil {
				logrus.Warnf("Failed to stat resolved path for %s: %v", entry.Name(), err)

				continue
			}

			isDir = targetInfo.IsDir()
		} else {
			isDir = info.IsDir()
		}

		// Add directories containing "nvim"
		if isDir {
			name := strings.ToLower(entry.Name())
			if strings.Contains(name, "nvim") {
				// Exclude nvim-data on Windows
				if runtime.GOOS == constants.WindowsOS && strings.HasSuffix(name, "-data") {
					continue
				}

				configs = append(configs, entry.Name())
			}
		}
	}

	return configs, nil
}

// Launch launches Neovim with the specified configuration.
func (s *Service) Launch(ctx context.Context, configName string) error {
	baseConfigDir, err := platform.GetNvimConfigBaseDir()
	if err != nil {
		return fmt.Errorf("failed to determine config base dir: %w", err)
	}

	configDir := filepath.Join(baseConfigDir, configName)

	// Verify config exists
	info, err := os.Stat(configDir)
	if os.IsNotExist(err) || (err == nil && !info.IsDir()) {
		return fmt.Errorf("%w: %s", ErrConfigNotFound, configName)
	}

	if err != nil {
		return fmt.Errorf("failed to stat config directory: %w", err)
	}

	// Find nvim executable
	nvimExec, err := exec.LookPath("nvim")
	if err != nil {
		return fmt.Errorf("nvim not found in PATH: %w", err)
	}

	// Launch Neovim
	cmd := exec.CommandContext(ctx, nvimExec)

	cmd.Env = append(os.Environ(), "NVIM_APPNAME="+configName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to launch nvim: %w", err)
	}

	return nil
}

// ErrConfigNotFound is returned when a configuration is not found.
var ErrConfigNotFound = errors.New("configuration not found")

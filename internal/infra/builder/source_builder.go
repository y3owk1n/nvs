// Package builder provides source code building functionality for Neovim.
package builder

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/sirupsen/logrus"
)

const (
	repoURL      = "https://github.com/neovim/neovim.git"
	spinnerSpeed = 100
	commitLen    = 7
	dirPerm      = 0o755
	filePerm     = 0o644
	maxAttempts  = 2
)

// SourceBuilder builds Neovim from source code.
type SourceBuilder struct {
	execCommand ExecCommandFunc
}

// ExecCommandFunc is a function type for executing commands (allows mocking).
type ExecCommandFunc func(ctx context.Context, name string, args ...string) Commander

// Commander is an interface for command execution.
type Commander interface {
	Run() error
	SetDir(dir string)
	SetStdout(stdout interface{})
	SetStderr(stderr interface{})
	StdoutPipe() (interface{}, error)
	StderrPipe() (interface{}, error)
}

// New creates a new SourceBuilder instance.
func New(execFunc ExecCommandFunc) *SourceBuilder {
	if execFunc == nil {
		execFunc = defaultExecCommand
	}

	return &SourceBuilder{
		execCommand: execFunc,
	}
}

// BuildFromCommit builds Neovim from a specific commit or "master".
func (b *SourceBuilder) BuildFromCommit(ctx context.Context, commit string, dest string) error {
	localPath := filepath.Join(os.TempDir(), "neovim-src")
	logrus.Debugf("Temporary Neovim source directory: %s", localPath)

	var err error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = b.buildFromCommitInternal(ctx, commit, dest, localPath)
		if err == nil {
			return nil
		}

		logrus.Errorf("Build attempt %d failed: %v", attempt, err)

		// Clean up for retry
		if removeErr := os.RemoveAll(localPath); removeErr != nil {
			logrus.Warnf("Failed to remove temporary directory: %v", removeErr)
		}

		if attempt < maxAttempts {
			logrus.Info("Retrying build with clean directory...")
			time.Sleep(1 * time.Second)
		}
	}

	return fmt.Errorf("%w after %d attempts: %w", ErrBuildFailed, maxAttempts, err)
}

// buildFromCommitInternal performs the actual build process.
func (b *SourceBuilder) buildFromCommitInternal(ctx context.Context, commit, dest, localPath string) error {
	s := spinner.New(spinner.CharSets[14], spinnerSpeed*time.Millisecond)
	s.Start()
	defer s.Stop()

	// Clone repository if needed
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		s.Suffix = " Cloning repository..."
		logrus.Debug("Cloning repository from ", repoURL)

		cmd := b.execCommand(ctx, "git", "clone", "--quiet", repoURL, localPath)
		cmd.SetStdout(os.Stdout)
		cmd.SetStderr(os.Stderr)

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
	}

	// Checkout commit or master
	if commit == "master" {
		s.Suffix = " Checking out master branch..."
		logrus.Debug("Checking out master branch")

		checkoutCmd := b.execCommand(ctx, "git", "checkout", "--quiet", "master")
		checkoutCmd.SetDir(localPath)

		if err := checkoutCmd.Run(); err != nil {
			return fmt.Errorf("failed to checkout master: %w", err)
		}

		s.Suffix = " Pulling latest changes..."
		logrus.Debug("Pulling latest changes")

		pullCmd := b.execCommand(ctx, "git", "pull", "--quiet", "origin", "master")
		pullCmd.SetDir(localPath)

		if err := pullCmd.Run(); err != nil {
			return fmt.Errorf("failed to pull latest changes: %w", err)
		}
	} else {
		s.Suffix = " Checking out commit " + commit + "..."
		logrus.Debugf("Checking out commit %s", commit)

		checkoutCmd := b.execCommand(ctx, "git", "checkout", "--quiet", commit)
		checkoutCmd.SetDir(localPath)

		if err := checkoutCmd.Run(); err != nil {
			return fmt.Errorf("failed to checkout commit %s: %w", commit, err)
		}
	}

	// Get current commit hash
	cmd := b.execCommand(ctx, "git", "rev-parse", "--quiet", "HEAD")
	cmd.SetDir(localPath)

	var out bytes.Buffer
	cmd.SetStdout(&out)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to get commit hash: %w", err)
	}

	commitHashFull := strings.TrimSpace(out.String())
	if len(commitHashFull) < commitLen {
		return ErrCommitHashTooShort
	}

	commitHash := commitHashFull[:7]
	logrus.Debugf("Current commit hash: %s", commitHash)

	// Clean build directory
	buildPath := filepath.Join(localPath, "build")
	if _, err := os.Stat(buildPath); err == nil {
		logrus.Debug("Removing existing build directory")
		if err := os.RemoveAll(buildPath); err != nil {
			return fmt.Errorf("failed to remove build directory: %w", err)
		}
	}

	// Build Neovim
	s.Suffix = " Building Neovim..."
	logrus.Debug("Building Neovim")

	buildCmd := b.execCommand(ctx, "make", "CMAKE_BUILD_TYPE=Release")
	buildCmd.SetDir(localPath)

	if err := runCommandWithSpinner(ctx, s, buildCmd); err != nil {
		return fmt.Errorf("%w: %w", ErrBuildFailed, err)
	}

	// Create installation directory
	targetDir := filepath.Join(dest, commitHash)
	if err := os.MkdirAll(targetDir, dirPerm); err != nil {
		return fmt.Errorf("failed to create installation directory: %w", err)
	}

	// Install using cmake
	s.Suffix = " Installing Neovim..."
	logrus.Debugf("Installing to %s", targetDir)

	installCmd := b.execCommand(ctx, "cmake", "--install", "build", "--prefix="+targetDir)
	installCmd.SetDir(localPath)

	if err := runCommandWithSpinner(ctx, s, installCmd); err != nil {
		return fmt.Errorf("cmake install failed: %w", err)
	}

	// Verify binary exists
	installedBinary := filepath.Join(targetDir, "bin", "nvim")
	if _, err := os.Stat(installedBinary); os.IsNotExist(err) {
		return fmt.Errorf("%w at %s", ErrBinaryNotFound, installedBinary)
	}

	// Write version file
	versionFile := filepath.Join(targetDir, "version.txt")
	if err := os.WriteFile(versionFile, []byte(commitHashFull), filePerm); err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}

	s.Suffix = " Build complete!"
	logrus.Info("Build and installation successful")

	return nil
}

// runCommandWithSpinner runs a command while updating spinner with output.
func runCommandWithSpinner(ctx context.Context, s *spinner.Spinner, cmd Commander) error {
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// For now, just run the command
	// In a full implementation, we'd read from pipes and update spinner
	_ = stdoutPipe
	_ = stderrPipe

	return cmd.Run()
}

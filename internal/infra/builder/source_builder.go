// Package builder provides source code building functionality for Neovim.
package builder

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/sirupsen/logrus"
)

const (
	bufferSize   = 4096
	spinnerSpeed = 100
	numReaders   = 2
	maxAttempts  = 3
	repoURL      = "https://github.com/neovim/neovim.git"
	commitLen    = 7
	dirPerm      = 0o755
	filePerm     = 0o644
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
	SetStdout(stdout any)
	SetStderr(stderr any)
	StdoutPipe() (any, error)
	StderrPipe() (any, error)
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
		removeErr := os.RemoveAll(localPath)
		if removeErr != nil {
			logrus.Warnf("Failed to remove temporary directory: %v", removeErr)
		}

		if attempt < maxAttempts {
			logrus.Info("Retrying build with clean directory...")
			time.Sleep(1 * time.Second)
		}
	}

	joined := errors.Join(ErrBuildFailed, err)

	return fmt.Errorf("after %d attempts: %w", maxAttempts, joined)
}

// buildFromCommitInternal performs the actual build process.
func (b *SourceBuilder) buildFromCommitInternal(
	ctx context.Context,
	commit, dest, localPath string,
) error {
	var err error

	buildSpinner := spinner.New(spinner.CharSets[14], spinnerSpeed*time.Millisecond)

	buildSpinner.Start()
	defer buildSpinner.Stop()

	// Clone repository if needed
	gitDir := filepath.Join(localPath, ".git")

	_, err = os.Stat(gitDir)
	if os.IsNotExist(err) {
		// Clean up partial clone if exists
		_ = os.RemoveAll(localPath)

		buildSpinner.Suffix = " Cloning repository..."

		logrus.Debug("Cloning repository from ", repoURL)

		cmd := b.execCommand(ctx, "git", "clone", "--quiet", repoURL, localPath)
		cmd.SetStdout(os.Stdout)
		cmd.SetStderr(os.Stderr)

		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
	}

	// Checkout commit or master
	if commit == "master" {
		buildSpinner.Suffix = " Checking out master branch..."

		logrus.Debug("Checking out master branch")

		checkoutCmd := b.execCommand(ctx, "git", "checkout", "--quiet", "master")
		checkoutCmd.SetDir(localPath)

		err = checkoutCmd.Run()
		if err != nil {
			return fmt.Errorf("failed to checkout master: %w", err)
		}
	} else {
		buildSpinner.Suffix = " Checking out commit..."

		logrus.Debugf("Checking out commit %s", commit)

		checkoutCmd := b.execCommand(ctx, "git", "checkout", "--quiet", commit)
		checkoutCmd.SetDir(localPath)

		err = checkoutCmd.Run()
		if err != nil {
			return fmt.Errorf("failed to checkout commit %s: %w", commit, err)
		}
	}

	// Get current commit hash
	cmd := b.execCommand(ctx, "git", "rev-parse", "--quiet", "HEAD")
	cmd.SetDir(localPath)

	var out bytes.Buffer
	cmd.SetStdout(&out)

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to get commit hash: %w", err)
	}

	commitHashFull := strings.TrimSpace(out.String())
	if len(commitHashFull) < commitLen {
		return ErrCommitHashTooShort
	}

	commitHash := commitHashFull[:commitLen]
	logrus.Debugf("Current commit hash: %s", commitHash)

	// Clean build directory
	buildPath := filepath.Join(localPath, "build")

	_, err = os.Stat(buildPath)
	if err == nil {
		logrus.Debug("Removing existing build directory")

		err = os.RemoveAll(buildPath)
		if err != nil {
			return fmt.Errorf("failed to remove build directory: %w", err)
		}
	}

	// Build Neovim
	buildSpinner.Suffix = " Building Neovim..."

	logrus.Debug("Building Neovim")

	buildCmd := b.execCommand(ctx, "make", "CMAKE_BUILD_TYPE=Release")
	buildCmd.SetDir(localPath)

	err = runCommandWithSpinner(buildCmd)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrBuildFailed, err)
	}

	// Create installation directory
	targetDir := filepath.Join(dest, commitHash)

	err = os.MkdirAll(targetDir, dirPerm)
	if err != nil {
		return fmt.Errorf("failed to create installation directory: %w", err)
	}

	// Install using cmake
	buildSpinner.Suffix = " Installing Neovim..."

	logrus.Debugf("Installing to %s", targetDir)

	installCmd := b.execCommand(ctx, "cmake", "--install", "build", "--prefix="+targetDir)
	installCmd.SetDir(localPath)

	err = runCommandWithSpinner(installCmd)
	if err != nil {
		return fmt.Errorf("cmake install failed: %w", err)
	}

	// Verify binary exists
	installedBinary := filepath.Join(targetDir, "bin", "nvim")

	_, err = os.Stat(installedBinary)
	if os.IsNotExist(err) {
		return fmt.Errorf("%w at %s", ErrBinaryNotFound, installedBinary)
	}

	// Write version file
	versionFile := filepath.Join(targetDir, "version.txt")

	err = os.WriteFile(versionFile, []byte(commitHashFull), filePerm)
	if err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}

	buildSpinner.Suffix = " Build complete!"

	logrus.Info("Build and installation successful")

	return nil
}

// runCommandWithSpinner runs a command while updating spinner with output.
func runCommandWithSpinner(cmd Commander) error {
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Cast pipes to io.Reader for reading
	stdoutReader, stdoutOk := stdoutPipe.(io.Reader)
	if !stdoutOk {
		return ErrStdoutPipeNotReader
	}

	stderrReader, stderrOk := stderrPipe.(io.Reader)
	if !stderrOk {
		return ErrStderrPipeNotReader
	}

	// Run command and capture output concurrently
	errChan := make(chan error, 1)

	go func() {
		errChan <- cmd.Run()
	}()

	// Read from both pipes concurrently
	var waitGroup sync.WaitGroup
	waitGroup.Add(numReaders)

	go func() {
		defer waitGroup.Done()

		buf := make([]byte, bufferSize)
		for {
			n, err := stdoutReader.Read(buf)
			if n > 0 {
				line := strings.TrimSpace(string(buf[:n]))
				if line != "" {
					logrus.Debugf("Build output: %s", line)
				}
			}

			if err != nil {
				break
			}
		}
	}()

	go func() {
		defer waitGroup.Done()

		buf := make([]byte, bufferSize)
		for {
			n, err := stderrReader.Read(buf)
			if n > 0 {
				line := strings.TrimSpace(string(buf[:n]))
				if line != "" {
					logrus.Debugf("Build error: %s", line)
				}
			}

			if err != nil {
				break
			}
		}
	}()

	// Wait for command to complete
	err = <-errChan

	waitGroup.Wait()

	return err
}

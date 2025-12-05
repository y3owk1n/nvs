// Package builder provides source code building functionality for Neovim.
package builder

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/internal/domain/installer"
)

const (
	bufferSize     = 4096
	spinnerSpeed   = 100
	numReaders     = 2
	maxAttempts    = 3
	repoURL        = "https://github.com/neovim/neovim.git"
	commitLen      = 7
	dirPerm        = 0o755
	filePerm       = 0o644
	progressStart  = 0
	progressLow    = 10
	progressMid    = 20
	progressHigh   = 80
	progressDone   = 100
	tickerInterval = 10
	outputChanSize = 10
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
func (b *SourceBuilder) BuildFromCommit(
	ctx context.Context,
	commit string,
	dest string,
	progress installer.ProgressFunc,
) (string, error) {
	// Check for required build tools
	err := b.checkRequiredTools()
	if err != nil {
		return "", fmt.Errorf("build requirements not met: %w", err)
	}

	// Clean up any leftover temp directories from previous runs
	b.cleanupTempDirectories()

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		localPath := filepath.Join(os.TempDir(), fmt.Sprintf("neovim-src-%d", attempt))
		logrus.Debugf("Temporary Neovim source directory: %s", localPath)

		resolvedHash, err := b.buildFromCommitInternal(ctx, commit, dest, localPath, progress)
		if err == nil {
			return resolvedHash, nil
		}

		logrus.Errorf("Build attempt %d failed: %v", attempt, err)

		// Don't retry if build requirements are not met
		if errors.Is(err, ErrBuildRequirementsNotMet) {
			break
		}

		// Clean up for retry (though not strictly necessary with unique paths)
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

	return "", fmt.Errorf("after %d attempts: %w", maxAttempts, joined)
}

// buildFromCommitInternal performs the actual build process.
func (b *SourceBuilder) buildFromCommitInternal(
	ctx context.Context,
	commit, dest, localPath string,
	progress installer.ProgressFunc,
) (string, error) {
	var err error

	// Clone repository if needed
	gitDir := filepath.Join(localPath, ".git")

	_, err = os.Stat(gitDir)
	if os.IsNotExist(err) {
		// Ensure clean directory for clone
		removeErr := os.RemoveAll(localPath)
		if removeErr != nil {
			logrus.Warnf("Failed to remove temp directory: %v", removeErr)
		}

		if progress != nil {
			progress("Cloning repository (large repo, may take a while)", -1)
		}

		// Ensure the target directory doesn't exist (git clone expects to create it)
		err := os.RemoveAll(localPath)
		if err != nil {
			logrus.Warnf("Failed to clean target directory: %v", err)
		}

		logrus.Debug("Cloning repository from ", repoURL)

		cmd := b.execCommand(ctx, "git", "clone", "--quiet", repoURL, localPath)
		cmd.SetStdout(os.Stdout)
		cmd.SetStderr(os.Stderr)

		err = cmd.Run()
		if err != nil {
			return "", fmt.Errorf("failed to clone repository: %w", err)
		}
	}

	// Checkout commit or master
	if commit == "master" {
		if progress != nil {
			progress("Checking out master branch", -1)
		}

		logrus.Debug("Checking out master branch")

		checkoutCmd := b.execCommand(ctx, "git", "checkout", "--quiet", "master")
		checkoutCmd.SetDir(localPath)

		err = checkoutCmd.Run()
		if err != nil {
			return "", fmt.Errorf("failed to checkout master: %w", err)
		}
	} else {
		if progress != nil {
			progress("Checking out commit", -1)
		}

		logrus.Debugf("Checking out commit %s", commit)

		checkoutCmd := b.execCommand(ctx, "git", "checkout", "--quiet", commit)
		checkoutCmd.SetDir(localPath)

		err = checkoutCmd.Run()
		if err != nil {
			return "", fmt.Errorf("failed to checkout commit %s: %w", commit, err)
		}
	}

	// Get current commit hash
	cmd := b.execCommand(ctx, "git", "rev-parse", "--quiet", "HEAD")
	cmd.SetDir(localPath)

	var out bytes.Buffer
	cmd.SetStdout(&out)

	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", err)
	}

	commitHashFull := strings.TrimSpace(out.String())
	if len(commitHashFull) < commitLen {
		return "", ErrCommitHashTooShort
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
			return "", fmt.Errorf("failed to remove build directory: %w", err)
		}
	}

	// Build Neovim
	logrus.Debug("Building Neovim")

	buildCmd := b.execCommand(ctx, "make", "CMAKE_BUILD_TYPE=Release")
	buildCmd.SetDir(localPath)

	err = runCommandWithProgress(buildCmd, progress, "Building Neovim")
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrBuildFailed, err)
	}

	// Create installation directory
	targetDir := filepath.Join(dest, commitHash)

	err = os.MkdirAll(targetDir, dirPerm)
	if err != nil {
		return "", fmt.Errorf("failed to create installation directory: %w", err)
	}

	// Install using cmake
	logrus.Debugf("Installing to %s", targetDir)

	installCmd := b.execCommand(ctx, "cmake", "--install", "build", "--prefix="+targetDir)
	installCmd.SetDir(localPath)

	err = runCommandWithProgress(installCmd, progress, "Installing Neovim")
	if err != nil {
		return "", fmt.Errorf("cmake install failed: %w", err)
	}

	// Verify binary exists
	installedBinary := filepath.Join(targetDir, "bin", "nvim")

	_, err = os.Stat(installedBinary)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("%w at %s", ErrBinaryNotFound, installedBinary)
	}

	// Write version file
	versionFile := filepath.Join(targetDir, "version.txt")

	err = os.WriteFile(versionFile, []byte(commitHashFull), filePerm)
	if err != nil {
		return "", fmt.Errorf("failed to write version file: %w", err)
	}

	if progress != nil {
		progress("Build complete", progressDone)
	}

	logrus.Info("Build and installation successful")

	return commitHash, nil
}

// checkRequiredTools verifies that all required build tools are available.
func (b *SourceBuilder) checkRequiredTools() error {
	requiredTools := []string{"git", "make", "cmake", "gettext", "ninja", "curl"}

	for _, tool := range requiredTools {
		_, err := exec.LookPath(tool)
		if err != nil {
			return fmt.Errorf(
				"%w: %s is not installed or not in PATH",
				ErrBuildRequirementsNotMet,
				tool,
			)
		}
	}

	return nil
}

// cleanupTempDirectories removes any leftover neovim-src-* directories from previous runs.
func (b *SourceBuilder) cleanupTempDirectories() {
	tempDir := os.TempDir()

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		logrus.Warnf("Failed to read temp directory for cleanup: %v", err)

		return
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "neovim-src-") && entry.IsDir() {
			dirPath := filepath.Join(tempDir, entry.Name())

			err := os.RemoveAll(dirPath)
			if err != nil {
				logrus.Warnf("Failed to remove leftover temp directory %s: %v", dirPath, err)
			} else {
				logrus.Debugf("Cleaned up leftover temp directory: %s", dirPath)
			}
		}
	}
}

// runCommandWithProgress runs a command while updating progress with elapsed time.
func runCommandWithProgress(cmd Commander, progress installer.ProgressFunc, phase string) error {
	if progress == nil {
		return runCommandWithSpinner(cmd)
	}

	startTime := time.Now()

	ticker := time.NewTicker(tickerInterval * time.Second)
	defer ticker.Stop()

	// Start progress
	progress(phase, -1)

	// Channel to signal completion
	done := make(chan error, 1)

	// Channel for important output lines
	outputChan := make(chan string, outputChanSize)

	var lastMessage string

	go func() {
		done <- runCommandWithSpinnerAndOutput(cmd, func(line string) {
			// Show important cmake messages and error messages
			isImportant := strings.HasPrefix(line, "-- ") &&
				!strings.HasPrefix(line, "-- Looking for") &&
				!strings.HasPrefix(line, "-- Performing Test")
			lowerLine := strings.ToLower(line)
			isError := strings.Contains(lowerLine, "error") || strings.Contains(lowerLine, "failed")

			if isImportant || isError {
				select {
				case outputChan <- line:
				default:
					// Channel full, skip
				}
			}
		})
	}()

	for {
		select {
		case err := <-done:
			// Update final progress
			elapsed := time.Since(startTime)
			if lastMessage != "" {
				progress(
					fmt.Sprintf(
						"%s: %s (completed in %v)",
						phase,
						lastMessage,
						elapsed.Round(time.Second),
					),
					-1,
				)
			} else {
				progress(fmt.Sprintf("%s (completed in %v)", phase, elapsed.Round(time.Second)), -1)
			}

			return err
		case output := <-outputChan:
			// Update progress with latest message
			lastMessage = strings.TrimPrefix(output, "-- ")
			elapsed := time.Since(startTime)
			progress(
				fmt.Sprintf("%s: %s (elapsed: %v)", phase, lastMessage, elapsed.Round(time.Second)),
				-1,
			)
		case <-ticker.C:
			// Update progress with elapsed time
			elapsed := time.Since(startTime)
			if lastMessage != "" {
				progress(
					fmt.Sprintf(
						"%s: %s (elapsed: %v)",
						phase,
						lastMessage,
						elapsed.Round(time.Second),
					),
					-1,
				)
			} else {
				progress(fmt.Sprintf("%s (elapsed: %v)", phase, elapsed.Round(time.Second)), -1)
			}
		}
	}
}

// runCommandWithSpinner runs a command while updating spinner with output.
func runCommandWithSpinner(cmd Commander) error {
	return runCommandWithSpinnerAndOutput(cmd, nil)
}

// runCommandWithSpinnerAndOutput runs a command while updating spinner with output.
func runCommandWithSpinnerAndOutput(cmd Commander, outputCallback func(string)) error {
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

					if outputCallback != nil {
						outputCallback(line)
					}
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

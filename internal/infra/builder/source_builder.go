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
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/domain/installer"
)

const toolCheckTimeout = 30 * time.Second

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
	// Clean up any leftover temp directories from previous runs
	b.cleanupTempDirectories()

	// Generate unique build ID to avoid conflicts with concurrent builds
	buildID := fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	logrus.Debugf("Build ID: %s", buildID)

	// Create lock file to prevent cleanup of in-progress builds
	lockFile := filepath.Join(os.TempDir(), fmt.Sprintf("neovim-src-%s.lock", buildID))

	lockErr := os.WriteFile(lockFile, []byte(strconv.Itoa(os.Getpid())), constants.FilePerm)
	if lockErr != nil {
		logrus.Warnf("Failed to create lock file: %v", lockErr)
	}
	// Ensure lock file is removed when function exits
	defer func() {
		removeErr := os.Remove(lockFile)
		if removeErr != nil && !os.IsNotExist(removeErr) {
			logrus.Warnf("Failed to remove lock file: %v", removeErr)
		}
	}()

	var (
		resolvedHash string
		err          error
	)
	for attempt := 1; attempt <= constants.MaxAttempts; attempt++ {
		// Check for required build tools on each attempt
		err = b.checkRequiredTools(ctx)
		if err != nil {
			// Don't retry if build requirements are not met
			if errors.Is(err, ErrBuildRequirementsNotMet) {
				return "", fmt.Errorf("build requirements not met: %w", err)
			}
			// Return immediately on context cancellation/timeout
			return "", fmt.Errorf("build requirements check failed: %w", err)
		}

		localPath := filepath.Join(os.TempDir(), fmt.Sprintf("neovim-src-%s-%d", buildID, attempt))
		logrus.Debugf("Temporary Neovim source directory: %s", localPath)

		resolvedHash, err = b.buildFromCommitInternal(ctx, commit, dest, localPath, progress)
		if err == nil {
			// Clean up temp directory on successful build
			removeErr := os.RemoveAll(localPath)
			if removeErr != nil {
				logrus.Warnf("Failed to remove temporary directory: %v", removeErr)
			}

			return resolvedHash, nil
		}

		logrus.Errorf("Build attempt %d failed: %v", attempt, err)

		// Check for context cancellation - return immediately without retrying
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return "", err
		}

		// Clean up for retry (though not strictly necessary with unique paths)
		removeErr := os.RemoveAll(localPath)
		if removeErr != nil {
			logrus.Warnf("Failed to remove temporary directory: %v", removeErr)
		}

		if attempt < constants.MaxAttempts {
			logrus.Info("Retrying build with clean directory...")

			select {
			case <-time.After(1 * time.Second):
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
	}

	joined := errors.Join(ErrBuildFailed, err)

	return "", fmt.Errorf("after %d attempts: %w", constants.MaxAttempts, joined)
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

		logrus.Debug("Cloning repository from ", constants.RepoURL)

		cmd := b.execCommand(ctx, "git", "clone", "--quiet", constants.RepoURL, localPath)
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
	if len(commitHashFull) < constants.ShortCommitLen {
		return "", ErrCommitHashTooShort
	}

	commitHash := commitHashFull[:constants.ShortCommitLen]
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

	err = runCommandWithProgress(ctx, buildCmd, progress, "Building Neovim")
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrBuildFailed, err)
	}

	// Create installation directory
	targetDir := filepath.Join(dest, commitHash)

	err = os.MkdirAll(targetDir, constants.DirPerm)
	if err != nil {
		return "", fmt.Errorf("failed to create installation directory: %w", err)
	}

	// Install using cmake
	logrus.Debugf("Installing to %s", targetDir)

	installCmd := b.execCommand(ctx, "cmake", "--install", "build", "--prefix="+targetDir)
	installCmd.SetDir(localPath)

	err = runCommandWithProgress(ctx, installCmd, progress, "Installing Neovim")
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

	err = os.WriteFile(versionFile, []byte(commitHashFull), constants.FilePerm)
	if err != nil {
		return "", fmt.Errorf("failed to write version file: %w", err)
	}

	if progress != nil {
		progress("Build complete", constants.ProgressDone)
	}

	logrus.Info("Build and installation successful")

	return commitHash, nil
}

// checkRequiredTools verifies that all required build tools are available.
func (b *SourceBuilder) checkRequiredTools(ctx context.Context) error {
	requiredTools := []string{"git", "make", "cmake", "gettext", "ninja", "curl"}

	checkCmd := "which"
	if runtime.GOOS == constants.WindowsOS {
		checkCmd = "where"
	}

	checkCtx, cancel := context.WithTimeout(ctx, toolCheckTimeout)
	defer cancel()

	for _, tool := range requiredTools {
		cmd := b.execCommand(checkCtx, checkCmd, tool)

		err := cmd.Run()
		if err != nil {
			if checkCtx.Err() != nil {
				return fmt.Errorf("tool check timed out or was canceled: %w", checkCtx.Err())
			}

			return fmt.Errorf(
				"%w: %s is not installed or not in PATH: %w",
				ErrBuildRequirementsNotMet,
				tool,
				err,
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

			// Check for lock file to skip directories from active builds
			// Format: neovim-src-{buildID}.lock where buildID = {pid}-{timestamp}
			// Directory format: neovim-src-{pid}-{timestamp}-{attempt}
			// Need to extract buildID by removing the attempt suffix
			parts := strings.Split(entry.Name(), "-")

			var lockFileName string
			if len(parts) >= constants.TempDirNamePartsMin {
				buildIDParts := parts[:len(parts)-1]
				lockFileName = strings.Join(buildIDParts, "-") + ".lock"
			} else {
				lockFileName = entry.Name() + ".lock"
			}

			lockFilePath := filepath.Join(tempDir, lockFileName)

			_, err = os.Stat(lockFilePath)
			if err == nil {
				logrus.Debugf("Skipping cleanup of locked directory: %s", dirPath)

				continue
			}

			// Check if the directory was modified recently (within last 5 minutes)
			// to avoid interfering with concurrent builds
			info, err := entry.Info()
			if err != nil {
				logrus.Warnf("Failed to get info for temp directory %s: %v", dirPath, err)

				continue
			}

			if time.Since(info.ModTime()) < 5*time.Minute {
				logrus.Debugf("Skipping cleanup of recent temp directory: %s", dirPath)

				continue
			}

			err = os.RemoveAll(dirPath)
			if err != nil {
				logrus.Warnf("Failed to remove leftover temp directory %s: %v", dirPath, err)
			} else {
				logrus.Debugf("Cleaned up leftover temp directory: %s", dirPath)
			}
		}
	}

	// Clean up orphaned lock files
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "neovim-src-") &&
			strings.HasSuffix(entry.Name(), ".lock") &&
			!entry.IsDir() {
			lockFilePath := filepath.Join(tempDir, entry.Name())

			info, err := entry.Info()
			if err != nil {
				logrus.Warnf("Failed to get info for lock file %s: %v", lockFilePath, err)

				continue
			}

			if time.Since(info.ModTime()) < 5*time.Minute {
				logrus.Debugf("Skipping cleanup of recent lock file: %s", lockFilePath)

				continue
			}

			// Read PID from lock file to check if process is still running
			pidData, readErr := os.ReadFile(lockFilePath)
			if readErr == nil {
				var pid int

				_, err = fmt.Sscanf(string(pidData), "%d", &pid)
				if err == nil {
					if isProcessAlive(pid) {
						logrus.Debugf(
							"Skipping cleanup of lock file for running process %d: %s",
							pid,
							lockFilePath,
						)

						continue
					}
				}
			}

			err = os.Remove(lockFilePath)
			if err != nil {
				logrus.Warnf("Failed to remove orphaned lock file %s: %v", lockFilePath, err)
			} else {
				logrus.Debugf("Cleaned up orphaned lock file: %s", lockFilePath)
			}
		}
	}
}

// runCommandWithProgress runs a command while updating progress with elapsed time.
func runCommandWithProgress(
	ctx context.Context,
	cmd Commander,
	progress installer.ProgressFunc,
	phase string,
) error {
	if progress == nil {
		return runCommandWithSpinner(ctx, cmd)
	}

	startTime := time.Now()

	ticker := time.NewTicker(constants.TickerInterval * time.Second)
	defer ticker.Stop()

	// Start progress
	progress(phase, -1)

	// Channel to signal completion
	done := make(chan error, 1)

	// Channel for important output lines
	outputChan := make(chan string, constants.OutputChanSize)

	var lastMessage string

	go func() {
		done <- runCommandWithSpinnerAndOutput(ctx, cmd, func(line string) {
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
		case <-ctx.Done():
			// Check if command completed before context was canceled
			select {
			case doneErr := <-done:
				// Command finished - if successful, return success despite cancellation
				if doneErr == nil {
					elapsed := time.Since(startTime)
					progress(
						fmt.Sprintf("%s (completed in %v)", phase, elapsed.Round(time.Second)),
						-1,
					)

					return nil
				}
				// Command failed - return the error
				return doneErr
			default:
				// No result yet - context was canceled before command completed
				return ctx.Err()
			}
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
func runCommandWithSpinner(ctx context.Context, cmd Commander) error {
	return runCommandWithSpinnerAndOutput(ctx, cmd, nil)
}

// runCommandWithSpinnerAndOutput runs a command while updating spinner with output.
func runCommandWithSpinnerAndOutput(
	ctx context.Context,
	cmd Commander,
	outputCallback func(string),
) error {
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

	// Wait for all goroutines to complete
	var waitGroup sync.WaitGroup
	waitGroup.Add(constants.NumReaders + 1)

	go func() {
		defer waitGroup.Done()

		errChan <- cmd.Run()
	}()

	// Read from both pipes concurrently
	go func() {
		defer waitGroup.Done()

		buf := make([]byte, constants.BufferSize)
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

		buf := make([]byte, constants.BufferSize)
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
	select {
	case err = <-errChan:
	case <-ctx.Done():
		// Context canceled - check if command completed before returning
		select {
		case doneErr := <-errChan:
			// Command finished - return its result
			waitGroup.Wait()

			return doneErr
		default:
			// No result yet - context was canceled before command completed
			waitGroup.Wait()

			return ctx.Err()
		}
	}

	waitGroup.Wait()

	return err
}

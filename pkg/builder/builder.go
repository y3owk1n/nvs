// Package builder provides functions for building Neovim from source.
package builder

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/pkg/helpers"
)

// Constants for builder operations.
const (
	SpinnerSpeed = 100
	CommitLen    = 7
	DirPerm      = 0o755
	FilePerm     = 0o644
)

// Errors for builder operations.
var (
	ErrCommitHashTooShort      = errors.New("commit hash too short")
	ErrInstalledBinaryNotFound = errors.New("installed binary not found")
)

const repoURL = "https://github.com/neovim/neovim.git"

// ExecCommandFunc is a variable to allow overriding the exec.CommandContext function in tests.
var ExecCommandFunc = exec.CommandContext

// buildFromCommitInternal clones the Neovim repository (if not already present),
// checks out the specified commit (or master branch), builds Neovim, and installs it into the provided versionsDir.
// It returns an error if any of these steps fail.
//
// Example usage:
//
//	ctx := context.Background()
//	commit := "master" // or a specific commit hash
//	versionsDir := "/path/to/installations"
//	localPath := "/tmp/neovim-src" // temporary source directory
//	if err := buildFromCommitInternal(ctx, commit, versionsDir, localPath); err != nil {
//	    // handle error
//	}
func buildFromCommitInternal(ctx context.Context, commit, versionsDir, localPath string) error {
	spinner := spinner.New(spinner.CharSets[14], SpinnerSpeed*time.Millisecond)

	spinner.Start()
	defer spinner.Stop()

	var err error

	// Clone repository if localPath doesn't exist.
	_, statErr := os.Stat(localPath)
	if os.IsNotExist(statErr) {
		spinner.Suffix = " Cloning repository..."

		logrus.Debug("Cloning repository from ", repoURL)
		cloneCmd := ExecCommandFunc(ctx, "git", "clone", "--quiet", repoURL, localPath)
		cloneCmd.Stdout = os.Stdout

		cloneCmd.Stderr = os.Stderr

		err = cloneCmd.Run()
		if err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
	}

	// Checkout the appropriate commit or master branch.
	if commit == "master" {
		spinner.Suffix = " Checking out master branch..."

		logrus.Debug("Checking out master branch")

		checkoutCmd := ExecCommandFunc(ctx, "git", "checkout", "--quiet", "master")

		checkoutCmd.Dir = localPath

		err := checkoutCmd.Run()
		if err != nil {
			return fmt.Errorf("failed to checkout master branch: %w", err)
		}

		spinner.Suffix = " Pulling latest changespinner..."

		logrus.Debug("Pulling latest changes on master branch")

		pullCmd := ExecCommandFunc(ctx, "git", "pull", "--quiet", "origin", "master")

		pullCmd.Dir = localPath

		err = pullCmd.Run()
		if err != nil {
			return fmt.Errorf("failed to pull latest changes: %w", err)
		}
	} else {
		spinner.Suffix = " Checking out commit " + commit + "..."
		logrus.Debug("Checking out commit ", commit)
		checkoutCmd := ExecCommandFunc(ctx, "git", "checkout", "--quiet", commit)

		checkoutCmd.Dir = localPath

		err = checkoutCmd.Run()
		if err != nil {
			return fmt.Errorf("failed to checkout commit %s: %w", commit, err)
		}
	}

	// Retrieve the current commit hash.
	cmd := ExecCommandFunc(ctx, "git", "rev-parse", "--quiet", "HEAD")
	cmd.Dir = localPath

	var out bytes.Buffer

	cmd.Stdout = &out

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to get current commit hash: %w", err)
	}

	commitHashFull := strings.TrimSpace(out.String())
	if len(commitHashFull) < CommitLen {
		return ErrCommitHashTooShort
	}

	commitHash := commitHashFull[:7]
	logrus.Debug("Current commit hash: ", commitHash)

	// Clear the build directory if it exists.
	depsPath := filepath.Join(localPath, "build")

	_, err = os.Stat(depsPath)
	if err == nil {
		logrus.Debug("Removing existing build directory...")

		err := os.RemoveAll(depsPath)
		if err != nil {
			return fmt.Errorf("failed to remove build directory: %w", err)
		}
	}

	// Build Neovim.
	spinner.Suffix = " Building Neovim..."

	logrus.Debug("Building Neovim at: ", localPath)

	buildCmd := ExecCommandFunc(ctx, "make", "CMAKE_BUILD_TYPE=Release")
	buildCmd.Dir = localPath

	err = helpers.RunCommandWithSpinner(ctx, spinner, buildCmd)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Create installation target directory.
	targetDir := filepath.Join(versionsDir, commitHash)

	err = os.MkdirAll(targetDir, DirPerm)
	if err != nil {
		return fmt.Errorf("failed to create installation directory: %w", err)
	}

	// Install runtime files using cmake.
	spinner.Suffix = " Installing Neovim..."

	logrus.Debug("Running cmake install with PREFIX=", targetDir)
	installCmd := ExecCommandFunc(ctx, "cmake", "--install", "build", "--prefix="+targetDir)
	installCmd.Dir = localPath

	err = helpers.RunCommandWithSpinner(ctx, spinner, installCmd)
	if err != nil {
		return fmt.Errorf("cmake install failed: %w", err)
	}

	// Verify that the installed binary exists.
	installedBinaryPath := filepath.Join(targetDir, "bin", "nvim")

	_, err = os.Stat(installedBinaryPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("%w at %s", ErrInstalledBinaryNotFound, installedBinaryPath)
	}

	// Write the full commit hash to a version file.
	versionFile := filepath.Join(targetDir, "version.txt")

	err = os.WriteFile(versionFile, []byte(commitHashFull), FilePerm)
	if err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}

	spinner.Suffix = " Build and installation complete!"

	logrus.Debug("Build and installation successful")

	_, err = fmt.Fprintf(os.Stdout,
		"\n%s %s\n",
		helpers.SuccessIcon(),
		helpers.CyanText("Build and installation successful!"),
	)
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	return nil
}

// BuildFromCommit is the public function that builds Neovim from a specified commit or branch.
// It creates a temporary directory for the Neovim source, attempts the build process (with retries),
// and returns an error if the build fails after the maximum number of attempts.
//
// Example usage:
//
//	ctx := context.Background()
//	commit := "master" // or a specific commit hash
//	versionsDir := "/path/to/installations"
//	if err := BuildFromCommit(ctx, commit, versionsDir); err != nil {
//	    // handle build failure
//	}
func BuildFromCommit(ctx context.Context, commit, versionsDir string) error {
	localPath := filepath.Join(os.TempDir(), "neovim-src")
	logrus.Debug("Temporary Neovim Src directory: ", localPath)

	var err error

	const maxAttempts = 2

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = buildFromCommitInternal(ctx, commit, versionsDir, localPath)
		if err == nil {
			return nil
		}

		logrus.Error("Error building from commit: ", err)
		logrus.Debugf("Attempt %d failed: %v", attempt, err)

		removeErr := os.RemoveAll(localPath)
		if removeErr != nil {
			logrus.Errorf(
				"Failed to remove temporary directory %s: %v",
				localPath,
				removeErr,
			)
		}

		if attempt < maxAttempts {
			logrus.Errorf(
				"Retrying build process with clean directory (attempt %d)...",
				attempt+1,
			)
			time.Sleep(1 * time.Second)
		}
	}

	return fmt.Errorf("build failed after %d attempts: %w", maxAttempts, err)
}

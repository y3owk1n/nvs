package builder

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/pkg/utils"
)

const repoURL = "https://github.com/neovim/neovim.git"

var execCommandFunc = exec.CommandContext

func buildFromCommitInternal(ctx context.Context, commit, versionsDir, localPath string) error {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Start()
	defer s.Stop()

	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		s.Suffix = " Cloning repository..."
		logrus.Debug("Cloning repository from ", repoURL)
		cloneCmd := execCommandFunc(ctx, "git", "clone", "--quiet", repoURL, localPath)
		cloneCmd.Stdout = os.Stdout
		cloneCmd.Stderr = os.Stderr
		if err := cloneCmd.Run(); err != nil {
			return fmt.Errorf("failed to clone repository: %v", err)
		}
	}

	if commit == "master" {
		s.Suffix = " Checking out master branch..."
		logrus.Debug("Checking out master branch")
		checkoutCmd := execCommandFunc(ctx, "git", "checkout", "--quiet", "master")
		checkoutCmd.Dir = localPath
		if err := checkoutCmd.Run(); err != nil {
			return fmt.Errorf("failed to checkout master branch: %v", err)
		}

		s.Suffix = " Pulling latest changes..."
		logrus.Debug("Pulling latest changes on master branch")
		pullCmd := execCommandFunc(ctx, "git", "pull", "--quiet", "origin", "master")
		pullCmd.Dir = localPath
		if err := pullCmd.Run(); err != nil {
			return fmt.Errorf("failed to pull latest changes: %v", err)
		}
	} else {
		s.Suffix = " Checking out commit " + commit + "..."
		logrus.Debug("Checking out commit ", commit)
		checkoutCmd := execCommandFunc(ctx, "git", "checkout", "--quiet", commit)
		checkoutCmd.Dir = localPath
		if err := checkoutCmd.Run(); err != nil {
			return fmt.Errorf("failed to checkout commit %s: %v", commit, err)
		}
	}

	cmd := execCommandFunc(ctx, "git", "rev-parse", "--quiet", "HEAD")
	cmd.Dir = localPath
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to get current commit hash: %v", err)
	}
	commitHashFull := strings.TrimSpace(out.String())
	if len(commitHashFull) < 7 {
		return fmt.Errorf("commit hash too short")
	}
	commitHash := commitHashFull[:7]
	logrus.Debug("Current commit hash: ", commitHash)

	// clear the build directory first
	depsPath := filepath.Join(localPath, "build")
	if _, err := os.Stat(depsPath); err == nil {
		logrus.Debug("Removing existing build directory...")
		if err := os.RemoveAll(depsPath); err != nil {
			return fmt.Errorf("failed to remove build directory: %v", err)
		}
	}

	// Build Neovim
	s.Suffix = " Building Neovim..."
	logrus.Debug("Building Neovim at: ", localPath)
	buildCmd := execCommandFunc(ctx, "make", "CMAKE_BUILD_TYPE=Release")
	buildCmd.Dir = localPath

	if err := utils.RunCommandWithSpinner(ctx, s, buildCmd); err != nil {
		return fmt.Errorf("build failed: %v", err)
	}

	targetDir := filepath.Join(versionsDir, commitHash)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create installation directory: %v", err)
	}

	// Install runtime files
	s.Suffix = " Installing Neovim..."
	logrus.Debug("Running cmake install with PREFIX=", targetDir)
	installCmd := execCommandFunc(ctx, "cmake", "--install", "build", "--prefix="+targetDir)
	installCmd.Dir = localPath

	if err := utils.RunCommandWithSpinner(ctx, s, installCmd); err != nil {
		return fmt.Errorf("cmake install failed: %v", err)
	}

	installedBinaryPath := filepath.Join(targetDir, "bin", "nvim")
	if _, err := os.Stat(installedBinaryPath); os.IsNotExist(err) {
		return fmt.Errorf("installed binary not found at %s", installedBinaryPath)
	}

	versionFile := filepath.Join(targetDir, "version.txt")
	if err := os.WriteFile(versionFile, []byte(commitHashFull), 0644); err != nil {
		return fmt.Errorf("failed to write version file: %v", err)
	}

	s.Suffix = " Build and installation complete!"
	logrus.Debug("Build and installation successful")
	fmt.Printf("\n%s %s\n", utils.SuccessIcon(), utils.CyanText("Build and installation successful!"))
	return nil
}

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
		if removeErr := os.RemoveAll(localPath); removeErr != nil {
			logrus.Errorf("Failed to remove temporary directory %s: %v", localPath, removeErr)
		}
		if attempt < maxAttempts {
			logrus.Errorf("Retrying build process with clean directory (attempt %d)...", attempt+1)
			time.Sleep(1 * time.Second)
		}
	}
	return fmt.Errorf("build failed after %d attempts: %v", maxAttempts, err)
}

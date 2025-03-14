package builder

import (
	"bytes"
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

var (
	execCommandFunc = exec.Command
	copyFileFunc    = utils.CopyFile
)

func BuildFromCommit(commit, versionsDir string) error {
	localPath := filepath.Join(os.TempDir(), "neovim-src")

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Start()
	defer s.Stop()

	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		s.Suffix = " Cloning repository..."
		logrus.Debug("Cloning repository from ", repoURL)
		cloneCmd := execCommandFunc("git", "clone", repoURL, localPath)
		cloneCmd.Stdout = os.Stdout
		cloneCmd.Stderr = os.Stderr
		if err := cloneCmd.Run(); err != nil {
			return fmt.Errorf("failed to clone repository: %v", err)
		}
	}

	if commit == "master" {
		s.Suffix = " Checking out master branch..."
		logrus.Debug("Checking out master branch")
		checkoutCmd := execCommandFunc("git", "checkout", "master")
		checkoutCmd.Dir = localPath
		checkoutCmd.Stdout = os.Stdout
		checkoutCmd.Stderr = os.Stderr
		if err := checkoutCmd.Run(); err != nil {
			return fmt.Errorf("failed to checkout master branch: %v", err)
		}

		s.Suffix = " Pulling latest changes..."
		logrus.Debug("Pulling latest changes on master branch")
		pullCmd := execCommandFunc("git", "pull", "origin", "master")
		pullCmd.Dir = localPath
		pullCmd.Stdout = os.Stdout
		pullCmd.Stderr = os.Stderr
		if err := pullCmd.Run(); err != nil {
			return fmt.Errorf("failed to pull latest changes: %v", err)
		}
	} else {
		s.Suffix = " Checking out commit " + commit + "..."
		logrus.Debug("Checking out commit ", commit)
		checkoutCmd := execCommandFunc("git", "checkout", commit)
		checkoutCmd.Dir = localPath
		checkoutCmd.Stdout = os.Stdout
		checkoutCmd.Stderr = os.Stderr
		if err := checkoutCmd.Run(); err != nil {
			return fmt.Errorf("failed to checkout commit %s: %v", commit, err)
		}
	}

	cmd := execCommandFunc("git", "rev-parse", "HEAD")
	cmd.Dir = localPath
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to get current commit hash: %v", err)
	}
	commitHash := strings.TrimSpace(out.String())[:7]
	logrus.Debug("Current commit hash: ", commitHash)

	s.Suffix = " Building Neovim..."
	logrus.Debug("Building Neovim...")
	buildCmd := execCommandFunc("make", "CMAKE_BUILD_TYPE=Release")
	buildCmd.Dir = localPath
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("build failed: %v", err)
	}

	binaryPath := filepath.Join(localPath, "build", "bin", "nvim")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		binaryPath = filepath.Join(localPath, "bin", "nvim")
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			return fmt.Errorf("built binary not found")
		}
	}

	targetDir := filepath.Join(versionsDir, commitHash)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create installation directory: %v", err)
	}
	targetPath := filepath.Join(targetDir, "nvim")

	if err := copyFileFunc(binaryPath, targetPath); err != nil {
		return fmt.Errorf("failed to copy built binary: %v", err)
	}

	versionFile := filepath.Join(targetDir, "version.txt")
	if err := os.WriteFile(versionFile, []byte(commitHash), 0644); err != nil {
		return fmt.Errorf("failed to write version file: %v", err)
	}

	s.Suffix = " Build complete!"
	logrus.Debug("Build and installation successful")
	fmt.Printf("\n%s %s\n", utils.SuccessIcon(), utils.CyanText("Build and installation successful!"))
	return nil
}

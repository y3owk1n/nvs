package builder

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/pkg/utils"
)

const repoURL = "https://github.com/neovim/neovim.git"

var execCommandFunc = exec.CommandContext

func runCommandWithSpinner(ctx context.Context, s *spinner.Spinner, cmd *exec.Cmd) error {
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %v", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %v", err)
	}

	// Function to update the spinner based on the output of a given pipe.
	updateSpinner := func(pipeOutput io.Reader, wg *sync.WaitGroup) {
		defer wg.Done()
		scanner := bufio.NewScanner(pipeOutput)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				s.Suffix = " " + line
			}
		}
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go updateSpinner(stdoutPipe, &wg)
	go updateSpinner(stderrPipe, &wg)

	// Channel to capture command completion.
	cmdErrChan := make(chan error, 1)
	go func() {
		cmdErrChan <- cmd.Wait()
	}()

	// Wait for either the command to finish or the context to be done.
	select {
	case <-ctx.Done():
		return fmt.Errorf("command cancelled: %v", ctx.Err())
	case err := <-cmdErrChan:
		// Ensure spinner update goroutines finish.
		wg.Wait()
		if err != nil {
			return err
		}
	}

	return nil
}

func BuildFromCommit(ctx context.Context, commit, versionsDir string) error {
	localPath := filepath.Join(os.TempDir(), "neovim-src")

	logrus.Debug("Temporary path for neovim builder: ", localPath)

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

	if err := runCommandWithSpinner(ctx, s, buildCmd); err != nil {
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

	if err := runCommandWithSpinner(ctx, s, installCmd); err != nil {
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

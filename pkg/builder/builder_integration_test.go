//go:build integration

package builder_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/y3owk1n/nvs/pkg/builder"
	"github.com/y3owk1n/nvs/pkg/helpers"
)

const Abcdef1 = "Abcdef1"

// Global counter to simulate an error only on the first call.
var simulateErrorCount int

// fakeExecCommand is used to override execCommand in tests.
func fakeExecCommand(ctx context.Context, command string, args ...string) *exec.Cmd {
	// If SIMULATE_RETRY is enabled and we haven't yet simulated an error, force an error.
	if os.Getenv("SIMULATE_RETRY") == "1" && simulateErrorCount == 0 {
		simulateErrorCount++
		// Build the helper process command with SIMULATE_ERROR set.
		cs := []string{"-test.run=TestHelperProcess", "--", command}
		cs = append(cs, args...)
		cmd := exec.CommandContext(context.Background(), os.Args[0], cs...)
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1", "SIMULATE_ERROR=1"}

		return cmd
	}

	// Normal behavior: use the helper process simulation.
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.CommandContext(context.Background(), os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}

	return cmd
}

// TestHelperProcess is not a real test. It is invoked as a subprocess
// by fakeExecCommand.
func TestHelperProcess(t *testing.T) {
	// Only run if the special env var is present.
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	// Check if we are simulating an error.
	if os.Getenv("SIMULATE_ERROR") == "1" {
		fmt.Fprint(os.Stderr, "simulated error")
		os.Exit(1)
	}

	// Parse arguments: look for "--" and then the command.
	args := os.Args

	idx := 0
	for i, arg := range args {
		if arg == "--" {
			idx = i + 1

			break
		}
	}

	if idx >= len(args) {
		fmt.Fprint(os.Stderr, "no command provided")
		os.Exit(1)
	}

	cmd := args[idx]
	// Capture any subcommand if provided.
	var subcmd string
	if len(args) > idx+1 {
		subcmd = args[idx+1]
	}

	switch cmd {
	case "git":
		switch subcmd {
		case "clone":
			// Simulate cloning by creating the destination directory.
			dest := args[len(args)-1]

			err := os.MkdirAll(dest, 0o755)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to create dir: %v", err)
				os.Exit(1)
			}

			os.Exit(0)
		case "checkout":
			// Simulate a successful checkout.
			os.Exit(0)
		case "pull":
			// Simulate a successful pull.
			os.Exit(0)
		case "rev-parse":
			// Simulate printing a commit hash.
			// Dummy hash "Abcdef1\n" (at least 7 characters).
			_, _ = fmt.Fprint(os.Stdout, "Abcdef1\n")

			os.Exit(0)
		default:
			os.Exit(0)
		}
	case "make":
		// Simulate a successful build.
		os.Exit(0)
	case "cmake":
		// Simulate the install step.
		// Look for the "--prefix=" argument to get the target directory.
		var prefix string
		for _, arg := range args {
			if strings.HasPrefix(arg, "--prefix=") {
				prefix = arg[len("--prefix="):]

				break
			}
		}

		if prefix != "" {
			// Simulate installation by creating the directory structure and dummy binary.
			targetBin := filepath.Join(prefix, "bin")

			err := os.MkdirAll(targetBin, 0o755)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to create target bin dir: %v", err)
				os.Exit(1)
			}

			nvimPath := filepath.Join(targetBin, "nvim")

			err = os.WriteFile(nvimPath, []byte("installed dummy binary"), 0o755)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to write nvim binary: %v", err)
				os.Exit(1)
			}
		}

		os.Exit(0)
	default:
		os.Exit(0)
	}
}

// TestBuildFromCommit_Master tests builder.BuildFromCommit when commit is "master".
func TestBuildFromCommit_Master(t *testing.T) {
	t.Skip("Integration test requiring neovim build")
	// Reset simulateErrorCount to ensure no simulation.
	simulateErrorCount = 0

	var err error

	err = os.Unsetenv("SIMULATE_RETRY")
	if err != nil {
		t.Errorf("Failed to unset env: %v", err)
	}

	// Override execCommand for testing.
	oldExecCommand := helpers.ExecCommandFunc

	helpers.ExecCommandFunc = fakeExecCommand
	defer func() {
		helpers.ExecCommandFunc = oldExecCommand
	}()

	versionsDir := t.TempDir()

	// Call builder.BuildFromCommit with commit "master".
	err = builder.BuildFromCommit(context.Background(), "master", versionsDir)
	if err != nil {
		t.Fatalf("builder.BuildFromCommit failed: %v", err)
	}

	// builder.BuildFromCommit extracts the commit hash from "git rev-parse",
	// takes its first 7 characters ("Abcdef1"), and uses that as the target directory name.
	targetDir := filepath.Join(versionsDir, "Abcdef1")
	versionFile := filepath.Join(targetDir, "version.txt")

	// Verify that the version file exists and contains the expected content.
	_, err = os.Stat(versionFile)
	if !os.IsNotExist(err) {
		data, err := os.ReadFile(versionFile)
		if err != nil {
			t.Fatalf("failed to read version file: %v", err)
		}

		if strings.TrimSpace(string(data)) != Abcdef1 {
			t.Errorf("version file content = %q; want %q", string(data), Abcdef1)
		}
	} else {
		t.Errorf("version file not found at %s", versionFile)
	}

	// Verify that the installed binary exists.
	installedBinary := filepath.Join(targetDir, "bin", "nvim")

	_, err = os.Stat(installedBinary)
	if os.IsNotExist(err) {
		t.Errorf("installed binary not found at %s", installedBinary)
	}
}

// TestBuildFromCommit_Commit tests builder.BuildFromCommit for a non-master commit.
func TestBuildFromCommit_Commit(t *testing.T) {
	localPath := filepath.Join(os.TempDir(), "neovim-src")
	_ = os.RemoveAll(localPath)

	versionsDir := t.TempDir()

	// Override execCommand for testing.
	oldExecCommand := helpers.ExecCommandFunc

	helpers.ExecCommandFunc = fakeExecCommand
	defer func() {
		helpers.ExecCommandFunc = oldExecCommand
	}()

	// Call builder.BuildFromCommit with a non-master commit (e.g. "abc1234").
	err := builder.BuildFromCommit(context.Background(), "abc1234", versionsDir)
	if err != nil {
		t.Fatalf("builder.BuildFromCommit failed: %v", err)
	}

	// The commit hash is derived from the dummy "git rev-parse" output ("Abcdef1"),
	// so the target directory should be versionsDir/Abcdef1.
	targetDir := filepath.Join(versionsDir, "Abcdef1")
	versionFile := filepath.Join(targetDir, "version.txt")

	data, err := os.ReadFile(versionFile)
	if err != nil {
		t.Fatalf("failed to read version file: %v", err)
	}

	if strings.TrimSpace(string(data)) != Abcdef1 {
		t.Errorf("version file content = %q; want %q", string(data), "Abcdef1")
	}

	// Verify that the installed binary exists.
	installedBinary := filepath.Join(targetDir, "bin", "nvim")

	_, err = os.Stat(installedBinary)
	if os.IsNotExist(err) {
		t.Errorf("installed binary not found at %s", installedBinary)
	}
}

// TestBuildFromCommit_Retry tests that builder.BuildFromCommit will retry once on failure.
func TestBuildFromCommit_Retry(t *testing.T) {
	// Enable error simulation for the first call.
	t.Setenv("SIMULATE_RETRY", "1")
	// Reset the counter.
	simulateErrorCount = 0

	oldExecCommand := helpers.ExecCommandFunc

	helpers.ExecCommandFunc = fakeExecCommand
	defer func() {
		helpers.ExecCommandFunc = oldExecCommand

		_ = os.Unsetenv("SIMULATE_RETRY")
	}()

	// Remove the local clone directory to force cloning.
	localPath := filepath.Join(os.TempDir(), "neovim-src")
	_ = os.RemoveAll(localPath)

	versionsDir := t.TempDir()

	// Call builder.BuildFromCommit; the first attempt will simulate an error,
	// triggering the retry mechanism. On the second attempt, it should succeed.
	err := builder.BuildFromCommit(context.Background(), "master", versionsDir)
	if err != nil {
		t.Fatalf("builder.BuildFromCommit (with retry) failed: %v", err)
	}

	// Verify that the build ultimately succeeded.
	targetDir := filepath.Join(versionsDir, "Abcdef1")
	versionFile := filepath.Join(targetDir, "version.txt")

	data, err := os.ReadFile(versionFile)
	if err != nil {
		t.Fatalf("failed to read version file: %v", err)
	}

	if strings.TrimSpace(string(data)) != Abcdef1 {
		t.Errorf("version file content = %q; want %q", string(data), "Abcdef1")
	}

	installedBinary := filepath.Join(targetDir, "bin", "nvim")

	_, err = os.Stat(installedBinary)
	if os.IsNotExist(err) {
		t.Errorf("installed binary not found at %s", installedBinary)
	}
}

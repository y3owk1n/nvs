package builder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeExecCommand is used to override execCommand in tests.
func fakeExecCommand(ctx context.Context, command string, args ...string) *exec.Cmd {
	// We use the standard helper process trick.
	// The arguments passed to the helper process are:
	//   -test.run=TestHelperProcess
	//   "--", the command, then its args.
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	// Set an env var so the helper process knows it’s supposed to run.
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
			if err := os.MkdirAll(dest, 0755); err != nil {
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
			// Dummy hash "abcdef1\n" (at least 7 characters).
			fmt.Fprint(os.Stdout, "abcdef1\n")
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
			if err := os.MkdirAll(targetBin, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "failed to create target bin dir: %v", err)
				os.Exit(1)
			}
			nvimPath := filepath.Join(targetBin, "nvim")
			if err := os.WriteFile(nvimPath, []byte("installed dummy binary"), 0755); err != nil {
				fmt.Fprintf(os.Stderr, "failed to write nvim binary: %v", err)
				os.Exit(1)
			}
		}
		os.Exit(0)
	default:
		os.Exit(0)
	}
}

// TestBuildFromCommit_Master tests BuildFromCommit when commit is "master".
func TestBuildFromCommit_Master(t *testing.T) {
	// Override execCommand for testing.
	oldExecCommand := execCommandFunc
	execCommandFunc = fakeExecCommand
	defer func() { execCommandFunc = oldExecCommand }()

	// Remove the local clone directory to force cloning.
	localPath := filepath.Join(os.TempDir(), "neovim-src")
	os.RemoveAll(localPath)

	// Create a temporary versions directory.
	versionsDir, err := os.MkdirTemp("", "versions")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(versionsDir)

	// For this test, we don't need to simulate a built binary in the localPath,
	// because the cmake install simulation will create the installed binary.
	// Call BuildFromCommit with commit "master".
	if err := BuildFromCommit(context.Background(), "master", versionsDir); err != nil {
		t.Fatalf("BuildFromCommit failed: %v", err)
	}

	// BuildFromCommit extracts the commit hash from "git rev-parse",
	// takes its first 7 characters ("abcdef1"), and uses that as the target directory name.
	targetDir := filepath.Join(versionsDir, "abcdef1")
	versionFile := filepath.Join(targetDir, "version.txt")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		t.Fatalf("failed to read version file: %v", err)
	}
	if strings.TrimSpace(string(data)) != "abcdef1" {
		t.Errorf("version file content = %q; want %q", string(data), "abcdef1")
	}

	// Verify that the installed binary exists.
	installedBinary := filepath.Join(targetDir, "bin", "nvim")
	if _, err := os.Stat(installedBinary); os.IsNotExist(err) {
		t.Errorf("installed binary not found at %s", installedBinary)
	}
}

// TestBuildFromCommit_Commit tests BuildFromCommit for a non-master commit.
func TestBuildFromCommit_Commit(t *testing.T) {
	oldExecCommand := execCommandFunc
	execCommandFunc = fakeExecCommand
	defer func() { execCommandFunc = oldExecCommand }()

	// Remove the local clone directory to force cloning.
	localPath := filepath.Join(os.TempDir(), "neovim-src")
	os.RemoveAll(localPath)

	versionsDir, err := os.MkdirTemp("", "versions")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(versionsDir)

	// Call BuildFromCommit with a non-master commit (e.g. "abc1234").
	if err := BuildFromCommit(context.Background(), "abc1234", versionsDir); err != nil {
		t.Fatalf("BuildFromCommit failed: %v", err)
	}

	// The commit hash is always derived from the dummy "git rev-parse" output ("abcdef1"),
	// so the target directory should be versionsDir/abcdef1.
	targetDir := filepath.Join(versionsDir, "abcdef1")
	versionFile := filepath.Join(targetDir, "version.txt")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		t.Fatalf("failed to read version file: %v", err)
	}
	if strings.TrimSpace(string(data)) != "abcdef1" {
		t.Errorf("version file content = %q; want %q", string(data), "abcdef1")
	}

	// Verify that the installed binary exists.
	installedBinary := filepath.Join(targetDir, "bin", "nvim")
	if _, err := os.Stat(installedBinary); os.IsNotExist(err) {
		t.Errorf("installed binary not found at %s", installedBinary)
	}
}

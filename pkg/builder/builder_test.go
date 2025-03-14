package builder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeExecCommand is used to override execCommand in tests.
func fakeExecCommand(command string, args ...string) *exec.Cmd {
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

	// Parse the command and its arguments.
	args := os.Args
	// Our argument list should contain "--" followed by the command.
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
	// For convenience, capture any subcommand.
	var subcmd string
	if len(args) > idx+1 {
		subcmd = args[idx+1]
	}

	switch cmd {
	case "git":
		switch subcmd {
		case "clone":
			// Simulate cloning by creating the target directory.
			// The last argument is assumed to be the destination.
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
			// (This dummy hash must be at least 7 characters; we use "abcdef1\n")
			fmt.Fprint(os.Stdout, "abcdef1\n")
			os.Exit(0)
		default:
			os.Exit(0)
		}
	case "make":
		// Simulate a successful build.
		os.Exit(0)
	default:
		os.Exit(0)
	}
}

// TestBuildFromCommit_Master tests the BuildFromCommit function when the commit is "master".
func TestBuildFromCommit_Master(t *testing.T) {
	// Override execCommand and utils.CopyFile for testing.
	oldExecCommand := execCommandFunc
	execCommandFunc = fakeExecCommand
	defer func() { execCommandFunc = oldExecCommand }()

	oldCopyFile := copyFileFunc
	copyFileFunc = func(src, dst string) error {
		// For testing, simply write a dummy binary file to dst.
		return os.WriteFile(dst, []byte("dummy binary"), 0755)
	}
	defer func() { copyFileFunc = oldCopyFile }()

	// Ensure that the local clone directory does not exist so that the clone branch is executed.
	localPath := filepath.Join(os.TempDir(), "neovim-src")
	os.RemoveAll(localPath)

	// Create a temporary versions directory.
	versionsDir, err := os.MkdirTemp("", "versions")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(versionsDir)

	// To simulate a successful build, we need to ensure that the built binary exists.
	// BuildFromCommit checks for the binary in two locations.
	// Here we simulate the second location: localPath/bin/nvim.
	binaryPath := filepath.Join(localPath, "bin", "nvim")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binaryPath, []byte("dummy binary"), 0755); err != nil {
		t.Fatal(err)
	}

	// Call BuildFromCommit with commit "master".
	if err := BuildFromCommit("master", versionsDir); err != nil {
		t.Fatalf("BuildFromCommit failed: %v", err)
	}

	// The BuildFromCommit function extracts the commit hash from "git rev-parse" output,
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
}

// TestBuildFromCommit_Commit tests BuildFromCommit for a non-master commit.
func TestBuildFromCommit_Commit(t *testing.T) {
	oldExecCommand := execCommandFunc
	execCommandFunc = fakeExecCommand
	defer func() { execCommandFunc = oldExecCommand }()

	oldCopyFile := copyFileFunc
	copyFileFunc = func(src, dst string) error {
		return os.WriteFile(dst, []byte("dummy binary"), 0755)
	}
	defer func() { copyFileFunc = oldCopyFile }()

	// Remove the local clone directory so that the clone branch runs.
	localPath := filepath.Join(os.TempDir(), "neovim-src")
	os.RemoveAll(localPath)

	versionsDir, err := os.MkdirTemp("", "versions")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(versionsDir)

	// Create a dummy built binary file for the non-master branch.
	binaryPath := filepath.Join(localPath, "bin", "nvim")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binaryPath, []byte("dummy binary"), 0755); err != nil {
		t.Fatal(err)
	}

	// Call BuildFromCommit with a non-master commit (e.g. "abc1234").
	if err := BuildFromCommit("abc1234", versionsDir); err != nil {
		t.Fatalf("BuildFromCommit failed: %v", err)
	}

	// The commit hash is always derived from the dummy output ("abcdef1").
	targetDir := filepath.Join(versionsDir, "abcdef1")
	versionFile := filepath.Join(targetDir, "version.txt")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		t.Fatalf("failed to read version file: %v", err)
	}
	if strings.TrimSpace(string(data)) != "abcdef1" {
		t.Errorf("version file content = %q; want %q", string(data), "abcdef1")
	}
}

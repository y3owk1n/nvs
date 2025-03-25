// utils_test.go
package utils

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

// TestIsInstalled creates a temporary version directory and checks if IsInstalled returns true.
func TestIsInstalled(t *testing.T) {
	tempDir := t.TempDir()
	version := "v1.0.0"
	installedDir := filepath.Join(tempDir, version)
	if err := os.Mkdir(installedDir, 0755); err != nil {
		t.Fatalf("failed to create version directory: %v", err)
	}

	if !IsInstalled(tempDir, version) {
		t.Errorf("IsInstalled returned false, expected true")
	}

	// Test non-installed version.
	if IsInstalled(tempDir, "nonexistent") {
		t.Errorf("IsInstalled returned true for nonexistent version")
	}
}

// TestListInstalledVersions creates several directories (including a "current" symlink) and verifies the returned list.
func TestListInstalledVersions(t *testing.T) {
	tempDir := t.TempDir()
	versions := []string{"v1.0.0", "v1.1.0", "current"}
	for _, v := range versions {
		if err := os.Mkdir(filepath.Join(tempDir, v), 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", v, err)
		}
	}
	list, err := ListInstalledVersions(tempDir)
	if err != nil {
		t.Fatalf("ListInstalledVersions failed: %v", err)
	}
	// "current" should be excluded.
	if len(list) != 2 {
		t.Errorf("expected 2 versions, got %d", len(list))
	}
}

// TestUpdateSymlink tests that UpdateSymlink creates or updates a symlink.
func TestUpdateSymlink(t *testing.T) {
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "target")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatalf("failed to create target directory: %v", err)
	}
	link := filepath.Join(tempDir, "mylink")
	// Create initial symlink.
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("failed to create initial symlink: %v", err)
	}
	// Create a new target.
	newTarget := filepath.Join(tempDir, "newtarget")
	if err := os.Mkdir(newTarget, 0755); err != nil {
		t.Fatalf("failed to create new target directory: %v", err)
	}
	// Update symlink.
	if err := UpdateSymlink(newTarget, link); err != nil {
		t.Fatalf("UpdateSymlink failed: %v", err)
	}
	resolved, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("failed to read symlink: %v", err)
	}
	if resolved != newTarget {
		t.Errorf("expected symlink to point to %q, got %q", newTarget, resolved)
	}
}

// TestGetCurrentVersion tests that GetCurrentVersion reads the base name from the "current" symlink.
func TestGetCurrentVersion(t *testing.T) {
	tempDir := t.TempDir()
	// Create a fake version directory.
	version := "v1.2.3"
	target := filepath.Join(tempDir, version)
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatalf("failed to create version directory: %v", err)
	}
	// Create a "current" symlink pointing to the version.
	currentLink := filepath.Join(tempDir, "current")
	if err := os.Symlink(target, currentLink); err != nil {
		t.Fatalf("failed to create current symlink: %v", err)
	}
	got, err := GetCurrentVersion(tempDir)
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}
	if got != version {
		t.Errorf("expected %q, got %q", version, got)
	}
}

// TestFindNvimBinary tests that FindNvimBinary returns the expected binary path.
// For Unix-like systems, create a temporary executable file.
func TestFindNvimBinary(t *testing.T) {
	tempDir := t.TempDir()
	var binName string
	if runtime.GOOS == "windows" {
		binName = "nvim.exe"
	} else {
		binName = "nvim"
	}
	binaryPath := filepath.Join(tempDir, binName)
	f, err := os.Create(binaryPath)
	if err != nil {
		t.Fatalf("failed to create fake binary: %v", err)
	}
	f.Close()
	// Make it executable.
	if runtime.GOOS != "windows" {
		if err := os.Chmod(binaryPath, 0755); err != nil {
			t.Fatalf("failed to chmod fake binary: %v", err)
		}
	}
	found := FindNvimBinary(tempDir)
	if found == "" {
		t.Errorf("FindNvimBinary did not find the binary")
	} else if found != binaryPath {
		t.Errorf("expected %q, got %q", binaryPath, found)
	}
}

// TestGetInstalledReleaseIdentifier tests reading a version.txt file.
func TestGetInstalledReleaseIdentifier(t *testing.T) {
	tempDir := t.TempDir()
	alias := "v1.0.0"
	versionFile := filepath.Join(tempDir, alias, "version.txt")
	if err := os.MkdirAll(filepath.Dir(versionFile), 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	content := "v1.0.0\n"
	if err := os.WriteFile(versionFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write version file: %v", err)
	}
	got, err := GetInstalledReleaseIdentifier(tempDir, alias)
	if err != nil {
		t.Fatalf("GetInstalledReleaseIdentifier failed: %v", err)
	}
	if got != strings.TrimSpace(content) {
		t.Errorf("expected %q, got %q", strings.TrimSpace(content), got)
	}
}

// TestLaunchNvimWithConfig tests LaunchNvimWithConfig in two branches.
// 1. When the configuration directory does not exist.
// 2. When it exists but exec.LookPath fails.
// We use go-mpatch to override functions.
func TestLaunchNvimWithConfig(t *testing.T) {
	// Patch os.UserHomeDir to return a temporary directory.
	tempHome := t.TempDir()
	origUserHomeDir := userHomeDir
	userHomeDir = func() (string, error) {
		return tempHome, nil
	}
	defer func() { userHomeDir = origUserHomeDir }()

	// Case 1: Config directory does not exist.
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	LaunchNvimWithConfig("nonexistent-config")
	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = origStdout
	if !strings.Contains(string(out), "✖ configuration") {
		t.Errorf("expected error message for nonexistent configuration, got %q", string(out))
	}

	// Case 2: Config exists but exec.LookPath fails.
	configName := "testconfig"
	configDir := filepath.Join(tempHome, ".config", configName)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	// Patch lookPath to simulate failure.
	origLookPath := lookPath
	lookPath = func(file string) (string, error) {
		return "", errors.New("nvim not found")
	}

	defer func() { lookPath = origLookPath }()

	// Patch fatalf so that it does not exit.
	calledFatal := false
	origFatalf := fatalf
	fatalf = func(format string, args ...any) {
		calledFatal = true
	}
	defer func() { fatalf = origFatalf }()

	LaunchNvimWithConfig(configName)
	if !calledFatal {
		t.Errorf("expected logrus.Fatalf to be called when nvim is not found")
	}
}

// TestClearDirectory creates files and subdirectories, then clears the directory.
func TestClearDirectory(t *testing.T) {
	tempDir := t.TempDir()
	// Create files and directories.
	file1 := filepath.Join(tempDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	// Clear directory.
	if err := ClearDirectory(tempDir); err != nil {
		t.Fatalf("ClearDirectory failed: %v", err)
	}
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected directory to be empty after clearing, got %d entries", len(entries))
	}
}

// TestTimeFormat tests both valid and invalid time strings.
func TestTimeFormat(t *testing.T) {
	valid := "2023-01-02T15:04:05Z"
	formatted := TimeFormat(valid)
	if formatted != "2023-01-02" {
		t.Errorf("expected 2023-01-02, got %q", formatted)
	}
	// For invalid input, the original string should be returned.
	invalid := "not-a-time"
	if TimeFormat(invalid) != invalid {
		t.Errorf("expected %q for invalid input, got %q", invalid, TimeFormat(invalid))
	}
}

// TestColorizeRow tests that each cell in the row is wrapped in the provided color formatting.
func TestColorizeRow(t *testing.T) {
	row := []string{"a", "b", "c"}
	c := color.New(color.FgRed)
	colored := ColorizeRow(row, c)
	for i, cell := range row {
		expected := c.Sprint(cell)
		if colored[i] != expected {
			t.Errorf("expected %q, got %q", expected, colored[i])
		}
	}
}

func TestCopyFile_Success(t *testing.T) {
	// Create a temporary directory for testing.
	tempDir, err := os.MkdirTemp("", "copyfiletest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a temporary source file.
	srcPath := filepath.Join(tempDir, "src.txt")
	content := []byte("Hello, world!")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Define destination file path.
	dstPath := filepath.Join(tempDir, "dst.txt")

	// Call CopyFile.
	if err := CopyFile(srcPath, dstPath); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Read the destination file to verify its contents.
	copied, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(copied) != string(content) {
		t.Errorf("copied content = %q; want %q", string(copied), string(content))
	}

	// On Unix systems, verify the file permissions.
	if runtime.GOOS != "windows" {
		info, err := os.Stat(dstPath)
		if err != nil {
			t.Fatalf("Stat failed: %v", err)
		}
		// Only check the permission bits.
		if info.Mode().Perm() != 0755 {
			t.Errorf("permissions = %o; want %o", info.Mode().Perm(), 0755)
		}
	}
}

func TestCopyFile_SrcNotExist(t *testing.T) {
	// Use a non-existent source file.
	err := CopyFile("nonexistent.src", "shouldnotmatter.dst")
	if err == nil {
		t.Errorf("expected error when source file does not exist")
	}
}

// TestRunCommandWithSpinner_Success tests that RunCommandWithSpinner successfully runs a command
// that writes output to stdout.
func TestRunCommandWithSpinner_Success(t *testing.T) {
	ctx := context.Background()
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Start()
	defer s.Stop()

	// Use the injected execCommandFunc to run a simple command.
	// Do NOT pre-set cmd.Stdout or cmd.Stderr since RunCommandWithSpinner calls StdoutPipe/StdErrPipe.
	cmd := execCommandFunc(ctx, "echo", "Hello, world!")

	err := RunCommandWithSpinner(ctx, s, cmd)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Note: Since RunCommandWithSpinner reads from the command's pipe internally,
	// we do not have direct access to its output in this test.
	// We assume that a successful run implies that the output was read correctly.
}

// TestRunCommandWithSpinner_Cancel tests that RunCommandWithSpinner returns an error when the context is cancelled.
func TestRunCommandWithSpinner_Cancel(t *testing.T) {
	// Create a cancellable context.
	ctx, cancel := context.WithCancel(context.Background())
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Start()
	defer s.Stop()

	// Use a command that would normally run for a while.
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = execCommandFunc(ctx, "ping", "-n", "10", "127.0.0.1")
	} else {
		cmd = execCommandFunc(ctx, "sleep", "10")
	}

	// Cancel the context immediately.
	cancel()

	err := RunCommandWithSpinner(ctx, s, cmd)
	if err == nil {
		t.Fatal("expected error due to cancellation, got nil")
	}
	// Accept error messages that contain either "command cancelled" or "context canceled".
	if !(strings.Contains(err.Error(), "command cancelled") || strings.Contains(err.Error(), "context canceled")) {
		t.Fatalf("expected error to mention cancellation, got %v", err)
	}
}

// TestRunCommandWithSpinner_Error tests that RunCommandWithSpinner returns an error when the command fails to start.
func TestRunCommandWithSpinner_Error(t *testing.T) {
	ctx := context.Background()
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Start()
	defer s.Stop()

	// Override execCommandFunc to simulate a failure to start.
	origFunc := execCommandFunc
	defer func() { execCommandFunc = origFunc }()

	execCommandFunc = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		// Return a command that is guaranteed to fail (non-existent command).
		return exec.CommandContext(ctx, "nonexistent_command_xyz")
	}

	cmd := execCommandFunc(ctx, "nonexistent_command_xyz")
	err := RunCommandWithSpinner(ctx, s, cmd)
	if err == nil {
		t.Fatal("expected error due to invalid command, got nil")
	}
	if !strings.Contains(err.Error(), "failed to get stdout pipe") && !strings.Contains(err.Error(), "failed to start command") {
		t.Fatalf("expected error to mention failure to start command, got %v", err)
	}
}

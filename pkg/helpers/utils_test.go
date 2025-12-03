// utils_test.go
package helpers_test

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
	"github.com/y3owk1n/nvs/pkg/helpers"
)

var ErrNvimNotFoundTest = errors.New("nvim not found")

const windowsOS = "windows"

// execCommandFunc is a variable to allow overriding the exec.CommandContext function in tests.
var execCommandFunc = exec.CommandContext

// TestIsInstalled creates a temporary version directory and checks if IsInstalled returns true.
func TestIsInstalled(t *testing.T) {
	tempDir := t.TempDir()
	version := "v1.0.0"

	installedDir := filepath.Join(tempDir, version)

	err := os.Mkdir(installedDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create version directory: %v", err)
	}

	if !helpers.IsInstalled(tempDir, version) {
		t.Errorf("IsInstalled returned false, expected true")
	}

	// Test non-installed version.
	if helpers.IsInstalled(tempDir, "nonexistent") {
		t.Errorf("IsInstalled returned true for nonexistent version")
	}
}

// TestListInstalledVersions creates several directories (including a "current" symlink) and verifies the returned list.
func TestListInstalledVersions(t *testing.T) {
	tempDir := t.TempDir()

	versions := []string{"v1.0.0", "v1.1.0", "current"}
	for _, v := range versions {
		err := os.Mkdir(filepath.Join(tempDir, v), 0o755)
		if err != nil {
			t.Fatalf("failed to create directory %s: %v", v, err)
		}
	}

	list, err := helpers.ListInstalledVersions(tempDir)
	if err != nil {
		t.Fatalf("ListInstalledVersions failed: %v", err)
	}
	// "current" should be excluded.
	if len(list) != 2 {
		t.Errorf("expected 2 versions, got %d", len(list))
	}
}

// TestUpdateSymlink tests that UpdateSymlink creates or updates a symlink (or junction on windowsOS).
func TestUpdateSymlink(t *testing.T) {
	t.Run("directory link", func(t *testing.T) {
		tempDir := t.TempDir()

		target := filepath.Join(tempDir, "target")

		err := os.Mkdir(target, 0o755)
		if err != nil {
			t.Fatalf("failed to create target directory: %v", err)
		}

		link := filepath.Join(tempDir, "mylink")

		err = helpers.UpdateSymlink(target, link, true)
		if err != nil {
			t.Fatalf("UpdateSymlink failed: %v", err)
		}

		resolved, _ := filepath.EvalSymlinks(link)

		want, _ := filepath.EvalSymlinks(target)
		if resolved != want {
			t.Errorf("expected %q, got %q", want, resolved)
		}
	})

	t.Run("file link", func(t *testing.T) {
		tempDir := t.TempDir()

		targetFile := filepath.Join(tempDir, "target.txt")

		err := os.WriteFile(targetFile, []byte("hello"), 0o644)
		if err != nil {
			t.Fatalf("failed to create target file: %v", err)
		}

		link := filepath.Join(tempDir, "mylink.txt")

		err = helpers.UpdateSymlink(targetFile, link, false)
		if err != nil {
			t.Fatalf("UpdateSymlink failed: %v", err)
		}

		resolved, _ := filepath.EvalSymlinks(link)

		want, _ := filepath.EvalSymlinks(targetFile)
		if resolved != want {
			t.Errorf("expected %q, got %q", want, resolved)
		}
	})
}

// TestGetCurrentVersion tests that GetCurrentVersion reads the base name from the "current" symlink.
func TestGetCurrentVersion(t *testing.T) {
	tempDir := t.TempDir()
	// Create a fake version directory.
	version := "v1.2.3"

	target := filepath.Join(tempDir, version)

	err := os.Mkdir(target, 0o755)
	if err != nil {
		t.Fatalf("failed to create version directory: %v", err)
	}
	// Create a "current" symlink pointing to the version.
	currentLink := filepath.Join(tempDir, "current")

	err = os.Symlink(target, currentLink)
	if err != nil {
		t.Fatalf("failed to create current symlink: %v", err)
	}

	got, err := helpers.GetCurrentVersion(tempDir)
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}

	if got != version {
		t.Errorf("expected %q, got %q", version, got)
	}
}

func withEnv(key, value string, function func()) {
	orig, ok := os.LookupEnv(key)

	_ = os.Setenv(key, value)
	defer func() {
		if ok {
			_ = os.Setenv(key, orig)
		} else {
			_ = os.Unsetenv(key)
		}
	}()

	function()
}

func TestGetStandardNvimConfigDir_XDG(t *testing.T) {
	withEnv("XDG_CONFIG_HOME", "/tmp/xdg", func() {
		dir, err := helpers.GetNvimConfigBaseDir()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "/tmp/xdg"
		if dir != expected {
			t.Errorf("expected %s, got %s", expected, dir)
		}
	})
}

func TestGetStandardNvimConfigDir_windowsOSLocalAppData(t *testing.T) {
	if runtime.GOOS != windowsOS {
		t.Skip("windowsOS-specific test")
	}

	withEnv("XDG_CONFIG_HOME", "", func() {
		withEnv("LOCALAPPDATA", `C:\Temp\LocalAppData`, func() {
			dir, err := helpers.GetNvimConfigBaseDir()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expected := `C:\Temp\LocalAppData`
			if dir != expected {
				t.Errorf("expected %s, got %s", expected, dir)
			}
		})
	})
}

func TestGetStandardNvimConfigDir_windowsOSFallback(t *testing.T) {
	if runtime.GOOS != windowsOS {
		t.Skip("windowsOS-specific test")
	}

	withEnv("XDG_CONFIG_HOME", "", func() {
		withEnv("LOCALAPPDATA", "", func() {
			home, err := os.UserHomeDir()
			if err != nil {
				t.Fatalf("could not get home: %v", err)
			}

			dir, err := helpers.GetNvimConfigBaseDir()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expected := filepath.Join(home, ".config")
			if dir != expected {
				t.Errorf("expected %s, got %s", expected, dir)
			}
		})
	})
}

func TestGetStandardNvimConfigDir_UnixDefault(t *testing.T) {
	if runtime.GOOS == windowsOS {
		t.Skip("non-windowsOS test")
	}

	withEnv("XDG_CONFIG_HOME", "", func() {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("could not get home: %v", err)
		}

		dir, err := helpers.GetNvimConfigBaseDir()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := filepath.Join(home, ".config")
		if dir != expected {
			t.Errorf("expected %s, got %s", expected, dir)
		}
	})
}

// TestFindNvimBinary tests that FindNvimBinary returns the expected binary path.
// For Unix-like systems, create a temporary executable file.
func TestFindNvimBinary(t *testing.T) {
	tempDir := t.TempDir()

	var binName string
	if runtime.GOOS == windowsOS {
		binName = "nvim.exe"
	} else {
		binName = "nvim"
	}

	var binaryPath string
	if runtime.GOOS == windowsOS {
		binaryPath = filepath.Join(tempDir, "nvim", "bin", binName)

		err := os.MkdirAll(filepath.Dir(binaryPath), 0o755)
		if err != nil {
			t.Fatalf("failed to create bin dir: %v", err)
		}
	} else {
		binaryPath = filepath.Join(tempDir, binName)
	}

	f, err := os.Create(binaryPath)
	if err != nil {
		t.Fatalf("failed to create fake binary: %v", err)
	}

	err = f.Close()
	if err != nil {
		t.Errorf("failed to close file: %v", err)
	}
	// Make it executable.
	if runtime.GOOS != windowsOS {
		err := os.Chmod(binaryPath, 0o755)
		if err != nil {
			t.Fatalf("failed to chmod fake binary: %v", err)
		}
	}

	found := helpers.FindNvimBinary(tempDir)
	if found == "" {
		t.Errorf("FindNvimBinary did not find the binary")
	} else {
		switch runtime.GOOS {
		case windowsOS:
			if found != filepath.Join(tempDir, "nvim") {
				t.Errorf("expected %q, got %q", filepath.Join(tempDir, "nvim"), found)
			}
		default:
			if found != binaryPath {
				t.Errorf("expected %q, got %q", binaryPath, found)
			}
		}
	}
}

// TestGetInstalledReleaseIdentifier tests reading a version.txt file.
func TestGetInstalledReleaseIdentifier(t *testing.T) {
	var err error

	tempDir := t.TempDir()
	alias := "v1.0.0"

	versionFile := filepath.Join(tempDir, alias, "version.txt")

	err = os.MkdirAll(filepath.Dir(versionFile), 0o755)
	if err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	content := "v1.0.0\n"

	err = os.WriteFile(versionFile, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("failed to write version file: %v", err)
	}

	got, err := helpers.GetInstalledReleaseIdentifier(tempDir, alias)
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
	var (
		err         error
		nvimBaseDir string
	)
	// Patch os.UserHomeDir to return a temporary directory.
	tempHome := t.TempDir()
	origUserHomeDir := helpers.UserHomeDir

	helpers.UserHomeDir = func() (string, error) {
		return tempHome, nil
	}
	defer func() { helpers.UserHomeDir = origUserHomeDir }()

	// Patch env vars depending on OS.
	var key string
	switch runtime.GOOS {
	case windowsOS:
		key = "LOCALAPPDATA"
	default:
		key = "XDG_CONFIG_HOME"
	}

	t.Setenv(key, tempHome) // ensures GetNvimConfigBaseDir uses this

	// Case 1: Config directory does not exist.
	origStdout := os.Stdout
	reader, w, _ := os.Pipe()
	os.Stdout = w

	helpers.LaunchNvimWithConfig("nonexistent-config")

	err = w.Close()
	if err != nil {
		t.Errorf("failed to close pipe: %v", err)
	}

	out, _ := io.ReadAll(reader)
	os.Stdout = origStdout

	if !strings.Contains(string(out), "✖ configuration") {
		t.Errorf("expected error message for nonexistent configuration, got %q", string(out))
	}

	// Case 2: Config exists but exec.LookPath fails.
	configName := "testconfig"

	nvimBaseDir, err = helpers.GetNvimConfigBaseDir()
	if err != nil {
		t.Fatalf("failed to get nvim base dir: %v", err)
	}

	configDir := filepath.Join(nvimBaseDir, configName)
	t.Logf("configDir: %s", configDir)

	err = os.MkdirAll(configDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	// Patch lookPath to simulate failure.
	origLookPath := helpers.LookPath
	helpers.LookPath = func(file string) (string, error) {
		return "", ErrNvimNotFoundTest
	}

	defer func() { helpers.LookPath = origLookPath }()

	// Patch fatalf so that it does not exit.
	calledFatal := false
	origFatalf := helpers.Fatalf

	helpers.Fatalf = func(format string, args ...any) {
		calledFatal = true
	}
	defer func() { helpers.Fatalf = origFatalf }()

	helpers.LaunchNvimWithConfig(configName)

	if !calledFatal {
		t.Errorf("expected logrus.Fatalf to be called when nvim is not found")
	}
}

// TestClearDirectory creates files and subdirectories, then clears the directory.
func TestClearDirectory(t *testing.T) {
	var (
		err     error
		entries []os.DirEntry
	)

	tempDir := t.TempDir()
	// Create files and directories.
	file1 := filepath.Join(tempDir, "file1.txt")

	err = os.WriteFile(file1, []byte("content"), 0o644)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	subDir := filepath.Join(tempDir, "subdir")

	err = os.Mkdir(subDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	// Clear directory.
	err = helpers.ClearDirectory(tempDir)
	if err != nil {
		t.Fatalf("ClearDirectory failed: %v", err)
	}

	entries, err = os.ReadDir(tempDir)
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

	formatted := helpers.TimeFormat(valid)
	if formatted != "2023-01-02" {
		t.Errorf("expected 2023-01-02, got %q", formatted)
	}
	// For invalid input, the original string should be returned.
	invalid := "not-a-time"
	if helpers.TimeFormat(invalid) != invalid {
		t.Errorf("expected %q for invalid input, got %q", invalid, helpers.TimeFormat(invalid))
	}
}

// TestColorizeRow tests that each cell in the row is wrapped in the provided color formatting.
func TestColorizeRow(t *testing.T) {
	row := []string{"a", "b", "c"}
	c := color.New(color.FgRed)

	colored := helpers.ColorizeRow(row, c)
	for i, cell := range row {
		expected := c.Sprint(cell)
		if colored[i] != expected {
			t.Errorf("expected %q, got %q", expected, colored[i])
		}
	}
}

func TestCopyFile_Success(t *testing.T) {
	var (
		err    error
		copied []byte
	)
	// Create a temporary directory for testing.
	tempDir := t.TempDir()

	// Create a temporary source file.
	srcPath := filepath.Join(tempDir, "src.txt")

	content := []byte("Hello, world!")

	err = os.WriteFile(srcPath, content, 0o644)
	if err != nil {
		t.Fatal(err)
	}

	// Define destination file path.
	dstPath := filepath.Join(tempDir, "dst.txt")

	// Call CopyFile.
	err = helpers.CopyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Read the destination file to verify its contents.
	copied, err = os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(copied) != string(content) {
		t.Errorf("copied content = %q; want %q", string(copied), string(content))
	}

	// On Unix systems, verify the file permissions.
	if runtime.GOOS != windowsOS {
		info, err := os.Stat(dstPath)
		if err != nil {
			t.Fatalf("Stat failed: %v", err)
		}
		// Only check the permission bits.
		if info.Mode().Perm() != 0o755 {
			t.Errorf("permissions = %o; want %o", info.Mode().Perm(), 0o755)
		}
	}
}

func TestCopyFile_SrcNotExist(t *testing.T) {
	// Use a non-existent source file.
	err := helpers.CopyFile("nonexistent.src", "shouldnotmatter.dst")
	if err == nil {
		t.Errorf("expected error when source file does not exist")
	}
}

// TestRunCommandWithSpinner_Success tests that RunCommandWithSpinner successfully runs a command
// that writes output to stdout.
func TestRunCommandWithSpinner_Success(t *testing.T) {
	ctx := context.Background()
	spinner := spinner.New(spinner.CharSets[14], 100*time.Millisecond)

	spinner.Start()
	defer spinner.Stop()

	// Use the injected execCommandFunc to run a simple command.
	// Do NOT pre-set cmd.Stdout or cmd.Stderr since RunCommandWithSpinner calls StdoutPipe/StdErrPipe.
	cmd := execCommandFunc(ctx, "echo", "Hello, world!")

	err := helpers.RunCommandWithSpinner(ctx, spinner, cmd)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Note: Since RunCommandWithSpinner reads from the command's pipe internally,
	// we do not have direct access to its output in this test.
	// We assume that a successful run implies that the output was read correctly.
}

// TestRunCommandWithSpinner_Cancel tests that RunCommandWithSpinner returns an error when the context is canceled.
func TestRunCommandWithSpinner_Cancel(t *testing.T) {
	// Create a cancellable context.
	ctx, cancel := context.WithCancel(context.Background())
	spinner := spinner.New(spinner.CharSets[14], 100*time.Millisecond)

	spinner.Start()
	defer spinner.Stop()

	// Use a command that would normally run for a while.
	var cmd *exec.Cmd
	if runtime.GOOS == windowsOS {
		cmd = execCommandFunc(ctx, "ping", "-n", "10", "127.0.0.1")
	} else {
		cmd = execCommandFunc(ctx, "sleep", "10")
	}

	// Cancel the context immediately.
	cancel()

	err := helpers.RunCommandWithSpinner(ctx, spinner, cmd)
	if err == nil {
		t.Fatal("expected error due to cancellation, got nil")
	}
	// Accept error messages that contain either "command canceled" or "context canceled".
	if !strings.Contains(err.Error(), "command canceled") &&
		!strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("expected error to mention cancellation, got %v", err)
	}
}

// TestRunCommandWithSpinner_Error tests that RunCommandWithSpinner returns an error when the command fails to start.
func TestRunCommandWithSpinner_Error(t *testing.T) {
	ctx := context.Background()
	spinner := spinner.New(spinner.CharSets[14], 100*time.Millisecond)

	spinner.Start()
	defer spinner.Stop()

	// Override execCommandFunc to simulate a failure to start.
	origFunc := execCommandFunc
	defer func() { execCommandFunc = origFunc }()

	execCommandFunc = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		// Return a command that is guaranteed to fail (non-existent command).
		return exec.CommandContext(ctx, "nonexistent_command_xyz")
	}

	cmd := execCommandFunc(ctx, "nonexistent_command_xyz")

	err := helpers.RunCommandWithSpinner(ctx, spinner, cmd)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to get stdout pipe") &&
		!strings.Contains(err.Error(), "failed to start command") {
		t.Fatalf("expected error to mention failure to start command, got %v", err)
	}
}

//go:build integration

package helpers_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/briandowns/spinner"
	"github.com/y3owk1n/nvs/pkg/helpers"
)

const (
	testVersion = "v1.0.0"
	windowsOS   = "windows"
)

func TestFindNvimBinary(t *testing.T) {
	tempDir := t.TempDir()

	var binName string
	if runtime.GOOS == windowsOS {
		binName = "nvim-test.exe"
	} else {
		binName = "nvim-test"
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
		t.Fatalf("failed to create binary: %v", err)
	}

	_ = f.Close()

	if runtime.GOOS != windowsOS {
		err = os.Chmod(binaryPath, 0o755)
		if err != nil {
			t.Fatalf("failed to chmod: %v", err)
		}
	}

	found := helpers.FindNvimBinary(tempDir)
	if found == "" {
		t.Errorf("FindNvimBinary did not find the binary")
	} else if found != binaryPath {
		t.Errorf("expected %q, got %q", binaryPath, found)
	}
}

func TestFindNvimBinary_NoBinary(t *testing.T) {
	tempDir := t.TempDir()

	found := helpers.FindNvimBinary(tempDir)
	if found != "" {
		t.Errorf("expected no binary found, got %s", found)
	}
}

func TestUseVersion(t *testing.T) {
	tempDir := t.TempDir()
	binDir := t.TempDir()

	version := testVersion
	target := filepath.Join(tempDir, version)

	err := os.MkdirAll(target, 0o755)
	if err != nil {
		t.Fatalf("failed to create version dir: %v", err)
	}

	// Create fake nvim binary
	binPath := filepath.Join(target, "nvim")

	f, err := os.Create(binPath)
	if err != nil {
		t.Fatalf("failed to create fake binary: %v", err)
	}

	_ = f.Close()

	if runtime.GOOS != windowsOS {
		err = os.Chmod(binPath, 0o755)
		if err != nil {
			t.Fatalf("failed to chmod: %v", err)
		}
	}

	err = helpers.UseVersion(version, "current", tempDir, binDir)
	if err != nil {
		t.Errorf("UseVersion failed: %v", err)
	}
}

func TestUseVersion_NoBinary(t *testing.T) {
	tempDir := t.TempDir()
	binDir := t.TempDir()

	version := testVersion
	target := filepath.Join(tempDir, version)

	err := os.MkdirAll(target, 0o755)
	if err != nil {
		t.Fatalf("failed to create version dir: %v", err)
	}

	// No binary
	err = helpers.UseVersion(version, "current", tempDir, binDir)
	if err == nil {
		t.Errorf("expected error when binary not found")
	}
}

func TestGetInstalledReleaseIdentifier(t *testing.T) {
	tempDir := t.TempDir()
	versionDir := filepath.Join(tempDir, "v1.0.0")

	err := os.MkdirAll(versionDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create version dir: %v", err)
	}

	versionFile := filepath.Join(versionDir, "version.txt")

	err = os.WriteFile(versionFile, []byte("v1.0.0"), 0o644)
	if err != nil {
		t.Fatalf("failed to create version file: %v", err)
	}

	result, err := helpers.GetInstalledReleaseIdentifier(tempDir, "v1.0.0")
	if err != nil {
		t.Errorf("GetInstalledReleaseIdentifier failed: %v", err)
	}

	if result != "v1.0.0" {
		t.Errorf("expected v1.0.0, got %s", result)
	}
}

func TestGetInstalledReleaseIdentifier_NoFile(t *testing.T) {
	tempDir := t.TempDir()

	_, err := helpers.GetInstalledReleaseIdentifier(tempDir, "v1.0.0")
	if err == nil {
		t.Errorf("expected error when file not exists")
	}
}

func TestLaunchNvimWithConfig(t *testing.T) {
	// This will try to launch nvim, which may not exist, but tests the logic
	helpers.LaunchNvimWithConfig("testconfig")
	// It may succeed or fail depending on system, but just test it doesn't panic
}

func TestClearDirectory(t *testing.T) {
	tempDir := t.TempDir()

	// Create files and subdirectories.
	file1 := filepath.Join(tempDir, "file1.txt")

	err := os.WriteFile(file1, []byte("content"), 0o644)
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

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("expected directory to be empty after clearing, got %d entries", len(entries))
	}
}

func TestClearDirectory_Empty(t *testing.T) {
	tempDir := t.TempDir()

	err := helpers.ClearDirectory(tempDir)
	if err != nil {
		t.Errorf("ClearDirectory failed: %v", err)
	}
}

func TestCopyFile_Success(t *testing.T) {
	tempDir := t.TempDir()

	src := filepath.Join(tempDir, "src.txt")
	dst := filepath.Join(tempDir, "dst.txt")

	err := os.WriteFile(src, []byte("content"), 0o644)
	if err != nil {
		t.Fatalf("failed to create src file: %v", err)
	}

	err = helpers.CopyFile(src, dst)
	if err != nil {
		t.Errorf("CopyFile failed: %v", err)
	}

	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read dst file: %v", err)
	}

	if string(content) != "content" {
		t.Errorf("expected content, got %s", string(content))
	}
}

func TestCopyFile_SrcNotExist(t *testing.T) {
	tempDir := t.TempDir()

	src := filepath.Join(tempDir, "nonexistent.txt")
	dst := filepath.Join(tempDir, "dst.txt")

	err := helpers.CopyFile(src, dst)
	if err == nil {
		t.Errorf("expected error when src does not exist")
	}
}

func TestRunCommandWithSpinner_Success(t *testing.T) {
	ctx := context.Background()
	spinner := spinner.New(spinner.CharSets[14], 100*time.Millisecond)

	spinner.Start()
	defer spinner.Stop()

	cmd := exec.CommandContext(ctx, "echo", "test")

	err := helpers.RunCommandWithSpinner(ctx, spinner, cmd)
	if err != nil {
		t.Errorf("RunCommandWithSpinner failed: %v", err)
	}
}

func TestRunCommandWithSpinner_Cancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	spinner := spinner.New(spinner.CharSets[14], 100*time.Millisecond)

	spinner.Start()
	defer spinner.Stop()

	cmd := exec.CommandContext(ctx, "sleep", "1")

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := helpers.RunCommandWithSpinner(ctx, spinner, cmd)
	if err == nil {
		t.Errorf("expected error when canceled")
	}
}

func TestRunCommandWithSpinner_Error(t *testing.T) {
	ctx := context.Background()
	spinner := spinner.New(spinner.CharSets[14], 100*time.Millisecond)

	spinner.Start()
	defer spinner.Stop()

	// Override ExecCommandFunc to simulate a failure to start.
	origFunc := helpers.ExecCommandFunc
	defer func() { helpers.ExecCommandFunc = origFunc }()

	helpers.ExecCommandFunc = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		// Return a command that is guaranteed to fail (non-existent command).
		return exec.CommandContext(ctx, "nonexistent_command_xyz")
	}

	cmd := helpers.ExecCommandFunc(ctx, "nonexistent_command_xyz")

	err := helpers.RunCommandWithSpinner(ctx, spinner, cmd)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to get stdout pipe") &&
		!strings.Contains(err.Error(), "failed to start command") {
		t.Fatalf("expected error to mention failure to start command, got %v", err)
	}
}

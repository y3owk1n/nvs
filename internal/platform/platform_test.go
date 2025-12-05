package platform_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	platform "github.com/y3owk1n/nvs/internal/platform"
)

func TestFindNvimBinary(t *testing.T) {
	tempDir := t.TempDir()

	var versionDir string
	if runtime.GOOS == platform.WindowsOS {
		// For Windows, create nvim-win64/v0.10.0 structure
		nvimWin64Dir := filepath.Join(tempDir, "nvim-win64")
		versionDir = filepath.Join(nvimWin64Dir, "v0.10.0")
	} else {
		versionDir = filepath.Join(tempDir, "v0.10.0")
	}

	err := os.MkdirAll(versionDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	var binName string
	if runtime.GOOS == platform.WindowsOS {
		binName = "nvim.exe"
	} else {
		binName = "nvim"
	}

	binPath := filepath.Join(versionDir, binName)

	err = os.WriteFile(binPath, []byte("#!/bin/bash\necho nvim"), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	found := platform.FindNvimBinary(tempDir)

	var expected string
	if runtime.GOOS == platform.WindowsOS {
		// On Windows FindNvimBinary returns two levels up from the .exe location.
		expected = filepath.Dir(versionDir)
	} else {
		expected = binPath
	}

	if found != expected {
		t.Errorf("expected %s, got %s", expected, found)
	}
}

func TestFindNvimBinary_Prefixed(t *testing.T) {
	tempDir := t.TempDir()

	var versionDir string
	if runtime.GOOS == platform.WindowsOS {
		// For Windows, create nvim-win64/v0.10.0 structure
		nvimWin64Dir := filepath.Join(tempDir, "nvim-win64")
		versionDir = filepath.Join(nvimWin64Dir, "v0.10.0")
	} else {
		versionDir = filepath.Join(tempDir, "v0.10.0")
	}

	err := os.MkdirAll(versionDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	var binName string
	if runtime.GOOS == platform.WindowsOS {
		binName = "nvim.exe"
	} else {
		binName = "nvim"
	}

	binPath := filepath.Join(versionDir, binName)

	err = os.WriteFile(binPath, []byte("#!/bin/bash\necho nvim"), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	found := platform.FindNvimBinary(tempDir)

	var expected string
	if runtime.GOOS == platform.WindowsOS {
		// On Windows FindNvimBinary returns two levels up from the .exe location.
		expected = filepath.Dir(versionDir)
	} else {
		expected = binPath
	}

	if found != expected {
		t.Errorf("expected %s, got %s", expected, found)
	}
}

func TestFindNvimBinary_NonExecutable(t *testing.T) {
	if runtime.GOOS == platform.WindowsOS {
		t.Skip("Skipping executable permission test on Windows")
	}

	tempDir := t.TempDir()
	versionDir := filepath.Join(tempDir, "v0.10.0")

	err := os.MkdirAll(versionDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Create a non-executable nvim file
	binPath := filepath.Join(versionDir, "nvim")

	err = os.WriteFile(binPath, []byte("#!/bin/bash\necho nvim"), 0o644) // not executable
	if err != nil {
		t.Fatal(err)
	}

	found := platform.FindNvimBinary(tempDir)
	if found != "" {
		t.Errorf("expected empty string for non-executable, got %s", found)
	}
}

func TestFindNvimBinary_EmptyDir(t *testing.T) {
	tempDir := t.TempDir()

	found := platform.FindNvimBinary(tempDir)
	if found != "" {
		t.Errorf("expected empty string for empty dir, got %s", found)
	}
}

func TestFindNvimBinary_InvalidDir(t *testing.T) {
	found := platform.FindNvimBinary("/nonexistent/path/that/does/not/exist")
	if found != "" {
		t.Errorf("expected empty string for invalid dir, got %s", found)
	}
}

func TestUpdateSymlink(t *testing.T) {
	if runtime.GOOS == platform.WindowsOS {
		t.Skip("Skipping symlink test on Windows")
	}

	tempDir := t.TempDir()

	target := filepath.Join(tempDir, "target")

	err := os.MkdirAll(target, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(tempDir, "link")

	err = platform.UpdateSymlink(target, link, true)
	if err != nil {
		t.Fatalf("platform.UpdateSymlink failed: %v", err)
	}

	// Verify symlink
	resolved, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}

	if resolved != target {
		t.Errorf("symlink points to %s, expected %s", resolved, target)
	}
}

func TestUpdateSymlink_UpdateExisting(t *testing.T) {
	if runtime.GOOS == platform.WindowsOS {
		t.Skip("Skipping symlink test on Windows")
	}

	tempDir := t.TempDir()

	target1 := filepath.Join(tempDir, "target1")
	target2 := filepath.Join(tempDir, "target2")

	err := os.MkdirAll(target1, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.MkdirAll(target2, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(tempDir, "link")

	// Create initial symlink
	err = platform.UpdateSymlink(target1, link, true)
	if err != nil {
		t.Fatalf("first UpdateSymlink failed: %v", err)
	}

	// Update to new target
	err = platform.UpdateSymlink(target2, link, true)
	if err != nil {
		t.Fatalf("second UpdateSymlink failed: %v", err)
	}

	// Verify symlink points to new target
	resolved, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}

	if resolved != target2 {
		t.Errorf("symlink points to %s, expected %s", resolved, target2)
	}
}

func TestGetNvimConfigBaseDir(t *testing.T) {
	// Test with XDG_CONFIG_HOME set
	tempDir := t.TempDir()

	t.Setenv("XDG_CONFIG_HOME", tempDir)

	dir, err := platform.GetNvimConfigBaseDir()
	if err != nil {
		t.Fatal(err)
	}

	if dir != tempDir {
		t.Errorf("expected %s, got %s", tempDir, dir)
	}
}

func TestGetNvimConfigBaseDir_Fallback(t *testing.T) {
	// Test without XDG_CONFIG_HOME
	t.Setenv("XDG_CONFIG_HOME", "")

	// For Windows, unset LOCALAPPDATA
	if runtime.GOOS == platform.WindowsOS {
		t.Setenv("LOCALAPPDATA", "")
	}

	dir, err := platform.GetNvimConfigBaseDir()
	if err != nil {
		t.Fatal(err)
	}

	// Should be home/.config
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	expected := filepath.Join(home, ".config")
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestGetNvimConfigBaseDir_WindowsLOCALAPPDATA(t *testing.T) {
	if runtime.GOOS != platform.WindowsOS {
		t.Skip("Skipping Windows-specific test on non-Windows")
	}

	tempDir := t.TempDir()

	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("LOCALAPPDATA", tempDir)

	dir, err := platform.GetNvimConfigBaseDir()
	if err != nil {
		t.Fatal(err)
	}

	if dir != tempDir {
		t.Errorf("expected %s, got %s", tempDir, dir)
	}
}

func TestIsNeovimRunning(t *testing.T) {
	// This test just verifies the function doesn't panic
	// We can't guarantee nvim is/isn't running
	running, count := platform.IsNeovimRunning()

	// count should be >= 0
	if count < 0 {
		t.Errorf("count should be >= 0, got %d", count)
	}

	// if running, count should be > 0
	if running && count == 0 {
		t.Error("if running is true, count should be > 0")
	}

	// if not running, count should be 0
	if !running && count != 0 {
		t.Errorf("if running is false, count should be 0, got %d", count)
	}
}

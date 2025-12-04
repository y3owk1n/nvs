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

	// Create a fake nvim structure
	versionDir := filepath.Join(tempDir, "v0.10.0")

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

	expected := filepath.Join(versionDir, binName)
	if found != expected {
		t.Errorf("expected %s, got %s", expected, found)
	}
}

func TestFindNvimBinary_Prefixed(t *testing.T) {
	tempDir := t.TempDir()

	versionDir := filepath.Join(tempDir, "v0.10.0")

	err := os.MkdirAll(versionDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	var binName string
	if runtime.GOOS == platform.WindowsOS {
		binName = "nvim-v0.10.0.exe"
	} else {
		binName = "nvim-v0.10.0"
	}

	binPath := filepath.Join(versionDir, binName)

	err = os.WriteFile(binPath, []byte("#!/bin/bash\necho nvim"), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	found := platform.FindNvimBinary(tempDir)

	expected := filepath.Join(versionDir, binName)
	if found != expected {
		t.Errorf("expected %s, got %s", expected, found)
	}
}

func TestFindNvimBinary_NotFound(t *testing.T) {
	tempDir := t.TempDir()

	found := platform.FindNvimBinary(tempDir)
	if found != "" {
		t.Errorf("expected empty string, got %s", found)
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
	if platform.WindowsOS == "windows" {
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

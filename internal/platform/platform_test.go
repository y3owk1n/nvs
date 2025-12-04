package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFindNvimBinary(t *testing.T) {
	tempDir := t.TempDir()

	// Create a fake nvim structure
	versionDir := filepath.Join(tempDir, "v0.10.0")
	err := os.MkdirAll(versionDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	var binName string
	if runtime.GOOS == "windows" {
		binName = "nvim.exe"
	} else {
		binName = "nvim"
	}

	binPath := filepath.Join(versionDir, binName)
	err = os.WriteFile(binPath, []byte("#!/bin/bash\necho nvim"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	found := FindNvimBinary(tempDir)
	expected := filepath.Join(versionDir, binName)
	if found != expected {
		t.Errorf("expected %s, got %s", expected, found)
	}
}

func TestFindNvimBinary_Prefixed(t *testing.T) {
	tempDir := t.TempDir()

	versionDir := filepath.Join(tempDir, "v0.10.0")
	err := os.MkdirAll(versionDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	var binName string
	if runtime.GOOS == "windows" {
		binName = "nvim-v0.10.0.exe"
	} else {
		binName = "nvim-v0.10.0"
	}

	binPath := filepath.Join(versionDir, binName)
	err = os.WriteFile(binPath, []byte("#!/bin/bash\necho nvim"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	found := FindNvimBinary(tempDir)
	expected := filepath.Join(versionDir, binName)
	if found != expected {
		t.Errorf("expected %s, got %s", expected, found)
	}
}

func TestFindNvimBinary_NotFound(t *testing.T) {
	tempDir := t.TempDir()

	found := FindNvimBinary(tempDir)
	if found != "" {
		t.Errorf("expected empty string, got %s", found)
	}
}

func TestUpdateSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping symlink test on Windows")
	}

	tempDir := t.TempDir()

	target := filepath.Join(tempDir, "target")
	err := os.MkdirAll(target, 0755)
	if err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(tempDir, "link")

	err = UpdateSymlink(target, link, true)
	if err != nil {
		t.Fatalf("UpdateSymlink failed: %v", err)
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
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	os.Setenv("XDG_CONFIG_HOME", tempDir)

	dir, err := GetNvimConfigBaseDir()
	if err != nil {
		t.Fatal(err)
	}

	if dir != tempDir {
		t.Errorf("expected %s, got %s", tempDir, dir)
	}
}

func TestGetNvimConfigBaseDir_Fallback(t *testing.T) {
	// Test without XDG_CONFIG_HOME
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	oldLocal := os.Getenv("LOCALAPPDATA")
	defer func() {
		if oldXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
		if oldLocal != "" {
			os.Setenv("LOCALAPPDATA", oldLocal)
		} else {
			os.Unsetenv("LOCALAPPDATA")
		}
	}()

	os.Unsetenv("XDG_CONFIG_HOME")

	if runtime.GOOS == "windows" {
		os.Unsetenv("LOCALAPPDATA")
	}

	dir, err := GetNvimConfigBaseDir()
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

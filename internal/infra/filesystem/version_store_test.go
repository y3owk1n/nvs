package filesystem_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/y3owk1n/nvs/internal/domain/version"
	filesystem "github.com/y3owk1n/nvs/internal/infra/filesystem"
)

const windowsOS = "windows"

func TestVersionStore_Switch(t *testing.T) {
	if runtime.GOOS == windowsOS {
		t.Skip("Skipping symlink test on Windows")
	}

	tempDir := t.TempDir()
	binDir := t.TempDir()

	store := filesystem.New(&filesystem.Config{
		VersionsDir:  tempDir,
		GlobalBinDir: binDir,
	})

	// Create version dir
	versionDir := filepath.Join(tempDir, "v1.0.0")

	err := os.MkdirAll(versionDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Create fake nvim binary
	nvimPath := filepath.Join(versionDir, "nvim")

	err = os.WriteFile(nvimPath, []byte("#!/bin/bash\necho nvim"), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	v := version.New("v1.0.0", version.TypeTag, "v1.0.0", "")

	err = store.Switch(v)
	if err != nil {
		t.Fatalf("Switch failed: %v", err)
	}

	// Check current symlink
	currentLink := filepath.Join(tempDir, "current")

	target, err := os.Readlink(currentLink)
	if err != nil {
		t.Fatal(err)
	}

	expectedVersionDir := filepath.Join(tempDir, "v1.0.0")
	if target != expectedVersionDir {
		t.Errorf("current symlink points to %s, expected %s", target, expectedVersionDir)
	}

	// Check global bin symlink
	globalBin := filepath.Join(binDir, "nvim")

	target, err = os.Readlink(globalBin)
	if err != nil {
		t.Fatal(err)
	}

	expectedNvimPath := filepath.Join(versionDir, "nvim")
	if target != expectedNvimPath {
		t.Errorf("global bin symlink points to %s, expected %s", target, expectedNvimPath)
	}
}

func TestVersionStore_List(t *testing.T) {
	tempDir := t.TempDir()
	binDir := t.TempDir()

	store := filesystem.New(&filesystem.Config{
		VersionsDir:  tempDir,
		GlobalBinDir: binDir,
	})

	// Create version directories
	versions := []string{"stable", "nightly", "v0.10.0"}
	for _, v := range versions {
		versionDir := filepath.Join(tempDir, v)

		err := os.MkdirAll(versionDir, 0o755)
		if err != nil {
			t.Fatal(err)
		}
		// Create version.txt
		err = os.WriteFile(filepath.Join(versionDir, "version.txt"), []byte("test-commit"), 0o644)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Create current symlink (should be ignored)
	if runtime.GOOS != windowsOS {
		_ = os.Symlink(filepath.Join(tempDir, "stable"), filepath.Join(tempDir, "current"))
	}

	list, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(list))
	}
}

func TestVersionStore_List_Empty(t *testing.T) {
	tempDir := t.TempDir()
	binDir := t.TempDir()

	store := filesystem.New(&filesystem.Config{
		VersionsDir:  tempDir,
		GlobalBinDir: binDir,
	})

	list, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 0 {
		t.Errorf("Expected 0 versions, got %d", len(list))
	}
}

func TestVersionStore_Current(t *testing.T) {
	if runtime.GOOS == windowsOS {
		t.Skip("Skipping symlink test on Windows")
	}

	tempDir := t.TempDir()
	binDir := t.TempDir()

	store := filesystem.New(&filesystem.Config{
		VersionsDir:  tempDir,
		GlobalBinDir: binDir,
	})

	// Create version directory
	versionDir := filepath.Join(tempDir, "stable")

	err := os.MkdirAll(versionDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Create version.txt
	err = os.WriteFile(filepath.Join(versionDir, "version.txt"), []byte("v0.10.0"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	// Create current symlink
	err = os.Symlink(versionDir, filepath.Join(tempDir, "current"))
	if err != nil {
		t.Fatal(err)
	}

	current, err := store.Current()
	if err != nil {
		t.Fatalf("Current failed: %v", err)
	}

	if current.Name() != "stable" {
		t.Errorf("Expected current name 'stable', got '%s'", current.Name())
	}
}

func TestVersionStore_Current_NoSymlink(t *testing.T) {
	tempDir := t.TempDir()
	binDir := t.TempDir()

	store := filesystem.New(&filesystem.Config{
		VersionsDir:  tempDir,
		GlobalBinDir: binDir,
	})

	_, err := store.Current()
	if err == nil {
		t.Error("Expected error when no current symlink exists")
	}
}

func TestVersionStore_IsInstalled(t *testing.T) {
	tempDir := t.TempDir()
	binDir := t.TempDir()

	store := filesystem.New(&filesystem.Config{
		VersionsDir:  tempDir,
		GlobalBinDir: binDir,
	})

	// Create version directory
	versionDir := filepath.Join(tempDir, "stable")

	err := os.MkdirAll(versionDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	v := version.New("stable", version.TypeStable, "stable", "")

	if !store.IsInstalled(v) {
		t.Error("Expected stable to be installed")
	}

	vNotInstalled := version.New("nightly", version.TypeNightly, "nightly", "")

	if store.IsInstalled(vNotInstalled) {
		t.Error("Expected nightly to NOT be installed")
	}
}

func TestVersionStore_Uninstall(t *testing.T) {
	tempDir := t.TempDir()
	binDir := t.TempDir()

	store := filesystem.New(&filesystem.Config{
		VersionsDir:  tempDir,
		GlobalBinDir: binDir,
	})

	// Create version directory
	versionDir := filepath.Join(tempDir, "v0.10.0")

	err := os.MkdirAll(versionDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	v := version.New("v0.10.0", version.TypeTag, "v0.10.0", "")

	// Uninstall with force=true (no current check)
	err = store.Uninstall(v, true)
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	// Verify directory is removed
	_, err = os.Stat(versionDir)
	if err == nil {
		t.Error("Version directory should have been removed")
	}
}

func TestVersionStore_GetInstalledReleaseIdentifier(t *testing.T) {
	tempDir := t.TempDir()
	binDir := t.TempDir()

	store := filesystem.New(&filesystem.Config{
		VersionsDir:  tempDir,
		GlobalBinDir: binDir,
	})

	// Create version directory with version.txt
	versionDir := filepath.Join(tempDir, "stable")

	err := os.MkdirAll(versionDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	expectedID := "v0.10.0"

	err = os.WriteFile(filepath.Join(versionDir, "version.txt"), []byte(expectedID), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	identifier, err := store.GetInstalledReleaseIdentifier("stable")
	if err != nil {
		t.Fatalf("GetInstalledReleaseIdentifier failed: %v", err)
	}

	if identifier != expectedID {
		t.Errorf("Expected identifier '%s', got '%s'", expectedID, identifier)
	}
}

func TestVersionStore_GetInstalledReleaseIdentifier_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	binDir := t.TempDir()

	store := filesystem.New(&filesystem.Config{
		VersionsDir:  tempDir,
		GlobalBinDir: binDir,
	})

	_, err := store.GetInstalledReleaseIdentifier("nonexistent")
	if err == nil {
		t.Error("Expected error when version.txt doesn't exist")
	}
}

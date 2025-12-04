package filesystem

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/y3owk1n/nvs/internal/domain/version"
)

func TestVersionStore_List(t *testing.T) {
	tempDir := t.TempDir()

	store := New()

	// Create some version directories
	versions := []string{"v1.0.0", "stable", "nightly"}
	for _, v := range versions {
		dir := filepath.Join(tempDir, v)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		// Create version.txt
		versionFile := filepath.Join(dir, "version.txt")
		content := v
		if v == "stable" {
			content = "v1.0.0" // stable points to v1.0.0
		}
		err = os.WriteFile(versionFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to write version file: %v", err)
		}
	}

	listed, err := store.List(tempDir)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(listed) != 3 {
		t.Errorf("expected 3 versions, got %d", len(listed))
	}

	// Check names
	names := make(map[string]bool)
	for _, v := range listed {
		names[v.Name()] = true
	}

	for _, expected := range versions {
		if !names[expected] {
			t.Errorf("missing version: %s", expected)
		}
	}
}

func TestVersionStore_Current(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping symlink test on Windows")
	}

	tempDir := t.TempDir()

	store := New()

	// Create version dir
	versionDir := filepath.Join(tempDir, "stable")
	err := os.MkdirAll(versionDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create version.txt
	versionFile := filepath.Join(versionDir, "version.txt")
	err = os.WriteFile(versionFile, []byte("v1.0.0"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create current symlink
	currentLink := filepath.Join(tempDir, "current")
	err = os.Symlink(versionDir, currentLink)
	if err != nil {
		t.Fatal(err)
	}

	current, err := store.Current(tempDir)
	if err != nil {
		t.Fatalf("Current failed: %v", err)
	}

	if current.Name() != "stable" {
		t.Errorf("expected current name 'stable', got '%s'", current.Name())
	}

	if current.Identifier() != "stable" {
		t.Errorf("expected identifier 'stable', got '%s'", current.Identifier())
	}

	if current.CommitHash() != "v1.0.0" {
		t.Errorf("expected commit hash 'v1.0.0', got '%s'", current.CommitHash())
	}
}

func TestVersionStore_IsInstalled(t *testing.T) {
	tempDir := t.TempDir()

	store := New()

	// Create version dir
	versionDir := filepath.Join(tempDir, "v1.0.0")
	err := os.MkdirAll(versionDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	v := version.New("v1.0.0", version.TypeTag, "v1.0.0", "")

	if !store.IsInstalled(v, tempDir) {
		t.Errorf("expected v1.0.0 to be installed")
	}

	v2 := version.New("v2.0.0", version.TypeTag, "v2.0.0", "")
	if store.IsInstalled(v2, tempDir) {
		t.Errorf("expected v2.0.0 to not be installed")
	}
}

func TestVersionStore_Switch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping symlink test on Windows")
	}

	tempDir := t.TempDir()
	binDir := t.TempDir()

	store := New()

	// Create version dir
	versionDir := filepath.Join(tempDir, "v1.0.0")
	err := os.MkdirAll(versionDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create fake nvim binary
	nvimPath := filepath.Join(versionDir, "nvim")
	err = os.WriteFile(nvimPath, []byte("#!/bin/bash\necho nvim"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	v := version.New("v1.0.0", version.TypeTag, "v1.0.0", "")

	err = store.Switch(v, tempDir, binDir)
	if err != nil {
		t.Fatalf("Switch failed: %v", err)
	}

	// Check current symlink
	currentLink := filepath.Join(tempDir, "current")
	target, err := os.Readlink(currentLink)
	if err != nil {
		t.Fatal(err)
	}

	if filepath.Base(target) != "v1.0.0" {
		t.Errorf("current symlink points to %s, expected v1.0.0", filepath.Base(target))
	}

	// Check global bin symlink
	globalBin := filepath.Join(binDir, "nvim")
	target, err = os.Readlink(globalBin)
	if err != nil {
		t.Fatal(err)
	}

	if filepath.Base(target) != "nvim" {
		t.Errorf("global bin symlink points to %s, expected nvim", filepath.Base(target))
	}
}

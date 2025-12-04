package filesystem_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/y3owk1n/nvs/internal/domain/version"
	filesystem "github.com/y3owk1n/nvs/internal/infra/filesystem"
)

func TestVersionStore_Switch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping symlink test on Windows")
	}

	tempDir := t.TempDir()
	binDir := t.TempDir()

	store := filesystem.New(&filesystem.Config{})

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

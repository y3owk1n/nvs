package config_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/y3owk1n/nvs/internal/app/config"
)

const (
	windowsOS = "windows"
)

func TestService_List(t *testing.T) {
	// Mock the config base dir by setting XDG_CONFIG_HOME
	tempDir := t.TempDir()

	t.Setenv("XDG_CONFIG_HOME", tempDir)

	service := config.New()

	// Create test config directories
	configs := []string{"nvim", "nvim-custom", "other-config"}
	for _, cfg := range configs {
		dir := filepath.Join(tempDir, cfg)

		err := os.MkdirAll(dir, 0o755)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Create a non-nvim dir
	err := os.MkdirAll(filepath.Join(tempDir, "vscode"), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Create a symlink to a nvim config
	linkTarget := filepath.Join(tempDir, "link-target")

	err = os.MkdirAll(linkTarget, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	linkPath := filepath.Join(tempDir, "nvim-link")

	err = os.Symlink(linkTarget, linkPath)
	if err != nil && runtime.GOOS == windowsOS {
		t.Skip("Symlinks not supported on Windows")
	} else if err != nil {
		t.Fatal(err)
	}

	listed, err := service.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Should find nvim, nvim-custom, nvim-link
	expected := map[string]bool{
		"nvim":        true,
		"nvim-custom": true,
		"nvim-link":   true,
	}

	if len(listed) != len(expected) {
		t.Errorf("expected %d configs, got %d: %v", len(expected), len(listed), listed)
	}

	for _, config := range listed {
		if !expected[config] {
			t.Errorf("unexpected config: %s", config)
		}

		delete(expected, config)
	}

	if len(expected) > 0 {
		t.Errorf("missing configs: %v", expected)
	}
}

func TestService_Launch_ConfigNotFound(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv("XDG_CONFIG_HOME", tempDir)

	service := config.New()

	err := service.Launch(context.Background(), "nonexistent-config")
	if err == nil {
		t.Errorf("expected error for nonexistent config")
	}

	// Check it's the right error using sentinel
	if !errors.Is(err, config.ErrConfigNotFound) {
		t.Errorf("expected ErrConfigNotFound, got: %v", err)
	}
}

func TestService_Launch_ConfigExists(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv("XDG_CONFIG_HOME", tempDir)

	service := config.New()

	// Create config dir
	configDir := filepath.Join(tempDir, "test-config")

	err := os.MkdirAll(configDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Mock nvim in PATH
	oldPath := os.Getenv("PATH")

	// Create fake nvim
	fakeNvimDir := t.TempDir()

	fakeNvim := filepath.Join(fakeNvimDir, "nvim")
	if runtime.GOOS == windowsOS {
		fakeNvim += ".exe"
	}

	var content []byte
	if runtime.GOOS == windowsOS {
		content = []byte{} // Empty file for Windows .exe
	} else {
		content = []byte("#!/bin/bash\necho fake nvim")
	}

	err = os.WriteFile(fakeNvim, content, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", fakeNvimDir+string(os.PathListSeparator)+oldPath)

	// Since Launch calls cmd.Run() which would hang, we'll test up to the point before Run
	// The test verifies the setup is correct

	// For this test, we'll just check that it doesn't return ErrConfigNotFound
	err = service.Launch(context.Background(), "test-config")
	// It will fail because nvim exits, but not with config not found
	if err != nil && err.Error() == "configuration not found: test-config" {
		t.Errorf("config should have been found")
	}
}

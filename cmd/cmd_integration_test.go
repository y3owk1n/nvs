//go:build integration

package cmd_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/cmd"
)

const (
	testVersion = "v1.0.0"
	windowsOS   = "windows"
)

func TestRunList(t *testing.T) {
	if runtime.GOOS == windowsOS {
		t.Skip("Skipping symlink test on Windows")
	}

	tempDir := t.TempDir()

	// Set env var
	t.Setenv("NVS_CONFIG_DIR", tempDir)

	cmd.InitConfig()

	// Create some version dirs
	versions := []string{"v1.0.0", "v1.1.0"}
	for _, v := range versions {
		err := os.Mkdir(filepath.Join(cmd.GetVersionsDir(), v), 0o755)
		if err != nil {
			t.Fatalf("failed to create version dir: %v", err)
		}
	}

	// Create current symlink
	current := testVersion

	err := os.Symlink(filepath.Join(cmd.GetVersionsDir(), current), filepath.Join(cmd.GetVersionsDir(), "current"))
	if err != nil {
		t.Fatalf("failed to create current symlink: %v", err)
	}

	// Call RunList
	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	err = cmd.RunList(cobraCmd, []string{})
	if err != nil {
		t.Errorf("RunList failed: %v", err)
	}
}

func TestRunList_NoVersions(t *testing.T) {
	tempDir := t.TempDir()

	// Set env var
	t.Setenv("NVS_CONFIG_DIR", tempDir)

	cmd.InitConfig()

	// No versions

	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	err := cmd.RunList(cobraCmd, []string{})
	if err != nil {
		t.Errorf("RunList failed: %v", err)
	}
}

func TestRunCurrent(t *testing.T) {
	if runtime.GOOS == windowsOS {
		t.Skip("Skipping symlink test on Windows")
	}

	tempDir := t.TempDir()

	// Set env vars
	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)

	cmd.InitConfig()

	// Create current symlink to a version
	version := testVersion
	target := filepath.Join(cmd.GetVersionsDir(), version)

	err := os.Mkdir(target, 0o755)
	if err != nil {
		t.Fatalf("failed to create version dir: %v", err)
	}

	err = os.Symlink(target, filepath.Join(cmd.GetVersionsDir(), "current"))
	if err != nil {
		t.Fatalf("failed to create current symlink: %v", err)
	}

	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	err = cmd.RunCurrent(cobraCmd, []string{})
	if err != nil {
		t.Errorf("RunCurrent failed: %v", err)
	}
}

func TestRunEnv(t *testing.T) {
	cobraCmd := &cobra.Command{}
	cobraCmd.Flags().Bool("source", false, "")
	cobraCmd.Flags().String("shell", "", "")
	cobraCmd.SetContext(context.Background())

	err := cmd.RunEnv(cobraCmd, []string{})
	if err != nil {
		t.Errorf("RunEnv failed: %v", err)
	}
}

func TestRunEnv_Source(t *testing.T) {
	cobraCmd := &cobra.Command{}
	cobraCmd.Flags().Bool("source", false, "") // default false
	cobraCmd.Flags().String("shell", "", "")   // default empty
	_ = cobraCmd.Flags().Set("source", "true")
	_ = cobraCmd.Flags().Set("shell", "bash")
	cobraCmd.SetContext(context.Background())

	err := cmd.RunEnv(cobraCmd, []string{})
	if err != nil {
		t.Errorf("RunEnv source failed: %v", err)
	}
}

func TestExecute(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"nvs", "--help"}

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
}

func TestInitConfig(t *testing.T) {
	cmd.InitConfig()

	if cmd.GetVersionsDir() == "" {
		t.Errorf("versionsDir not set")
	}

	if cmd.GetCacheFilePath() == "" {
		t.Errorf("cacheFilePath not set")
	}

	if cmd.GetGlobalBinDir() == "" {
		t.Errorf("globalBinDir not set")
	}
}

func TestDetectShell(t *testing.T) {
	shell := cmd.DetectShell()
	// DetectShell may return empty in CI environments without a proper shell
	t.Logf("DetectShell returned: %q", shell)
	// Optionally assert non-empty if running in a known environment
}

func TestRunReset(t *testing.T) {
	tempDir := t.TempDir()

	// Mock stdin with "y\n"
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	os.Stdin = reader

	_, err = writer.WriteString("y\n")
	if err != nil {
		t.Fatal(err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatal(err)
	}

	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	// Set env vars to temp
	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)

	cmd.InitConfig()

	err = cmd.RunReset(cobraCmd, []string{})
	if err != nil {
		t.Errorf("RunReset failed: %v", err)
	}
}

func TestRunInstall(t *testing.T) {
	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	// Test with invalid version
	err := cmd.RunInstall(
		cobraCmd,
		[]string{"THIS-VERSION-DOES-NOT-EXIST-FOR-TESTS"},
	)
	if err == nil {
		t.Errorf("expected error for invalid version")
	}
}

func TestRunPath(t *testing.T) {
	tempDir := t.TempDir()

	// Set env var
	t.Setenv("NVS_BIN_DIR", tempDir)

	cmd.InitConfig()

	// Mock stdin with "y\n"
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	os.Stdin = reader

	_, err = writer.WriteString("y\n")
	if err != nil {
		t.Fatal(err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatal(err)
	}

	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	err = cmd.RunPath(cobraCmd, []string{})
	if err != nil {
		t.Errorf("RunPath failed: %v", err)
	}
}

func TestRunConfig(t *testing.T) {
	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	// Test with arg
	err := cmd.RunConfig(cobraCmd, []string{"testconfig"})
	// RunConfig with a nonexistent config will call LaunchNvimWithConfig
	// which may fail in test environments - this is expected
	if err == nil {
		t.Error("expected RunConfig to fail with nonexistent config")
	}
}

func TestRunUse(t *testing.T) {
	tempDir := t.TempDir()

	// Set env vars
	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)

	cmd.InitConfig()

	// Create version (use a fake commit hash to avoid release lookup)
	version := "abc1234"
	target := filepath.Join(cmd.GetVersionsDir(), version)

	err := os.MkdirAll(target, 0o755)
	if err != nil {
		t.Fatalf("failed to create version dir: %v", err)
	}

	// Create binary
	binName := "nvim"
	if runtime.GOOS == "windows" {
		binName = "nvim.exe"
	}

	binPath := filepath.Join(target, binName)

	f, err := os.Create(binPath)
	if err != nil {
		t.Fatalf("failed to create binary: %v", err)
	}

	err = f.Close()
	if err != nil {
		t.Fatal(err)
	}

	if runtime.GOOS != "windows" {
		err = os.Chmod(binPath, 0o755)
		if err != nil {
			t.Fatalf("failed to chmod: %v", err)
		}
	}

	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	err = cmd.RunUse(cobraCmd, []string{version})
	if err != nil {
		t.Errorf("RunUse failed: %v", err)
	}
}

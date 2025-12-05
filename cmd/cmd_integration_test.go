//go:build integration

package cmd_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/cmd"
	appversion "github.com/y3owk1n/nvs/internal/app/version"
	"github.com/y3owk1n/nvs/internal/domain/installer"
	"github.com/y3owk1n/nvs/internal/domain/release"
	"github.com/y3owk1n/nvs/internal/domain/version"
)

const (
	testVersion    = "v1.0.0"
	testCommitHash = "abc1234"
	windowsOS      = "windows"
)

// mockVersionManagerForIntegration implements version.Manager for integration testing.
type mockVersionManagerForIntegration struct {
	installed map[string]bool
	current   version.Version
}

func (m *mockVersionManagerForIntegration) List() ([]version.Version, error) {
	versions := make([]version.Version, 0, len(m.installed))
	for name := range m.installed {
		v := version.New(name, version.TypeTag, name, "")
		versions = append(versions, v)
	}

	return versions, nil
}

func (m *mockVersionManagerForIntegration) Current() (version.Version, error) {
	return m.current, nil
}

func (m *mockVersionManagerForIntegration) Switch(v version.Version) error {
	m.current = v

	return nil
}

func (m *mockVersionManagerForIntegration) IsInstalled(v version.Version) bool {
	return m.installed[v.Name()]
}

func (m *mockVersionManagerForIntegration) Uninstall(v version.Version, force bool) error {
	delete(m.installed, v.Name())

	return nil
}

func (m *mockVersionManagerForIntegration) GetInstalledReleaseIdentifier(
	versionName string,
) (string, error) {
	return versionName, nil
}

// mockInstallerForIntegration implements installer.Installer for integration testing.
type mockInstallerForIntegration struct {
	installed map[string]bool
}

func (m *mockInstallerForIntegration) InstallRelease(
	ctx context.Context,
	rel installer.ReleaseInfo,
	dest, installName string,
	progress installer.ProgressFunc,
) error {
	// Simulate successful installation
	m.installed[installName] = true

	return nil
}

func (m *mockInstallerForIntegration) BuildFromCommit(
	ctx context.Context,
	commit, dest string,
	progress installer.ProgressFunc,
) (string, error) {
	return testCommitHash, nil
}

// mockReleaseRepoForIntegration implements release.Repository for integration testing.
type mockReleaseRepoForIntegration struct {
	releases map[string]release.Release
}

func (m *mockReleaseRepoForIntegration) FindStable(ctx context.Context) (release.Release, error) {
	if rel, ok := m.releases["stable"]; ok {
		return rel, nil
	}

	return release.Release{}, release.ErrReleaseNotFound
}

func (m *mockReleaseRepoForIntegration) FindNightly(ctx context.Context) (release.Release, error) {
	if rel, ok := m.releases["nightly"]; ok {
		return rel, nil
	}

	return release.Release{}, release.ErrReleaseNotFound
}

func (m *mockReleaseRepoForIntegration) FindByTag(
	ctx context.Context,
	tag string,
) (release.Release, error) {
	if rel, ok := m.releases[tag]; ok {
		return rel, nil
	}

	return release.Release{}, release.ErrReleaseNotFound
}

func (m *mockReleaseRepoForIntegration) GetAll(
	ctx context.Context,
	force bool,
) ([]release.Release, error) {
	releases := make([]release.Release, 0, len(m.releases))
	for _, rel := range m.releases {
		releases = append(releases, rel)
	}

	return releases, nil
}

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

	err := os.Symlink(
		filepath.Join(cmd.GetVersionsDir(), current),
		filepath.Join(cmd.GetVersionsDir(), "current"),
	)
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
	version := testCommitHash
	target := filepath.Join(cmd.GetVersionsDir(), version)

	err := os.MkdirAll(target, 0o755)
	if err != nil {
		t.Fatalf("failed to create version dir: %v", err)
	}

	// Create binary
	binName := "nvim"
	if runtime.GOOS == windowsOS {
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

func TestRunUse_InstallAndSwitch(t *testing.T) {
	var err error

	// Test that RunUse installs a missing version and switches to it
	// This tests the regression where use would install but not switch
	tempDir := t.TempDir()

	// Set env vars
	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)
	t.Setenv("NVS_TEST_MODE", "1")

	// Save original services
	originalVersionService := cmd.GetVersionService()
	defer func() {
		// Restore original services
		cmd.SetVersionServiceForTesting(originalVersionService)
	}()

	// Create shared installed map for mocked services
	sharedInstalled := make(map[string]bool)

	// Create mocked services for testing without network dependency
	mockManager := &mockVersionManagerForIntegration{
		installed: sharedInstalled,
		current:   version.Version{},
	}
	mockInstaller := &mockInstallerForIntegration{
		installed: sharedInstalled,
	}
	// Create assets based on current platform
	var assets []release.Asset
	switch runtime.GOOS {
	case "darwin":
		assets = []release.Asset{
			release.NewAsset("macos.tar.gz", "https://example.com/macos.tar.gz", 1000000),
		}
	case "linux":
		assets = []release.Asset{
			release.NewAsset(
				"nvim-linux64.tar.gz",
				"https://example.com/nvim-linux64.tar.gz",
				1000000,
			),
		}
	case "windows":
		assets = []release.Asset{
			release.NewAsset("nvim-win64.zip", "https://example.com/nvim-win64.zip", 1000000),
		}
	default:
		// Fallback for unknown platforms
		assets = []release.Asset{
			release.NewAsset("generic.tar.gz", "https://example.com/generic.tar.gz", 1000000),
		}
	}

	mockReleaseRepo := &mockReleaseRepoForIntegration{
		releases: map[string]release.Release{
			"stable": release.New("stable", false, "abc123", time.Now(), assets),
		},
	}

	// Create service with mocks
	mockService, err := appversion.New(
		mockReleaseRepo,
		mockManager,
		mockInstaller,
		&appversion.Config{
			VersionsDir:   tempDir,
			CacheFilePath: filepath.Join(tempDir, "cache.json"),
			GlobalBinDir:  tempDir,
		},
	)
	if err != nil {
		t.Fatalf("Failed to create mock service: %v", err)
	}

	cmd.SetVersionServiceForTesting(mockService)

	targetVersion := "stable"

	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	// This should install stable and switch to it
	err = cmd.RunUse(cobraCmd, []string{targetVersion})
	if err != nil {
		t.Errorf("RunUse install and switch failed: %v", err)
	}

	// Verify stable is now "installed" (in our mock)
	if !mockManager.installed["stable"] {
		t.Errorf("Stable was not installed")
	}

	// Verify it's current (check our mock)
	if mockManager.current.Name() != "stable" {
		t.Errorf("Current is not stable, got %s", mockManager.current.Name())
	}
}

func TestFullWorkflow(t *testing.T) {
	tempDir := t.TempDir()

	// Set env vars
	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)
	t.Setenv("NVS_TEST_MODE", "1")

	cmd.InitConfig()

	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	// 1. Test initial state - no versions installed
	err := cmd.RunList(cobraCmd, []string{})
	if err != nil {
		t.Errorf("RunList failed: %v", err)
	}

	// 2. Try to get current version (should fail or show none)
	_ = cmd.RunCurrent(cobraCmd, []string{})
	// This may succeed or fail depending on implementation, just ensure it doesn't crash

	// 3. Create a fake installed version for testing (use commit hash to avoid network)
	targetVersion := testCommitHash
	versionDir := filepath.Join(cmd.GetVersionsDir(), targetVersion)

	err = os.MkdirAll(versionDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Create version.txt
	versionFile := filepath.Join(versionDir, "version.txt")

	err = os.WriteFile(versionFile, []byte(targetVersion), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	// Create fake nvim binary
	binName := "nvim"
	if runtime.GOOS == windowsOS {
		binName = "nvim.exe"
	}

	binPath := filepath.Join(versionDir, binName)

	err = os.WriteFile(binPath, []byte("#!/bin/bash\necho test nvim"), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// 4. Test listing versions
	err = cmd.RunList(cobraCmd, []string{})
	if err != nil {
		t.Errorf("RunList with version failed: %v", err)
	}

	// 5. Test switching to the version
	err = cmd.RunUse(cobraCmd, []string{targetVersion})
	if err != nil {
		t.Errorf("RunUse failed: %v", err)
	}

	// 6. Verify it's now current
	currentLink := filepath.Join(cmd.GetVersionsDir(), "current")

	_, err = os.Lstat(currentLink)
	if err != nil {
		t.Errorf("Current symlink not created: %v", err)
	}

	// 7. Test current command
	err = cmd.RunCurrent(cobraCmd, []string{})
	if err != nil {
		t.Errorf("RunCurrent failed: %v", err)
	}

	// 8. Test global bin symlink
	globalBin := filepath.Join(cmd.GetGlobalBinDir(), "nvim")

	_, err = os.Lstat(globalBin)
	if err != nil {
		t.Errorf("Global bin symlink not created: %v", err)
	}

	// 9. Test config operations (if applicable)
	// Note: Config operations may not work in isolated env, but test the functions don't crash
	_ = cmd.RunEnv(cobraCmd, []string{})
	// May fail in test env, but shouldn't crash

	// 10. Test path command (with mocked input)
	oldStdinPath := os.Stdin
	defer func() { os.Stdin = oldStdinPath }()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	// Write to pipe BEFORE setting stdin and calling the command
	_, _ = writer.WriteString("y\n") // Confirm path setup
	_ = writer.Close()
	os.Stdin = reader

	err = cmd.RunPath(cobraCmd, []string{})
	if err != nil {
		t.Errorf("RunPath failed: %v", err)
	}

	// 11. Test reset command (with mocked input)
	reader, writer, err = os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	// Write to pipe BEFORE setting stdin and calling the command
	_, _ = writer.WriteString("y\n") // Confirm reset
	_ = writer.Close()
	os.Stdin = reader

	err = cmd.RunReset(cobraCmd, []string{})
	if err != nil {
		t.Errorf("RunReset failed: %v", err)
	}

	// 12. Verify reset cleaned up symlinks
	_, err = os.Lstat(currentLink)
	if err == nil {
		t.Errorf("Current symlink should have been removed by reset")
	}

	_, err = os.Lstat(globalBin)
	if err == nil {
		t.Errorf("Global bin symlink should have been removed by reset")
	}

	// 13. Test uninstall (recreate version first)
	err = os.MkdirAll(versionDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	err = cmd.RunUninstall(cobraCmd, []string{targetVersion})
	if err != nil {
		t.Errorf("RunUninstall failed: %v", err)
	}

	// Verify version was removed
	_, err = os.Stat(versionDir)
	if err == nil {
		t.Errorf("Version directory should have been removed")
	}
}

func TestRunListRemote(t *testing.T) {
	tempDir := t.TempDir()

	// Set env vars
	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)
	t.Setenv("NVS_TEST_MODE", "1")

	// Save original services
	originalVersionService := cmd.GetVersionService()
	defer func() {
		cmd.SetVersionServiceForTesting(originalVersionService)
	}()

	// Create mocked services
	mockManager := &mockVersionManagerForIntegration{
		installed: make(map[string]bool),
		current:   version.Version{},
	}
	mockInstaller := &mockInstallerForIntegration{
		installed: make(map[string]bool),
	}

	assets := createPlatformAssets()

	mockReleaseRepo := &mockReleaseRepoForIntegration{
		releases: map[string]release.Release{
			"stable":  release.New("stable", false, "abc123", time.Now(), assets),
			"nightly": release.New("nightly", true, "def456", time.Now(), assets),
			"v0.10.0": release.New("v0.10.0", false, "", time.Now(), assets),
		},
	}

	mockService, err := appversion.New(
		mockReleaseRepo,
		mockManager,
		mockInstaller,
		&appversion.Config{
			VersionsDir:   tempDir,
			CacheFilePath: filepath.Join(tempDir, "cache.json"),
			GlobalBinDir:  tempDir,
		},
	)
	if err != nil {
		t.Fatalf("Failed to create mock service: %v", err)
	}

	cmd.SetVersionServiceForTesting(mockService)

	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	err = cmd.RunListRemote(cobraCmd, []string{})
	if err != nil {
		t.Errorf("RunListRemote failed: %v", err)
	}
}

func TestRunListRemote_Force(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)
	t.Setenv("NVS_TEST_MODE", "1")

	originalVersionService := cmd.GetVersionService()
	defer func() {
		cmd.SetVersionServiceForTesting(originalVersionService)
	}()

	mockManager := &mockVersionManagerForIntegration{
		installed: make(map[string]bool),
		current:   version.Version{},
	}
	mockInstaller := &mockInstallerForIntegration{
		installed: make(map[string]bool),
	}

	assets := createPlatformAssets()

	mockReleaseRepo := &mockReleaseRepoForIntegration{
		releases: map[string]release.Release{
			"stable": release.New("stable", false, "abc123", time.Now(), assets),
		},
	}

	mockService, err := appversion.New(
		mockReleaseRepo,
		mockManager,
		mockInstaller,
		&appversion.Config{
			VersionsDir:   tempDir,
			CacheFilePath: filepath.Join(tempDir, "cache.json"),
			GlobalBinDir:  tempDir,
		},
	)
	if err != nil {
		t.Fatalf("Failed to create mock service: %v", err)
	}

	cmd.SetVersionServiceForTesting(mockService)

	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	// Test with force argument
	err = cmd.RunListRemote(cobraCmd, []string{"force"})
	if err != nil {
		t.Errorf("RunListRemote with force failed: %v", err)
	}
}

func TestRunListRemote_WithInstalledVersions(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)
	t.Setenv("NVS_TEST_MODE", "1")

	originalVersionService := cmd.GetVersionService()
	defer func() {
		cmd.SetVersionServiceForTesting(originalVersionService)
	}()

	// Mark stable as installed
	mockManager := &mockVersionManagerForIntegration{
		installed: map[string]bool{"stable": true},
		current:   version.New("stable", version.TypeTag, "stable", ""),
	}
	mockInstaller := &mockInstallerForIntegration{
		installed: map[string]bool{"stable": true},
	}

	assets := createPlatformAssets()

	mockReleaseRepo := &mockReleaseRepoForIntegration{
		releases: map[string]release.Release{
			"stable":  release.New("stable", false, "abc123", time.Now(), assets),
			"nightly": release.New("nightly", true, "def456", time.Now(), assets),
		},
	}

	mockService, err := appversion.New(
		mockReleaseRepo,
		mockManager,
		mockInstaller,
		&appversion.Config{
			VersionsDir:   tempDir,
			CacheFilePath: filepath.Join(tempDir, "cache.json"),
			GlobalBinDir:  tempDir,
		},
	)
	if err != nil {
		t.Fatalf("Failed to create mock service: %v", err)
	}

	cmd.SetVersionServiceForTesting(mockService)

	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	err = cmd.RunListRemote(cobraCmd, []string{})
	if err != nil {
		t.Errorf("RunListRemote with installed versions failed: %v", err)
	}
}

func TestRunUpgrade_InvalidTarget(t *testing.T) {
	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	// Test with invalid target (not stable or nightly)
	err := cmd.RunUpgrade(cobraCmd, []string{"invalid-target"})
	if err == nil {
		t.Errorf("expected error for invalid upgrade target")
	}
}

func TestRunUpgrade_NotInstalled(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)
	t.Setenv("NVS_TEST_MODE", "1")

	originalVersionService := cmd.GetVersionService()
	defer func() {
		cmd.SetVersionServiceForTesting(originalVersionService)
	}()

	// No versions installed
	mockManager := &mockVersionManagerForIntegration{
		installed: make(map[string]bool),
		current:   version.Version{},
	}
	mockInstaller := &mockInstallerForIntegration{
		installed: make(map[string]bool),
	}

	assets := createPlatformAssets()

	mockReleaseRepo := &mockReleaseRepoForIntegration{
		releases: map[string]release.Release{
			"stable": release.New("stable", false, "abc123", time.Now(), assets),
		},
	}

	mockService, err := appversion.New(
		mockReleaseRepo,
		mockManager,
		mockInstaller,
		&appversion.Config{
			VersionsDir:   tempDir,
			CacheFilePath: filepath.Join(tempDir, "cache.json"),
			GlobalBinDir:  tempDir,
		},
	)
	if err != nil {
		t.Fatalf("Failed to create mock service: %v", err)
	}

	cmd.SetVersionServiceForTesting(mockService)

	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	// Should skip (not error) when version not installed
	err = cmd.RunUpgrade(cobraCmd, []string{"stable"})
	if err != nil {
		t.Errorf("RunUpgrade should skip not installed version, got error: %v", err)
	}
}

func TestRunUpgrade_BothVersions(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)
	t.Setenv("NVS_TEST_MODE", "1")

	originalVersionService := cmd.GetVersionService()
	defer func() {
		cmd.SetVersionServiceForTesting(originalVersionService)
	}()

	// No versions installed - both should be skipped
	mockManager := &mockVersionManagerForIntegration{
		installed: make(map[string]bool),
		current:   version.Version{},
	}
	mockInstaller := &mockInstallerForIntegration{
		installed: make(map[string]bool),
	}

	assets := createPlatformAssets()

	mockReleaseRepo := &mockReleaseRepoForIntegration{
		releases: map[string]release.Release{
			"stable":  release.New("stable", false, "abc123", time.Now(), assets),
			"nightly": release.New("nightly", true, "def456", time.Now(), assets),
		},
	}

	mockService, err := appversion.New(
		mockReleaseRepo,
		mockManager,
		mockInstaller,
		&appversion.Config{
			VersionsDir:   tempDir,
			CacheFilePath: filepath.Join(tempDir, "cache.json"),
			GlobalBinDir:  tempDir,
		},
	)
	if err != nil {
		t.Fatalf("Failed to create mock service: %v", err)
	}

	cmd.SetVersionServiceForTesting(mockService)

	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	// No args = both stable and nightly
	err = cmd.RunUpgrade(cobraCmd, []string{})
	if err != nil {
		t.Errorf("RunUpgrade both versions failed: %v", err)
	}
}

func TestRunEnv_SourceFish(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)

	cmd.InitConfig()

	cobraCmd := &cobra.Command{}
	cobraCmd.Flags().Bool("source", false, "")
	cobraCmd.Flags().String("shell", "", "")
	_ = cobraCmd.Flags().Set("source", "true")
	_ = cobraCmd.Flags().Set("shell", "fish")
	cobraCmd.SetContext(context.Background())

	err := cmd.RunEnv(cobraCmd, []string{})
	if err != nil {
		t.Errorf("RunEnv with fish shell failed: %v", err)
	}
}

func TestRunEnv_SourceZsh(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)

	cmd.InitConfig()

	cobraCmd := &cobra.Command{}
	cobraCmd.Flags().Bool("source", false, "")
	cobraCmd.Flags().String("shell", "", "")
	_ = cobraCmd.Flags().Set("source", "true")
	_ = cobraCmd.Flags().Set("shell", "zsh")
	cobraCmd.SetContext(context.Background())

	err := cmd.RunEnv(cobraCmd, []string{})
	if err != nil {
		t.Errorf("RunEnv with zsh shell failed: %v", err)
	}
}

func TestRunEnv_SourceSh(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)

	cmd.InitConfig()

	cobraCmd := &cobra.Command{}
	cobraCmd.Flags().Bool("source", false, "")
	cobraCmd.Flags().String("shell", "", "")
	_ = cobraCmd.Flags().Set("source", "true")
	_ = cobraCmd.Flags().Set("shell", "sh")
	cobraCmd.SetContext(context.Background())

	err := cmd.RunEnv(cobraCmd, []string{})
	if err != nil {
		t.Errorf("RunEnv with sh shell failed: %v", err)
	}
}

// TestReadVersionFile tests reading version file from directory hierarchy.
func TestReadVersionFile(t *testing.T) {
	tempDir := t.TempDir()

	nestedDir := filepath.Join(tempDir, "project", "src")

	err := os.MkdirAll(nestedDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	versionFile := filepath.Join(tempDir, "project", ".nvs-version")

	err = os.WriteFile(versionFile, []byte("v1.0.0\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	version, foundFile, err := cmd.ReadVersionFile(nestedDir, false)
	if err != nil {
		t.Errorf("ReadVersionFile failed: %v", err)
	}

	if version != "v1.0.0" {
		t.Errorf("Version = %q, want %q", version, "v1.0.0")
	}

	if foundFile != versionFile {
		t.Errorf("Found file = %q, want %q", foundFile, versionFile)
	}
}

// TestReadVersionFile_NotFound tests when no version file exists.
func TestReadVersionFile_NotFound(t *testing.T) {
	tempDir := t.TempDir()

	_, _, err := cmd.ReadVersionFile(tempDir, false)
	if err == nil {
		t.Error("ReadVersionFile expected error when no file found")
	}
}

// TestGetNightlyHistory tests getting nightly history.
func TestGetNightlyHistory(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)

	cmd.InitConfig()

	history, err := cmd.GetNightlyHistory()
	if err != nil {
		t.Logf("GetNightlyHistory (first call): %v", err)
	} else if history == nil {
		t.Error("GetNightlyHistory returned nil history")
	}
}

// TestAddNightlyToHistory tests adding nightly to history.
func TestAddNightlyToHistory(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)

	cmd.InitConfig()

	commitHash := "abc1234567890"
	tagName := "nightly"

	err := cmd.AddNightlyToHistory(commitHash, tagName)
	if err != nil {
		t.Errorf("AddNightlyToHistory failed: %v", err)
	}

	history, err := cmd.GetNightlyHistory()
	if err != nil {
		t.Fatalf("GetNightlyHistory failed: %v", err)
	}

	if len(history.Entries) == 0 {
		t.Error("History should have at least one entry")
	}

	found := false
	for _, entry := range history.Entries {
		if entry.CommitHash == commitHash {
			found = true

			break
		}
	}

	if !found {
		t.Errorf("Commit hash %s not found in history", commitHash)
	}
}

// TestRunDoctor tests the doctor command.
func TestRunDoctor(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)

	cmd.InitConfig()

	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	// Doctor may fail if some checks don't pass, but shouldn't panic
	err := cmd.RunDoctor(cobraCmd, []string{})
	t.Logf("RunDoctor result: %v", err)
}

// TestRunHook tests the hook command for different shells.
func TestRunHook(t *testing.T) {
	tests := []struct {
		name    string
		shell   string
		wantErr bool
	}{
		{"bash hook", "bash", false},
		{"zsh hook", "zsh", false},
		{"fish hook", "fish", false},
		{"unsupported shell", "unsupported", true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			cobraCmd := &cobra.Command{}
			cobraCmd.SetContext(context.Background())

			err := cmd.RunHook(cobraCmd, []string{testCase.shell})
			if testCase.wantErr {
				if err == nil {
					t.Errorf("RunHook(%s) expected error, got nil", testCase.shell)
				}
			} else {
				if err != nil {
					t.Errorf("RunHook(%s) unexpected error: %v", testCase.shell, err)
				}
			}
		})
	}
}

// TestRunPin tests the pin command.
func TestRunPin(t *testing.T) {
	if runtime.GOOS == windowsOS {
		t.Skip("Skipping symlink test on Windows")
	}

	tempDir := t.TempDir()

	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)

	cmd.InitConfig()

	versionName := testVersion
	versionDir := filepath.Join(cmd.GetVersionsDir(), versionName)

	err := os.MkdirAll(versionDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	currentLink := filepath.Join(cmd.GetVersionsDir(), "current")

	err = os.Symlink(versionDir, currentLink)
	if err != nil {
		t.Fatal(err)
	}

	t.Chdir(tempDir)

	cobraCmd := &cobra.Command{}
	cobraCmd.Flags().Bool("global", false, "")
	cobraCmd.SetContext(context.Background())

	err = cmd.RunPin(cobraCmd, []string{versionName})
	if err != nil {
		t.Errorf("RunPin with explicit version failed: %v", err)
	}

	// Verify .nvs-version file was created
	versionFile := filepath.Join(tempDir, ".nvs-version")

	data, err := os.ReadFile(versionFile)
	if err != nil {
		t.Errorf("Failed to read version file: %v", err)
	}

	if string(data) != versionName+"\n" {
		t.Errorf("Version file content = %q, want %q", string(data), versionName+"\n")
	}
}

// TestRunRollback_NoHistory tests rollback when no history exists.
func TestRunRollback_NoHistory(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)

	cmd.InitConfig()

	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	err := cmd.RunRollback(cobraCmd, []string{})
	// May error or just show empty list
	t.Logf("RunRollback with no history result: %v", err)
}

// TestRunRun_VersionNotInstalled tests run command with non-existent version.
func TestRunRun_VersionNotInstalled(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)

	cmd.InitConfig()

	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	err := cmd.RunRun(cobraCmd, []string{"nonexistent-version"})
	if err == nil {
		t.Error("RunRun expected error for non-existent version")
	}
}

func TestRunEnv_SourceUnsupportedShell(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)

	cmd.InitConfig()

	cobraCmd := &cobra.Command{}
	cobraCmd.Flags().Bool("source", false, "")
	cobraCmd.Flags().String("shell", "", "")
	_ = cobraCmd.Flags().Set("source", "true")
	_ = cobraCmd.Flags().Set("shell", "unsupported-shell")
	cobraCmd.SetContext(context.Background())

	err := cmd.RunEnv(cobraCmd, []string{})
	if err == nil {
		t.Errorf("expected error for unsupported shell")
	}
}

func TestRunUninstall_NotInstalled(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)

	cmd.InitConfig()

	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	// Try to uninstall a version that doesn't exist
	err := cmd.RunUninstall(cobraCmd, []string{"nonexistent-version"})
	if err == nil {
		t.Errorf("expected error when uninstalling non-existent version")
	}
}

func TestRunUninstall_CurrentAborted(t *testing.T) {
	if runtime.GOOS == windowsOS {
		t.Skip("Skipping symlink test on Windows")
	}

	tempDir := t.TempDir()

	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)

	cmd.InitConfig()

	// Create a version directory
	versionName := "v1.0.0"
	versionDir := filepath.Join(cmd.GetVersionsDir(), versionName)

	err := os.MkdirAll(versionDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Create current symlink pointing to this version
	currentLink := filepath.Join(cmd.GetVersionsDir(), "current")

	err = os.Symlink(versionDir, currentLink)
	if err != nil {
		t.Fatal(err)
	}

	// Mock stdin with "n\n" to abort
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	// Write to pipe BEFORE setting stdin and calling the command
	_, _ = writer.WriteString("n\n") // Abort
	_ = writer.Close()
	os.Stdin = reader

	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	// Should not error - just abort
	err = cmd.RunUninstall(cobraCmd, []string{versionName})
	if err != nil {
		t.Errorf("RunUninstall aborted should not error: %v", err)
	}

	// Version should still exist
	_, err = os.Stat(versionDir)
	if err != nil {
		t.Errorf("Version directory should still exist after abort")
	}
}

func TestRunCurrent_NoCurrent(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)

	cmd.InitConfig()

	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	// No current version set - should return error
	err := cmd.RunCurrent(cobraCmd, []string{})
	if err == nil {
		t.Log("RunCurrent with no current version may or may not error depending on implementation")
	}
}

func TestRunCurrent_WithStable(t *testing.T) {
	if runtime.GOOS == windowsOS {
		t.Skip("Skipping symlink test on Windows")
	}

	tempDir := t.TempDir()

	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)
	t.Setenv("NVS_TEST_MODE", "1")

	cmd.InitConfig()

	// Create stable version directory
	versionDir := filepath.Join(cmd.GetVersionsDir(), "stable")

	err := os.MkdirAll(versionDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Create current symlink
	currentLink := filepath.Join(cmd.GetVersionsDir(), "current")

	err = os.Symlink(versionDir, currentLink)
	if err != nil {
		t.Fatal(err)
	}

	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	err = cmd.RunCurrent(cobraCmd, []string{})
	// May error if release fetch fails, but shouldn't crash
	t.Logf("RunCurrent with stable result: %v", err)
}

func TestRunCurrent_WithNightly(t *testing.T) {
	if runtime.GOOS == windowsOS {
		t.Skip("Skipping symlink test on Windows")
	}

	tempDir := t.TempDir()

	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)
	t.Setenv("NVS_TEST_MODE", "1")

	cmd.InitConfig()

	// Create nightly version directory
	versionDir := filepath.Join(cmd.GetVersionsDir(), "nightly")

	err := os.MkdirAll(versionDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Create current symlink
	currentLink := filepath.Join(cmd.GetVersionsDir(), "current")

	err = os.Symlink(versionDir, currentLink)
	if err != nil {
		t.Fatal(err)
	}

	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	err = cmd.RunCurrent(cobraCmd, []string{})
	// May error if release fetch fails, but shouldn't crash
	t.Logf("RunCurrent with nightly result: %v", err)
}

func TestRunCurrent_WithCommitHash(t *testing.T) {
	if runtime.GOOS == windowsOS {
		t.Skip("Skipping symlink test on Windows")
	}

	tempDir := t.TempDir()

	t.Setenv("NVS_CONFIG_DIR", tempDir)
	t.Setenv("NVS_CACHE_DIR", tempDir)
	t.Setenv("NVS_BIN_DIR", tempDir)

	cmd.InitConfig()

	// Create commit hash version directory
	commitHash := testCommitHash
	versionDir := filepath.Join(cmd.GetVersionsDir(), commitHash)

	err := os.MkdirAll(versionDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Create current symlink
	currentLink := filepath.Join(cmd.GetVersionsDir(), "current")

	err = os.Symlink(versionDir, currentLink)
	if err != nil {
		t.Fatal(err)
	}

	cobraCmd := &cobra.Command{}
	cobraCmd.SetContext(context.Background())

	err = cmd.RunCurrent(cobraCmd, []string{})
	if err != nil {
		t.Errorf("RunCurrent with commit hash failed: %v", err)
	}
}

// createPlatformAssets creates test assets based on the current platform.
func createPlatformAssets() []release.Asset {
	var assets []release.Asset
	switch runtime.GOOS {
	case "darwin":
		assets = []release.Asset{
			release.NewAsset("macos.tar.gz", "https://example.com/macos.tar.gz", 1000000),
		}
	case "linux":
		assets = []release.Asset{
			release.NewAsset(
				"nvim-linux64.tar.gz",
				"https://example.com/nvim-linux64.tar.gz",
				1000000,
			),
		}
	case windowsOS:
		assets = []release.Asset{
			release.NewAsset("nvim-win64.zip", "https://example.com/nvim-win64.zip", 1000000),
		}
	default:
		assets = []release.Asset{
			release.NewAsset("generic.tar.gz", "https://example.com/generic.tar.gz", 1000000),
		}
	}

	return assets
}

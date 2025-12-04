package version

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/y3owk1n/nvs/internal/domain/installer"
	"github.com/y3owk1n/nvs/internal/domain/release"
	"github.com/y3owk1n/nvs/internal/domain/version"
)

// mockReleaseRepo implements release.Repository for testing
type mockReleaseRepo struct {
	stable      release.Release
	nightly     release.Release
	tags        map[string]release.Release
	getAllForce bool // records if GetAll was called with force=true
}

func (m *mockReleaseRepo) FindStable() (release.Release, error) {
	return m.stable, nil
}

func (m *mockReleaseRepo) FindNightly() (release.Release, error) {
	return m.nightly, nil
}

func (m *mockReleaseRepo) FindByTag(tag string) (release.Release, error) {
	return m.tags[tag], nil
}

func (m *mockReleaseRepo) GetAll(force bool) ([]release.Release, error) {
	m.getAllForce = force
	return []release.Release{m.stable, m.nightly}, nil
}

// mockVersionManager implements version.Manager for testing
type mockVersionManager struct {
	installed map[string]bool
	current   version.Version
}

func (m *mockVersionManager) List(versionsDir string) ([]version.Version, error) {
	var versions []version.Version
	for name := range m.installed {
		v := version.New(name, version.TypeTag, name, "")
		versions = append(versions, v)
	}
	return versions, nil
}

func (m *mockVersionManager) Current(versionsDir string) (version.Version, error) {
	return m.current, nil
}

func (m *mockVersionManager) Switch(v version.Version, versionsDir, binDir string) error {
	m.current = v
	return nil
}

func (m *mockVersionManager) IsInstalled(v version.Version, versionsDir string) bool {
	return m.installed[v.Name()]
}

func (m *mockVersionManager) Uninstall(v version.Version, versionsDir string, force bool) error {
	delete(m.installed, v.Name())
	return nil
}

func (m *mockVersionManager) GetInstalledReleaseIdentifier(versionName, versionsDir string) (string, error) {
	return versionName, nil
}

// mockReleaseInfo implements installer.ReleaseInfo for testing
type mockReleaseInfo struct {
	identifier string
}

func (m *mockReleaseInfo) GetAssetURL() (string, error) {
	return "mock-url", nil
}

func (m *mockReleaseInfo) GetChecksumURL() (string, error) {
	return "mock-checksum", nil
}

func (m *mockReleaseInfo) GetIdentifier() string {
	return m.identifier
}

// mockInstaller implements installer.Installer for testing
type mockInstaller struct {
	installed map[string]bool
}

func (m *mockInstaller) InstallRelease(ctx context.Context, rel installer.ReleaseInfo, dest, installName string, progress installer.ProgressFunc) error {
	m.installed[installName] = true
	return nil
}

func (m *mockInstaller) BuildFromCommit(ctx context.Context, commit, dest string) error {
	return nil
}

// mockReleaseRepo uses release.Release directly

func TestService_Use_Stable(t *testing.T) {
	repo := &mockReleaseRepo{
		stable: release.New("v0.10.0", false, "abc123", time.Time{}, nil),
	}
	manager := &mockVersionManager{
		installed: map[string]bool{"stable": true},
		current:   version.New("nightly", version.TypeNightly, "nightly", ""),
	}
	install := &mockInstaller{
		installed: make(map[string]bool),
	}

	service := New(repo, manager, install, &Config{})

	err := service.Use(context.Background(), "stable")
	if err != nil {
		t.Fatalf("Use stable failed: %v", err)
	}

	if manager.current.Name() != "stable" {
		t.Errorf("Expected current version name 'stable', got '%s'", manager.current.Name())
	}

	if manager.current.Type() != version.TypeStable {
		t.Errorf("Expected current version type 'stable', got '%s'", manager.current.Type())
	}

	if manager.current.Identifier() != "v0.10.0" {
		t.Errorf("Expected identifier 'v0.10.0', got '%s'", manager.current.Identifier())
	}
}

func TestService_Use_Nightly(t *testing.T) {
	repo := &mockReleaseRepo{
		nightly: release.New("nightly-2024-12-04", true, "def456", time.Time{}, nil),
	}
	manager := &mockVersionManager{
		installed: map[string]bool{"nightly": true},
		current:   version.New("stable", version.TypeStable, "v0.9.0", ""),
	}
	install := &mockInstaller{}

	service := New(repo, manager, install, &Config{})

	err := service.Use(context.Background(), "nightly")
	if err != nil {
		t.Fatalf("Use nightly failed: %v", err)
	}

	if manager.current.Name() != "nightly" {
		t.Errorf("Expected current version name 'nightly', got '%s'", manager.current.Name())
	}

	if manager.current.Type() != version.TypeNightly {
		t.Errorf("Expected current version type 'nightly', got '%s'", manager.current.Type())
	}
}

func TestService_Use_Tag(t *testing.T) {
	repo := &mockReleaseRepo{
		tags: map[string]release.Release{
			"v0.9.5": release.New("v0.9.5", false, "ghi789", time.Time{}, nil),
		},
	}
	manager := &mockVersionManager{
		installed: map[string]bool{"v0.9.5": true},
	}
	install := &mockInstaller{}

	service := New(repo, manager, install, &Config{})

	err := service.Use(context.Background(), "v0.9.5")
	if err != nil {
		t.Fatalf("Use v0.9.5 failed: %v", err)
	}

	if manager.current.Name() != "v0.9.5" {
		t.Errorf("Expected current version name 'v0.9.5', got '%s'", manager.current.Name())
	}

	if manager.current.Type() != version.TypeTag {
		t.Errorf("Expected current version type 'tag', got '%s'", manager.current.Type())
	}
}

func TestService_ListRemote_ForceFalse(t *testing.T) {
	repo := &mockReleaseRepo{
		stable:  release.New("v0.10.0", false, "abc123", time.Time{}, nil),
		nightly: release.New("nightly-2024-12-04", true, "def456", time.Time{}, nil),
	}
	manager := &mockVersionManager{}
	install := &mockInstaller{}

	service := New(repo, manager, install, &Config{})

	_, err := service.ListRemote(false)
	if err != nil {
		t.Fatalf("ListRemote(false) failed: %v", err)
	}

	if repo.getAllForce {
		t.Errorf("Expected GetAll to be called with force=false, but was true")
	}
}

func TestService_ListRemote_ForceTrue(t *testing.T) {
	repo := &mockReleaseRepo{
		stable:  release.New("v0.10.0", false, "abc123", time.Time{}, nil),
		nightly: release.New("nightly-2024-12-04", true, "def456", time.Time{}, nil),
	}
	manager := &mockVersionManager{}
	install := &mockInstaller{}

	service := New(repo, manager, install, &Config{})

	_, err := service.ListRemote(true)
	if err != nil {
		t.Fatalf("ListRemote(true) failed: %v", err)
	}

	if !repo.getAllForce {
		t.Errorf("Expected GetAll to be called with force=true, but was false")
	}
}

func TestService_Use_VersionNotFound(t *testing.T) {
	repo := &mockReleaseRepo{
		stable: release.New("v0.10.0", false, "abc123", time.Time{}, nil),
	}
	manager := &mockVersionManager{
		installed: map[string]bool{}, // nightly not installed
	}
	install := &mockInstaller{}

	service := New(repo, manager, install, &Config{})

	err := service.Use(context.Background(), "nightly")
	if err == nil {
		t.Fatalf("Expected error for non-installed version, got nil")
	}

	if !errors.Is(err, version.ErrVersionNotFound) {
		t.Errorf("Expected ErrVersionNotFound, got %v", err)
	}
}

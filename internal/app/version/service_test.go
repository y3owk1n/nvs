package version_test

import (
	"context"
	"errors"
	"testing"
	"time"

	appversion "github.com/y3owk1n/nvs/internal/app/version"
	"github.com/y3owk1n/nvs/internal/domain/installer"
	"github.com/y3owk1n/nvs/internal/domain/release"
	"github.com/y3owk1n/nvs/internal/domain/version"
)

// mockReleaseRepo implements release.Repository for testing.
var errTagNotFound = errors.New("tag not found")

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
	rel, ok := m.tags[tag]
	if !ok {
		return release.Release{}, errTagNotFound
	}

	return rel, nil
}

func (m *mockReleaseRepo) GetAll(force bool) ([]release.Release, error) {
	m.getAllForce = force

	return []release.Release{m.stable, m.nightly}, nil
}

// mockVersionManager implements version.Manager for testing.
type mockVersionManager struct {
	installed map[string]bool
	current   version.Version
}

func (m *mockVersionManager) List() ([]version.Version, error) {
	versions := make([]version.Version, 0, len(m.installed))
	for name := range m.installed {
		v := version.New(name, version.TypeTag, name, "")
		versions = append(versions, v)
	}

	return versions, nil
}

func (m *mockVersionManager) Current() (version.Version, error) {
	return m.current, nil
}

func (m *mockVersionManager) Switch(v version.Version) error {
	m.current = v

	return nil
}

func (m *mockVersionManager) IsInstalled(v version.Version) bool {
	return m.installed[v.Name()]
}

func (m *mockVersionManager) Uninstall(v version.Version, force bool) error {
	delete(m.installed, v.Name())

	return nil
}

func (m *mockVersionManager) GetInstalledReleaseIdentifier(versionName string) (string, error) {
	return versionName, nil
}

// mockInstaller implements installer.Installer for testing.
type mockInstaller struct {
	installed             map[string]bool
	buildFromCommitCalled bool
	lastCommit            string
}

func (m *mockInstaller) InstallRelease(
	ctx context.Context,
	rel installer.ReleaseInfo,
	dest, installName string,
	progress installer.ProgressFunc,
) error {
	m.installed[installName] = true

	return nil
}

func (m *mockInstaller) BuildFromCommit(ctx context.Context, commit, dest string) error {
	m.buildFromCommitCalled = true
	m.lastCommit = commit
	return nil
}

// mockReleaseRepo uses release.Release directly

func TestService_Use_Stable(t *testing.T) {
	repo := &mockReleaseRepo{
		stable: release.New("v0.10.0", false, "abc123", time.Time{}, nil),
	}
	manager := &mockVersionManager{
		installed: map[string]bool{appversion.StableVersion: true},
		current: version.New(
			appversion.NightlyVersion,
			version.TypeNightly,
			appversion.NightlyVersion,
			"",
		),
	}
	install := &mockInstaller{
		installed: make(map[string]bool),
	}

	service := appversion.New(repo, manager, install, &appversion.Config{})

	err := service.Use(context.Background(), appversion.StableVersion)
	if err != nil {
		t.Fatalf("Use stable failed: %v", err)
	}

	if manager.current.Name() != appversion.StableVersion {
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
		installed: map[string]bool{appversion.NightlyVersion: true},
		current:   version.New(appversion.StableVersion, version.TypeStable, "v0.9.0", ""),
	}
	install := &mockInstaller{installed: make(map[string]bool)}

	service := appversion.New(repo, manager, install, &appversion.Config{})

	err := service.Use(context.Background(), appversion.NightlyVersion)
	if err != nil {
		t.Fatalf("Use nightly failed: %v", err)
	}

	if manager.current.Name() != appversion.NightlyVersion {
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
	install := &mockInstaller{installed: make(map[string]bool)}

	service := appversion.New(repo, manager, install, &appversion.Config{})

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
	install := &mockInstaller{installed: make(map[string]bool)}

	service := appversion.New(repo, manager, install, &appversion.Config{})

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
	install := &mockInstaller{installed: make(map[string]bool)}

	service := appversion.New(repo, manager, install, &appversion.Config{})

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
	install := &mockInstaller{installed: make(map[string]bool)}

	service := appversion.New(repo, manager, install, &appversion.Config{})

	err := service.Use(context.Background(), appversion.NightlyVersion)
	if err == nil {
		t.Fatalf("Expected error for non-installed version, got nil")
	}

	if !errors.Is(err, version.ErrVersionNotFound) {
		t.Errorf("Expected ErrVersionNotFound, got %v", err)
	}
}

func TestService_Install_CommitHash(t *testing.T) {
	// Test installing from a commit hash
	repo := &mockReleaseRepo{}
	manager := &mockVersionManager{
		installed: make(map[string]bool),
	}
	install := &mockInstaller{installed: make(map[string]bool)}

	service := appversion.New(repo, manager, install, &appversion.Config{})

	commitHash := "abc123def456"
	err := service.Install(context.Background(), commitHash, nil)
	if err != nil {
		t.Errorf("Install commit hash failed: %v", err)
	}

	// Verify BuildFromCommit was called
	if !install.buildFromCommitCalled {
		t.Errorf("BuildFromCommit should have been called")
	}
	if install.lastCommit != commitHash {
		t.Errorf("Expected commit %s, got %s", commitHash, install.lastCommit)
	}
}

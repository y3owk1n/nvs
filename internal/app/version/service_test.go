package version_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	appversion "github.com/y3owk1n/nvs/internal/app/version"
	"github.com/y3owk1n/nvs/internal/domain/installer"
	"github.com/y3owk1n/nvs/internal/domain/release"
	"github.com/y3owk1n/nvs/internal/domain/version"
)

var errTagNotFound = errors.New("tag not found")

const testVersionTag = "v0.10.0"

type mockReleaseRepo struct {
	stable         release.Release
	nightly        release.Release
	tags           map[string]release.Release
	findNightlyErr error
	getAllForce    bool // records if GetAll was called with force=true
}

func (m *mockReleaseRepo) FindStable(ctx context.Context) (release.Release, error) {
	return m.stable, nil
}

func (m *mockReleaseRepo) FindNightly(ctx context.Context) (release.Release, error) {
	if m.findNightlyErr != nil {
		return release.Release{}, m.findNightlyErr
	}

	return m.nightly, nil
}

func (m *mockReleaseRepo) FindByTag(ctx context.Context, tag string) (release.Release, error) {
	rel, ok := m.tags[tag]
	if !ok {
		return release.Release{}, errTagNotFound
	}

	return rel, nil
}

func (m *mockReleaseRepo) GetAll(ctx context.Context, force bool) ([]release.Release, error) {
	m.getAllForce = force

	return []release.Release{m.stable, m.nightly}, nil
}

// mockVersionManager implements version.Manager for testing.
type mockVersionManager struct {
	installed   map[string]version.Version
	current     version.Version
	identifiers map[string]string
}

func (m *mockVersionManager) List() ([]version.Version, error) {
	versions := make([]version.Version, 0, len(m.installed))
	for _, v := range m.installed {
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
	_, exists := m.installed[v.Name()]

	return exists
}

func (m *mockVersionManager) Uninstall(v version.Version, force bool) error {
	delete(m.installed, v.Name())

	return nil
}

func (m *mockVersionManager) GetInstalledReleaseIdentifier(versionName string) (string, error) {
	if id, ok := m.identifiers[versionName]; ok {
		return id, nil
	}

	return versionName, nil
}

// mockInstaller implements installer.Installer for testing.
type mockInstaller struct {
	installed             map[string]version.Version
	buildFromCommitCalled bool
	lastCommit            string
	lastDest              string
}

func (m *mockInstaller) InstallRelease(
	ctx context.Context,
	rel installer.ReleaseInfo,
	dest, installName string,
	progress installer.ProgressFunc,
) error {
	// Create a version with the installed name (using TypeTag as default)
	v := version.New(installName, version.TypeTag, installName, "")
	m.installed[installName] = v

	return nil
}

func (m *mockInstaller) BuildFromCommit(ctx context.Context, commit, dest string) error {
	m.buildFromCommitCalled = true
	m.lastCommit = commit
	m.lastDest = dest

	return nil
}

// mockReleaseRepo uses release.Release directly

func TestService_Use_Stable(t *testing.T) {
	repo := &mockReleaseRepo{
		stable: release.New("v0.10.0", false, "abc123", time.Time{}, nil),
	}
	manager := &mockVersionManager{
		installed: map[string]version.Version{
			appversion.StableVersion: version.New(
				appversion.StableVersion,
				version.TypeStable,
				"v0.10.0",
				"abc123",
			),
		},
		current: version.New(
			appversion.NightlyVersion,
			version.TypeNightly,
			appversion.NightlyVersion,
			"",
		),
	}
	install := &mockInstaller{
		installed: make(map[string]version.Version),
	}

	service, newErr := appversion.New(
		repo,
		manager,
		install,
		&appversion.Config{VersionsDir: "/tmp"},
	)
	if newErr != nil {
		t.Fatalf("Failed to create service: %v", newErr)
	}

	resolvedVersion, err := service.Use(context.Background(), appversion.StableVersion)
	if err != nil {
		t.Fatalf("Use stable failed: %v", err)
	}

	if manager.current.Name() != appversion.StableVersion {
		t.Errorf("Expected current version name 'stable', got '%s'", manager.current.Name())
	}

	if manager.current.Type() != version.TypeStable {
		t.Errorf("Expected current version type 'stable', got '%s'", manager.current.Type())
	}

	if manager.current.Identifier() != testVersionTag {
		t.Errorf(
			"Expected current version identifier 'v0.10.0', got '%s'",
			manager.current.Identifier(),
		)
	}

	if manager.current.CommitHash() != "abc123" {
		t.Errorf(
			"Expected current version commit hash 'abc123', got '%s'",
			manager.current.CommitHash(),
		)
	}

	if resolvedVersion != "v0.10.0" {
		t.Errorf("Expected resolved version 'v0.10.0', got '%s'", resolvedVersion)
	}
}

func TestService_Use_Nightly_NotAvailable(t *testing.T) {
	repo := &mockReleaseRepo{
		findNightlyErr: release.ErrNoNightlyRelease,
	}
	manager := &mockVersionManager{}
	install := &mockInstaller{}

	service, newErr := appversion.New(
		repo,
		manager,
		install,
		&appversion.Config{VersionsDir: "/tmp"},
	)
	if newErr != nil {
		t.Fatalf("Failed to create service: %v", newErr)
	}

	_, err := service.Use(context.Background(), appversion.NightlyVersion)
	if err == nil {
		t.Error("Expected error when nightly release is not available")
	}
}

func TestService_Use_Tag(t *testing.T) {
	repo := &mockReleaseRepo{
		tags: map[string]release.Release{
			"v0.9.5": release.New("v0.9.5", false, "ghi789", time.Time{}, nil),
		},
	}
	manager := &mockVersionManager{
		installed: map[string]version.Version{
			"v0.9.5": version.New("v0.9.5", version.TypeTag, "v0.9.5", ""),
		},
	}
	install := &mockInstaller{installed: make(map[string]version.Version)}

	service, err := appversion.New(repo, manager, install, &appversion.Config{VersionsDir: "/tmp"})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = service.Use(context.Background(), "v0.9.5")
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
	install := &mockInstaller{installed: make(map[string]version.Version)}

	service, err := appversion.New(repo, manager, install, &appversion.Config{VersionsDir: "/tmp"})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = service.ListRemote(context.Background(), false)
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
	install := &mockInstaller{installed: make(map[string]version.Version)}

	service, err := appversion.New(repo, manager, install, &appversion.Config{VersionsDir: "/tmp"})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = service.ListRemote(context.Background(), true)
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
		installed: make(map[string]version.Version), // nightly not installed
	}
	install := &mockInstaller{installed: make(map[string]version.Version)}

	service, err := appversion.New(repo, manager, install, &appversion.Config{VersionsDir: "/tmp"})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = service.Use(context.Background(), appversion.NightlyVersion)
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
		installed: make(map[string]version.Version),
	}
	install := &mockInstaller{installed: make(map[string]version.Version)}

	config := &appversion.Config{VersionsDir: "/tmp/versions"}

	service, err := appversion.New(repo, manager, install, config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	commitHash := "abc123def456"

	err = service.Install(context.Background(), commitHash, nil)
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

	expectedDest := filepath.Join(config.VersionsDir, commitHash)
	if install.lastDest != expectedDest {
		t.Errorf("Expected dest %s, got %s", expectedDest, install.lastDest)
	}
}

func TestService_List(t *testing.T) {
	repo := &mockReleaseRepo{}
	manager := &mockVersionManager{
		installed: map[string]version.Version{
			"stable":  version.New("stable", version.TypeStable, "v0.10.0", ""),
			"nightly": version.New("nightly", version.TypeNightly, "nightly", "abc123"),
		},
	}
	install := &mockInstaller{installed: make(map[string]version.Version)}

	service, err := appversion.New(repo, manager, install, &appversion.Config{VersionsDir: "/tmp"})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	list, err := service.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(list))
	}
}

func TestService_Current(t *testing.T) {
	repo := &mockReleaseRepo{}
	manager := &mockVersionManager{
		current: version.New("stable", version.TypeStable, "v0.10.0", ""),
	}
	install := &mockInstaller{installed: make(map[string]version.Version)}

	service, err := appversion.New(repo, manager, install, &appversion.Config{VersionsDir: "/tmp"})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	current, err := service.Current()
	if err != nil {
		t.Fatalf("Current failed: %v", err)
	}

	if current.Name() != "stable" {
		t.Errorf("Expected current name 'stable', got '%s'", current.Name())
	}
}

func TestService_Uninstall(t *testing.T) {
	repo := &mockReleaseRepo{}
	manager := &mockVersionManager{
		installed: map[string]version.Version{
			"v0.10.0": version.New("v0.10.0", version.TypeTag, "v0.10.0", ""),
		},
	}
	install := &mockInstaller{installed: make(map[string]version.Version)}

	service, err := appversion.New(repo, manager, install, &appversion.Config{VersionsDir: "/tmp"})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	err = service.Uninstall("v0.10.0", true)
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	// Verify version was removed from manager
	if _, exists := manager.installed["v0.10.0"]; exists {
		t.Error("Version should have been removed from manager")
	}
}

func TestService_Uninstall_NotInstalled(t *testing.T) {
	repo := &mockReleaseRepo{}
	manager := &mockVersionManager{
		installed: make(map[string]version.Version),
	}
	install := &mockInstaller{installed: make(map[string]version.Version)}

	service, err := appversion.New(repo, manager, install, &appversion.Config{VersionsDir: "/tmp"})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	err = service.Uninstall("nonexistent", true)
	if err == nil {
		t.Error("Expected error when uninstalling non-existent version")
	}
}

func TestService_IsVersionInstalled(t *testing.T) {
	repo := &mockReleaseRepo{}
	manager := &mockVersionManager{
		installed: map[string]version.Version{
			"stable": version.New("stable", version.TypeStable, "v0.10.0", ""),
		},
	}
	install := &mockInstaller{installed: make(map[string]version.Version)}

	service, err := appversion.New(repo, manager, install, &appversion.Config{VersionsDir: "/tmp"})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if !service.IsVersionInstalled("stable") {
		t.Error("Expected stable to be installed")
	}

	if service.IsVersionInstalled("nightly") {
		t.Error("Expected nightly to NOT be installed")
	}
}

func TestService_FindStable(t *testing.T) {
	repo := &mockReleaseRepo{
		stable: release.New("v0.10.0", false, "abc123", time.Time{}, nil),
	}
	manager := &mockVersionManager{}
	install := &mockInstaller{installed: make(map[string]version.Version)}

	service, err := appversion.New(repo, manager, install, &appversion.Config{VersionsDir: "/tmp"})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	stable, err := service.FindStable(context.Background())
	if err != nil {
		t.Fatalf("FindStable failed: %v", err)
	}

	if stable.TagName() != "v0.10.0" {
		t.Errorf("Expected tag 'v0.10.0', got '%s'", stable.TagName())
	}
}

func TestService_FindNightly(t *testing.T) {
	repo := &mockReleaseRepo{
		nightly: release.New("nightly", true, "def456", time.Time{}, nil),
	}
	manager := &mockVersionManager{}
	install := &mockInstaller{installed: make(map[string]version.Version)}

	service, err := appversion.New(repo, manager, install, &appversion.Config{VersionsDir: "/tmp"})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	nightly, err := service.FindNightly(context.Background())
	if err != nil {
		t.Fatalf("FindNightly failed: %v", err)
	}

	if nightly.CommitHash() != "def456" {
		t.Errorf("Expected commit 'def456', got '%s'", nightly.CommitHash())
	}
}

func TestService_IsCommitReference(t *testing.T) {
	repo := &mockReleaseRepo{}
	manager := &mockVersionManager{}
	install := &mockInstaller{installed: make(map[string]version.Version)}

	service, err := appversion.New(repo, manager, install, &appversion.Config{VersionsDir: "/tmp"})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Valid commit references
	if !service.IsCommitReference("abc1234") {
		t.Error("Expected 'abc1234' to be a commit reference")
	}

	if !service.IsCommitReference("master") {
		t.Error("Expected 'master' to be a commit reference")
	}

	// Invalid commit references
	if service.IsCommitReference("stable") {
		t.Error("Expected 'stable' to NOT be a commit reference")
	}

	if service.IsCommitReference("v0.10.0") {
		t.Error("Expected 'v0.10.0' to NOT be a commit reference")
	}
}

func TestService_GetInstalledVersionIdentifier(t *testing.T) {
	repo := &mockReleaseRepo{}
	manager := &mockVersionManager{
		identifiers: map[string]string{
			"stable": "v0.10.0",
		},
	}
	install := &mockInstaller{installed: make(map[string]version.Version)}

	service, err := appversion.New(repo, manager, install, &appversion.Config{VersionsDir: "/tmp"})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	identifier, err := service.GetInstalledVersionIdentifier("stable")
	if err != nil {
		t.Fatalf("GetInstalledVersionIdentifier failed: %v", err)
	}

	if identifier != testVersionTag {
		t.Errorf("Expected identifier 'v0.10.0', got '%s'", identifier)
	}
}

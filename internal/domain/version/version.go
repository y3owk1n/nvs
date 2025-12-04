// Package version provides the core domain model for Neovim version management.
package version

// Version represents a Neovim version.
type Version struct {
	name        string // e.g., "v0.9.0", "stable", "nightly", "1a2b3c4"
	versionType Type
	identifier  string // Display identifier
	commitHash  string // Full commit hash if available
}

// Type represents the type of version.
type Type int

const (
	// TypeStable represents a stable release version.
	TypeStable Type = iota
	// TypeNightly represents a nightly pre-release version.
	TypeNightly
	// TypeCommit represents a specific commit hash.
	TypeCommit
	// TypeTag represents a specific version tag.
	TypeTag
)

// New creates a new Version instance.
func New(name string, versionType Type, identifier, commitHash string) Version {
	return Version{
		name:        name,
		versionType: versionType,
		identifier:  identifier,
		commitHash:  commitHash,
	}
}

// Name returns the version name.
func (v Version) Name() string {
	return v.name
}

// Type returns the version type.
func (v Version) Type() Type {
	return v.versionType
}

// Identifier returns the display identifier.
func (v Version) Identifier() string {
	return v.identifier
}

// CommitHash returns the full commit hash if available.
func (v Version) CommitHash() string {
	return v.commitHash
}

// String returns a string representation of the Type.
func (t Type) String() string {
	switch t {
	case TypeStable:
		return "stable"
	case TypeNightly:
		return "nightly"
	case TypeCommit:
		return "commit"
	case TypeTag:
		return "tag"
	default:
		return "unknown"
	}
}

// Manager handles version operations.
type Manager interface {
	// List returns all installed versions.
	List() ([]Version, error)

	// Current returns the currently active version.
	Current() (Version, error)

	// Switch activates a specific version.
	Switch(version Version) error

	// IsInstalled checks if a version is installed.
	IsInstalled(version Version) bool

	// Uninstall removes an installed version.
	Uninstall(version Version, force bool) error

	// GetInstalledReleaseIdentifier returns the release identifier (e.g. commit hash) for an installed version.
	GetInstalledReleaseIdentifier(versionName string) (string, error)
}

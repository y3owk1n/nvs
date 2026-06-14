// Package vtypes provides the core domain model for Neovim version management.
package vtypes

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidVersionName is returned when a version name is not
// safe to use as a filesystem path component. The set of rejected
// names is defined by IsValidVersionName.
var ErrInvalidVersionName = errors.New("invalid version name")

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

// IsCommitReference checks if a string looks like a commit hash or branch reference.
// Accepts "master" and "main" branches and hexadecimal strings of length 7-40.
func IsCommitReference(str string) bool {
	if str == "master" || str == "main" {
		return true
	}

	if len(str) < 7 || len(str) > 40 {
		return false
	}

	for _, r := range str {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}

	return true
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

// NormalizeVersionForPath normalizes a version string for use as a directory name.
func NormalizeVersionForPath(versionStr string) string {
	if versionStr == "stable" || versionStr == "nightly" || IsCommitReference(versionStr) {
		return versionStr
	}

	if !strings.HasPrefix(versionStr, "v") {
		return "v" + versionStr
	}

	return versionStr
}

// IsValidVersionName reports whether name is safe to use as a
// filesystem path component (i.e. a version directory name).
//
// It rejects:
//   - Empty strings
//   - The "." and ".." segment names
//   - Path separators ('/' and '\\')
//   - Any non-printable or control character (NUL, newline, etc.)
//
// The allowed character set is a conservative [a-zA-Z0-9._-]
// subset. This covers every version form nvs actually produces
// or accepts:
//   - Literal aliases: "stable", "nightly"
//   - Release tags: "v0.10.0", "v0.10.0-beta1"
//   - Branch names: "master", "main"
//   - Commit hashes: 7-40 hexadecimal characters
//
// The validator runs as the last gate before a name is joined
// onto VersionsDir and used for filesystem or exec operations
// in service.Install, service.Use, service.Uninstall,
// service.Upgrade, and cmd/run.getNvimBinaryPath. Rejecting
// pathological input here turns what would otherwise be a path
// traversal (e.g. "../etc/passwd" → VersionsDir/../etc/passwd
// after filepath.Clean) or an unintended exec of an
// attacker-controlled binary into a clean error.
func IsValidVersionName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}

	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '.' || r == '-' || r == '_':
		default:
			return false
		}
	}

	return true
}

// ValidateVersionName returns ErrInvalidVersionName wrapped with
// the offending input if name is not a safe version name (see
// IsValidVersionName). On success it returns nil.
//
// The wrapped error preserves the rejected input so logs and
// error messages tell the user exactly which character or
// segment triggered the rejection.
func ValidateVersionName(name string) error {
	if IsValidVersionName(name) {
		return nil
	}

	return fmt.Errorf("%w: %q", ErrInvalidVersionName, name)
}

// Package installer provides the domain interface for Neovim installation.
package installer

import "context"

// Installer handles Neovim installation operations.
type Installer interface {
	// InstallRelease installs a pre-built release to the destination directory.
	// Progress is reported via the progress callback function.
	InstallRelease(
		ctx context.Context,
		release ReleaseInfo,
		dest string,
		installName string,
		progress ProgressFunc,
	) error

	// BuildFromCommit builds Neovim from source at a specific commit.
	// The built version is installed to the destination directory.
	// Returns the resolved commit hash that was installed.
	BuildFromCommit(
		ctx context.Context,
		commit string,
		dest string,
		progress ProgressFunc,
	) (string, error)
}

// ProgressFunc is a callback function for reporting installation progress.
// phase describes the current operation (e.g., "Downloading", "Extracting").
// percent is the completion percentage (0-100).
type ProgressFunc func(phase string, percent int)

// ReleaseInfo provides information needed to install a release.
type ReleaseInfo interface {
	// GetAssetURL returns the download URL for the platform-specific asset.
	GetAssetURL() (string, error)

	// GetChecksumURL returns the checksum URL for verification.
	GetChecksumURL() (string, error)

	// GetIdentifier returns a unique identifier for this release.
	GetIdentifier() string
}

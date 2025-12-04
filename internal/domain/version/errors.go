package version

import "errors"

// Domain errors for version operations.
var (
	// ErrVersionNotFound is returned when a requested version is not found.
	ErrVersionNotFound = errors.New("version not found")

	// ErrVersionInUse is returned when attempting to uninstall the currently active version.
	ErrVersionInUse = errors.New("version is currently in use")

	// ErrInvalidVersion is returned when a version string has an invalid format.
	ErrInvalidVersion = errors.New("invalid version format")

	// ErrNoCurrentVersion is returned when no version is currently set as active.
	ErrNoCurrentVersion = errors.New("no current version set")
)

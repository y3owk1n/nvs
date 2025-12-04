package release

import "errors"

// Domain errors for release operations.
var (
	// ErrNoStableRelease is returned when no stable release is found.
	ErrNoStableRelease = errors.New("no stable release found")

	// ErrNoNightlyRelease is returned when no nightly release is found.
	ErrNoNightlyRelease = errors.New("no nightly release found")

	// ErrReleaseNotFound is returned when a specific release cannot be found.
	ErrReleaseNotFound = errors.New("release not found")

	// ErrNoMatchingAsset is returned when no asset matches the current platform.
	ErrNoMatchingAsset = errors.New("no matching asset found for platform")
)

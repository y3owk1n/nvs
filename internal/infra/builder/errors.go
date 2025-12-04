package builder

import "errors"

// Infrastructure errors for builder operations.
var (
	// ErrBuildFailed is returned when building from source fails.
	ErrBuildFailed = errors.New("build failed")

	// ErrCommitHashTooShort is returned when the commit hash is too short.
	ErrCommitHashTooShort = errors.New("commit hash too short")

	// ErrBinaryNotFound is returned when the built binary is not found.
	ErrBinaryNotFound = errors.New("built binary not found")
)

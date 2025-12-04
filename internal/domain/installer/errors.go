package installer

import "errors"

// Domain errors for installer operations.
var (
	// ErrDownloadFailed is returned when downloading a release fails.
	ErrDownloadFailed = errors.New("download failed")

	// ErrChecksumMismatch is returned when checksum verification fails.
	ErrChecksumMismatch = errors.New("checksum mismatch")

	// ErrExtractionFailed is returned when archive extraction fails.
	ErrExtractionFailed = errors.New("extraction failed")

	// ErrBuildFailed is returned when building from source fails.
	ErrBuildFailed = errors.New("build failed")

	// ErrInvalidCommit is returned when a commit hash is invalid.
	ErrInvalidCommit = errors.New("invalid commit hash")
)

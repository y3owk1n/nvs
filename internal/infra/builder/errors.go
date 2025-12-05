package builder

import "errors"

// Infrastructure errors for builder operations.
var (
	// ErrBuildFailed is returned when building from source fails.
	ErrBuildFailed = errors.New("build failed")

	// ErrBuildRequirementsNotMet is returned when required build tools are missing.
	ErrBuildRequirementsNotMet = errors.New("build requirements not met")

	// ErrCommitHashTooShort is returned when the commit hash is too short.
	ErrCommitHashTooShort = errors.New("commit hash too short")

	// ErrBinaryNotFound is returned when the built binary is not found.
	ErrBinaryNotFound = errors.New("built binary not found")

	// ErrStdoutPipeNotReader is returned when stdout pipe cannot be cast to io.Reader.
	ErrStdoutPipeNotReader = errors.New("stdout pipe is not a reader")

	// ErrStderrPipeNotReader is returned when stderr pipe cannot be cast to io.Reader.
	ErrStderrPipeNotReader = errors.New("stderr pipe is not a reader")
)

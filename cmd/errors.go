package cmd

import "errors"

var (
	// ErrUnsupportedShell is returned when the shell type is not supported.
	ErrUnsupportedShell = errors.New("unsupported shell type")

	// ErrRequiredDirsNotDetermined is returned when required directories cannot be determined.
	ErrRequiredDirsNotDetermined = errors.New("required directories could not be determined")

	// ErrVersionDirNotFound is returned when the version directory does not exist.
	ErrVersionDirNotFound = errors.New("version directory not found")

	// ErrNvimBinaryNotFound is returned when the nvim binary cannot be found.
	ErrNvimBinaryNotFound = errors.New("nvim binary not found")

	// ErrNvimExitNonZero is returned when nvim exits with a non-zero exit code.
	ErrNvimExitNonZero = errors.New("nvim exited with non-zero status")
)

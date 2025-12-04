package version

import "errors"

// Service errors.
var (
	// ErrCommitHashNotImplemented is returned when commit hash installation is not implemented.
	ErrCommitHashNotImplemented = errors.New(
		"commit hash installation not yet implemented in service layer",
	)
	// ErrOnlyStableNightlyUpgrade is returned when trying to upgrade non-stable/nightly versions.
	ErrOnlyStableNightlyUpgrade = errors.New("only stable and nightly versions can be upgraded")
	// ErrNotInstalled is returned when a version is not installed.
	ErrNotInstalled = errors.New("not installed")
	// ErrAlreadyUpToDate is returned when a version is already up-to-date.
	ErrAlreadyUpToDate = errors.New("already up-to-date")
)

package version

import "errors"

// Service errors.
var (
	// ErrOnlyStableNightlyUpgrade is returned when trying to upgrade non-stable/nightly versions.
	ErrOnlyStableNightlyUpgrade = errors.New("only stable and nightly versions can be upgraded")
	// ErrNotInstalled is returned when a version is not installed.
	ErrNotInstalled = errors.New("not installed")
	// ErrAlreadyUpToDate is returned when a version is already up-to-date.
	ErrAlreadyUpToDate = errors.New("already up-to-date")
	// ErrConfigNil is returned when config is nil.
	ErrConfigNil = errors.New("config cannot be nil")
	// ErrVersionsDirEmpty is returned when VersionsDir is empty.
	ErrVersionsDirEmpty = errors.New("config.VersionsDir cannot be empty")
)

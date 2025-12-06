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

	// ErrVersionFileNotFound is returned when no .nvs-version file is found.
	ErrVersionFileNotFound = errors.New("no .nvs-version file found")

	// ErrInvalidIndex is returned when an invalid index is provided.
	ErrInvalidIndex = errors.New("invalid index")

	// ErrNightlyVersionNotExists is returned when a nightly version no longer exists on disk.
	ErrNightlyVersionNotExists = errors.New("nightly version no longer exists on disk")

	// ErrInvalidUpgradeTarget is returned when an invalid upgrade target is specified.
	ErrInvalidUpgradeTarget = errors.New("upgrade can only be performed for 'stable' or 'nightly'")

	// ErrVersionNotInstalled is returned when attempting to uninstall a version that is not installed.
	ErrVersionNotInstalled = errors.New("version not installed")

	// ErrCouldNotDetectShell is returned when the shell cannot be detected.
	ErrCouldNotDetectShell = errors.New("could not detect shell")

	// ErrCouldNotResolveVersionsDir is returned when the versions directory cannot be resolved.
	ErrCouldNotResolveVersionsDir = errors.New("could not resolve versions directory")

	// ErrCouldNotDetectShellSpecify is returned when the shell cannot be detected and user needs to specify.
	ErrCouldNotDetectShellSpecify = errors.New(
		"could not detect shell, please specify: nvs hook [bash|zsh|fish]",
	)

	// ErrIssuesFound is returned when issues are found during doctor check.
	ErrIssuesFound = errors.New("issues found")

	// ErrUnknownOSArch is returned when the OS/Arch combination is unknown.
	ErrUnknownOSArch = errors.New("unknown OS/Arch")

	// ErrBinDirNotInPath is returned when the bin directory is not in PATH.
	ErrBinDirNotInPath = errors.New("bin directory not in PATH")

	// ErrMissingDependency is returned when a dependency is missing.
	ErrMissingDependency = errors.New("missing dependency")

	// ErrUnsupportedShellHook is returned when the shell is unsupported.
	ErrUnsupportedShellHook = errors.New("unsupported shell")

	// ErrMutuallyExclusiveFlags is returned when --source and --json flags are both provided.
	ErrMutuallyExclusiveFlags = errors.New("--source and --json flags are mutually exclusive")
)

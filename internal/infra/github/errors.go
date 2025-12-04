package github

import "errors"

// Infrastructure errors for GitHub operations.
var (
	// ErrRateLimitExceeded is returned when GitHub API rate limit is exceeded.
	ErrRateLimitExceeded = errors.New("GitHub API rate limit exceeded")

	// ErrAPIRequestFailed is returned when an API request fails.
	ErrAPIRequestFailed = errors.New("API request failed")

	// ErrUnsupportedArch is returned when the architecture is not supported.
	ErrUnsupportedArch = errors.New("unsupported architecture")

	// ErrUnsupportedOS is returned when the operating system is not supported.
	ErrUnsupportedOS = errors.New("unsupported operating system")
)

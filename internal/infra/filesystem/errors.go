package filesystem

import "errors"

// Infrastructure errors for filesystem operations.
var (
	// ErrBinaryNotFound is returned when the Neovim binary cannot be found.
	ErrBinaryNotFound = errors.New("neovim binary not found")
)

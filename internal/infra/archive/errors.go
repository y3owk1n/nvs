package archive

import "errors"

// Infrastructure errors for archive operations.
var (
	// ErrUnsupportedFormat is returned when the archive format is not supported.
	ErrUnsupportedFormat = errors.New("unsupported archive format")

	// ErrEmptyFile is returned when the file is empty.
	ErrEmptyFile = errors.New("empty file")

	// ErrUnknownFileType is returned when the file type cannot be determined.
	ErrUnknownFileType = errors.New("unknown file type")
)

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

// IllegalPathError is returned when an archive contains an illegal file path.
type IllegalPathError struct {
	Path string
}

func (e *IllegalPathError) Error() string {
	return "illegal file path in archive: " + e.Path
}

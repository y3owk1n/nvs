package downloader

import "errors"

// Infrastructure errors for downloader operations.
var (
	// ErrDownloadFailed is returned when a download fails.
	ErrDownloadFailed = errors.New("download failed")

	// ErrChecksumDownloadFailed is returned when downloading a checksum file fails.
	ErrChecksumDownloadFailed = errors.New("checksum download failed")

	// ErrChecksumFileEmpty is returned when a checksum file is empty.
	ErrChecksumFileEmpty = errors.New("checksum file is empty")

	// ErrChecksumMismatch is returned when checksum verification fails.
	ErrChecksumMismatch = errors.New("checksum mismatch")
)

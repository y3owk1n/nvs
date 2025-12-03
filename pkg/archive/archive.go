// Package archive provides functions for extracting archives.
package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/h2non/filetype"
	"github.com/sirupsen/logrus"
)

// Constants for archive operations.
const (
	BufSize = 262
	DirPerm = 0o755
	Zip     = "zip"
	TarGz   = "tar.gz"
)

// Errors for archive operations.
var (
	ErrUnsupportedArchiveFormat = errors.New("unsupported archive format")
	ErrFileTypeMatching         = errors.New("file type matching error")
	ErrUnknownFileType          = errors.New("unknown file type")
)

// ExtractArchive extracts the contents of an archive (tar.gz or zip) from the provided source file
// and writes them to the specified destination directory. It returns an error if extraction fails.
//
// Example usage:
//
//	src, err := os.Open("path/to/archive.tar.gz")
//	if err != nil {
//	    // handle error
//	}
//	defer src.Close()
//
//	dest := "path/to/destination"
//	if err := ExtractArchive(src, dest); err != nil {
//	    // handle extraction error
//	}
func ExtractArchive(src *os.File, dest string) error {
	// Ensure we start reading from the beginning of the file.
	_, err := src.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek to start of file: %w", err)
	}
	// Detect the archive format.
	format, err := DetectArchiveFormat(src)
	if err != nil {
		return fmt.Errorf("archive detection failed: %w", err)
	}

	logrus.Debugf("Detected archive format: %s", format)
	// Dispatch to the correct extraction function based on the format.
	switch format {
	case TarGz:
		return ExtractTarGz(src, dest)
	case Zip:
		return ExtractZip(src, dest)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedArchiveFormat, format)
	}
}

// DetectArchiveFormat reads the header of the file to determine its archive format.
// It supports tar.gz and zip formats and returns the format as a string, or an error if detection fails.
//
// Example usage:
//
//	src, _ := os.Open("path/to/archive.zip")
//	format, err := DetectArchiveFormat(src)
//	if err != nil {
//	    // handle error
//	}
//	fmt.Println("Detected format:", format)
func DetectArchiveFormat(file *os.File) (string, error) {
	// Read a chunk of the file for type detection.
	buf := make([]byte, BufSize)

	numBytes, err := file.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("failed to read file for type detection: %w", err)
	}

	if numBytes == 0 {
		return "", fmt.Errorf("%w: empty buffer", ErrFileTypeMatching)
	}
	// Reset file pointer to the beginning.
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return "", fmt.Errorf("failed to seek file: %w", err)
	}
	// Use the filetype package to detect the file format.
	kind, err := filetype.Match(buf[:numBytes])
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFileTypeMatching, err)
	}

	if kind == filetype.Unknown {
		return "", ErrUnknownFileType
	}
	// Map detected extension to supported archive format.
	if kind.Extension == "zip" {
		return "zip", nil
	}

	if kind.Extension == "gz" {
		return "tar.gz", nil
	}

	return "", fmt.Errorf("%w: %s", ErrUnsupportedArchiveFormat, kind.Extension)
}

// ExtractTarGz extracts a tar.gz archive from the provided source file into the destination directory.
// It handles gzip decompression and tar extraction, creating directories and files as needed.
//
// Example usage:
//
//	src, _ := os.Open("path/to/archive.tar.gz")
//	err := ExtractTarGz(src, "path/to/destination")
//	if err != nil {
//	    // handle error
//	}
func ExtractTarGz(src *os.File, dest string) error {
	// Create a gzip reader to decompress the file.
	gzr, err := gzip.NewReader(src)
	// Return err
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() {
		err := gzr.Close()
		if err != nil {
			logrus.Errorf("warning: failed to close gzip reader: %v", err)
		}
	}()
	// Create a tar reader to read the tar archive.
	tarReader := tar.NewReader(gzr)
	// Iterate over all files in the archive.
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			// End of archive.
			break
		}

		if err != nil {
			return fmt.Errorf("error reading tar archive: %w", err)
		}
		// Build the target file/directory path.
		target := filepath.Join(dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory if it doesn't exist.
			err := os.MkdirAll(target, DirPerm)
			if err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}
		case tar.TypeReg:
			// Ensure the directory for the file exists.
			err = os.MkdirAll(filepath.Dir(target), DirPerm)
			if err != nil {
				return fmt.Errorf("failed to create directory for file %s: %w", target, err)
			}
			// Create the file.
			file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			// Return err
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", target, err)
			}
			defer func() {
				err := file.Close()
				if err != nil {
					logrus.Errorf("warning: failed to close file %s: %v", target, err)
				}
			}()

			// Copy file content.
			_, err = io.Copy(file, tarReader)
			if err != nil {
				return fmt.Errorf("failed to copy file content to %s: %w", target, err)
			}
		}
	}

	return nil
}

// ExtractZip extracts a zip archive from the provided source file into the destination directory.
// It returns an error if any file within the archive cannot be extracted properly.
//
// Example usage:
//
//	src, _ := os.Open("path/to/archive.zip")
//	err := ExtractZip(src, "path/to/destination")
//	if err != nil {
//	    // handle error
//	}
func ExtractZip(src *os.File, dest string) error {
	// Retrieve file info to get the size for zip.NewReader.
	info, err := src.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	// Create a zip reader for the file.
	r, err := zip.NewReader(src, info.Size())
	if err != nil {
		return fmt.Errorf("failed to create zip reader: %w", err)
	}
	// Iterate over each file in the zip archive.
	for _, fileEntry := range r.File {
		path := filepath.Join(dest, fileEntry.Name)
		if fileEntry.FileInfo().IsDir() {
			// Create directory if necessary.
			err := os.MkdirAll(path, fileEntry.Mode())
			if err != nil {
				return fmt.Errorf("failed to create directory %s: %w", path, err)
			}

			continue
		}
		// Ensure the file's directory exists.
		err = os.MkdirAll(filepath.Dir(path), DirPerm)
		if err != nil {
			return fmt.Errorf("failed to create directory for file %s: %w", path, err)
		}
		// Open the file inside the zip archive.
		readerCloser, err := fileEntry.Open()
		// Return err
		if err != nil {
			return fmt.Errorf("failed to open file %s in zip: %w", fileEntry.Name, err)
		}
		defer func() {
			err := readerCloser.Close()
			if err != nil {
				logrus.Errorf("warning: failed to close file %s in zip: %v", fileEntry.Name, err)
			}
		}()
		// Create the output file.
		out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileEntry.Mode())
		// Return err
		if err != nil {
			return fmt.Errorf("failed to create output file %s: %w", path, err)
		}
		defer func() {
			err := out.Close()
			if err != nil {
				logrus.Errorf("warning: failed to close output file %s: %v", path, err)
			}
		}()
		// Copy the file contents.
		_, err = io.Copy(out, readerCloser)
		if err != nil {
			return fmt.Errorf("failed to copy file %s: %w", path, err)
		}
	}

	return nil
}

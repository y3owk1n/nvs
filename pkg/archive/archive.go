package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/h2non/filetype"
	"github.com/sirupsen/logrus"
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
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to start of file: %w", err)
	}
	// Detect the archive format.
	format, err := detectArchiveFormat(src)
	if err != nil {
		return fmt.Errorf("archive detection failed: %w", err)
	}
	logrus.Debugf("Detected archive format: %s", format)
	// Dispatch to the correct extraction function based on the format.
	switch format {
	case "tar.gz":
		return extractTarGz(src, dest)
	case "zip":
		return extractZip(src, dest)
	default:
		return fmt.Errorf("unsupported archive format: %s", format)
	}
}

// detectArchiveFormat reads the header of the file to determine its archive format.
// It supports tar.gz and zip formats and returns the format as a string, or an error if detection fails.
//
// Example usage:
//
//	src, _ := os.Open("path/to/archive.zip")
//	format, err := detectArchiveFormat(src)
//	if err != nil {
//	    // handle error
//	}
//	fmt.Println("Detected format:", format)
func detectArchiveFormat(f *os.File) (string, error) {
	// Read a chunk of the file for type detection.
	buf := make([]byte, 262)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read file for type detection: %w", err)
	}
	if n == 0 {
		return "", fmt.Errorf("file type matching error: empty buffer")
	}
	// Reset file pointer to the beginning.
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to seek file: %w", err)
	}
	// Use the filetype package to detect the file format.
	kind, err := filetype.Match(buf[:n])
	if err != nil {
		return "", fmt.Errorf("file type matching error: %w", err)
	}
	if kind == filetype.Unknown {
		return "", fmt.Errorf("unknown file type")
	}
	// Map detected extension to supported archive format.
	if kind.Extension == "zip" {
		return "zip", nil
	}
	if kind.Extension == "gz" {
		return "tar.gz", nil
	}
	return "", fmt.Errorf("unsupported archive format: %s", kind.Extension)
}

// extractTarGz extracts a tar.gz archive from the provided source file into the destination directory.
// It returns an error if extraction fails at any step.
//
// Example usage:
//
//	src, _ := os.Open("path/to/archive.tar.gz")
//	err := extractTarGz(src, "path/to/destination")
//	if err != nil {
//	    // handle error
//	}
func extractTarGz(src *os.File, dest string) error {
	// Create a gzip reader to decompress the file.
	gzr, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() {
		if err := gzr.Close(); err != nil {
			logrus.Errorf("warning: failed to close gzip reader: %v", err)
		}
	}()
	// Create a tar reader to read the tar archive.
	tr := tar.NewReader(gzr)
	// Iterate over all files in the archive.
	for {
		header, err := tr.Next()
		if err == io.EOF {
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
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}
		case tar.TypeReg:
			// Ensure the directory for the file exists.
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create directory for file %s: %w", target, err)
			}
			// Create the file.
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", target, err)
			}
			defer func() {
				if err := f.Close(); err != nil {
					logrus.Errorf("warning: failed to close file %s: %v", target, err)
				}
			}()

			// Copy file content.
			if _, err := io.Copy(f, tr); err != nil {
				return fmt.Errorf("failed to copy file content to %s: %w", target, err)
			}
		}
	}
	return nil
}

// extractZip extracts a zip archive from the provided source file into the destination directory.
// It returns an error if any file within the archive cannot be extracted properly.
//
// Example usage:
//
//	src, _ := os.Open("path/to/archive.zip")
//	err := extractZip(src, "path/to/destination")
//	if err != nil {
//	    // handle error
//	}
func extractZip(src *os.File, dest string) error {
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
	for _, f := range r.File {
		path := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			// Create directory if necessary.
			if err := os.MkdirAll(path, f.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", path, err)
			}
			continue
		}
		// Ensure the file's directory exists.
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("failed to create directory for file %s: %w", path, err)
		}
		// Open the file inside the zip archive.
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file %s in zip: %w", f.Name, err)
		}
		defer func() {
			if err := rc.Close(); err != nil {
				logrus.Errorf("warning: failed to close file %s in zip: %v", f.Name, err)
			}
		}()
		// Create the output file.
		out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("failed to create output file %s: %w", path, err)
		}
		defer func() {
			if err := out.Close(); err != nil {
				logrus.Errorf("warning: failed to close output file %s: %v", path, err)
			}
		}()
		// Copy the file contents.
		if _, err := io.Copy(out, rc); err != nil {
			return fmt.Errorf("failed to copy file %s: %w", path, err)
		}
	}
	return nil
}

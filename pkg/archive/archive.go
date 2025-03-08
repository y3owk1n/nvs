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

func ExtractArchive(src *os.File, dest string) error {
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to start of file: %w", err)
	}
	format, err := detectArchiveFormat(src)
	if err != nil {
		return fmt.Errorf("archive detection failed: %w", err)
	}
	logrus.Debugf("Detected archive format: %s", format)
	switch format {
	case "tar.gz":
		return extractTarGz(src, dest)
	case "zip":
		return extractZip(src, dest)
	default:
		return fmt.Errorf("unsupported archive format: %s", format)
	}
}

func detectArchiveFormat(f *os.File) (string, error) {
	buf := make([]byte, 262)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read file for type detection: %w", err)
	}
	if n == 0 {
		return "", fmt.Errorf("file type matching error: empty buffer")
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to seek file: %w", err)
	}
	kind, err := filetype.Match(buf[:n])
	if err != nil {
		return "", fmt.Errorf("file type matching error: %w", err)
	}
	if kind == filetype.Unknown {
		return "", fmt.Errorf("unknown file type")
	}
	if kind.Extension == "zip" {
		return "zip", nil
	}
	if kind.Extension == "gz" {
		return "tar.gz", nil
	}
	return "", fmt.Errorf("unsupported archive format: %s", kind.Extension)
}

func extractTarGz(src *os.File, dest string) error {
	gzr, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tar archive: %w", err)
		}
		target := filepath.Join(dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create directory for file %s: %w", target, err)
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", target, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("failed to copy file content to %s: %w", target, err)
			}
			f.Close()
		}
	}
	return nil
}

func extractZip(src *os.File, dest string) error {
	info, err := src.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	r, err := zip.NewReader(src, info.Size())
	if err != nil {
		return fmt.Errorf("failed to create zip reader: %w", err)
	}
	for _, f := range r.File {
		path := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, f.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", path, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("failed to create directory for file %s: %w", path, err)
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file %s in zip: %w", f.Name, err)
		}
		out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create output file %s: %w", path, err)
		}
		if _, err := io.Copy(out, rc); err != nil {
			rc.Close()
			out.Close()
			return fmt.Errorf("failed to copy file %s: %w", path, err)
		}
		rc.Close()
		out.Close()
	}
	return nil
}

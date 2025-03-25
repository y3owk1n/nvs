package installer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/pkg/archive"
	"github.com/y3owk1n/nvs/pkg/releases"
	"github.com/y3owk1n/nvs/pkg/utils"
)

var client = &http.Client{Timeout: 15 * time.Second}

// progressReader wraps an io.Reader to report progress via a callback.
type progressReader struct {
	reader   io.Reader
	total    int64
	current  int64
	callback func(progress int)
}

// Read implements the io.Reader interface and updates the progress callback.
func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.current += int64(n)
	if pr.total > 0 && pr.callback != nil {
		percent := int(float64(pr.current) / float64(pr.total) * 100)
		pr.callback(percent)
	}
	return n, err
}

// InstallVersion resolves a release version from the given alias, downloads the asset,
// verifies its checksum, extracts the archive, and writes a version file.
// It accepts a context for cancellation and timeout control.
//
// Example usage:
//
//	ctx := context.Background()
//	alias := "stable"  // or "nightly" or a specific version tag
//	versionsDir := "/path/to/installations"
//	cacheFilePath := "/path/to/cache/file"
//	if err := InstallVersion(ctx, alias, versionsDir, cacheFilePath); err != nil {
//	    // handle error
//	}
func InstallVersion(ctx context.Context, alias string, versionsDir string, cacheFilePath string) error {
	release, err := releases.ResolveVersion(alias, cacheFilePath)
	if err != nil {
		return fmt.Errorf("error resolving version: %v", err)
	}

	logrus.Debugf("Resolved release: %+v", release)

	installName := alias
	if alias != "stable" && alias != "nightly" {
		installName = release.TagName
	}

	logrus.Debugf("Determined install name: %s", installName)

	if utils.IsInstalled(versionsDir, installName) {
		logrus.Debugf("Version %s is already installed, skipping installation", installName)
		fmt.Printf("%s %s\n", utils.WarningIcon(), utils.WhiteText(fmt.Sprintf("Version %s is already installed.", utils.CyanText(installName))))
		os.Exit(0)
	}

	assetURL, assetPattern, err := releases.GetAssetURL(release)
	if err != nil {
		return fmt.Errorf("error getting asset URL: %v", err)
	}

	logrus.Debugf("Resolved asset URL: %s, asset pattern: %s", assetURL, assetPattern)

	checksumURL, err := releases.GetChecksumURL(release, assetPattern)
	if err != nil {
		return fmt.Errorf("error getting checksum URL: %v", err)
	}

	logrus.Debugf("Resolved checksum URL: %s", checksumURL)

	releaseIdentifier := releases.GetReleaseIdentifier(release, alias)
	logrus.Debugf("Determined release identifier: %s", releaseIdentifier)
	fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText(fmt.Sprintf("Installing Neovim %s...", utils.CyanText(alias))))

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " 0%"
	s.Start()

	err = DownloadAndInstall(
		ctx,
		versionsDir,
		installName,
		assetURL,
		checksumURL,
		releaseIdentifier,
		func(progress int) {
			logrus.Debugf("Download progress: %d%%", progress)
			s.Suffix = fmt.Sprintf(" %d%%", progress)
		},
		func(phase string) {
			logrus.Debugf("Installation phase: %s", phase)
			s.Prefix = phase + " "
			s.Suffix = ""
		},
	)
	s.Stop()
	if err != nil {
		return fmt.Errorf("installation failed: %v", err)
	}
	logrus.Debug("Installation successful")
	fmt.Printf("%s %s\n", utils.SuccessIcon(), utils.CyanText("Installation successful!"))
	return nil
}

// DownloadAndInstall downloads the asset from assetURL to a temporary file,
// verifies the checksum if available, extracts the archive into versionsDir,
// and writes the releaseIdentifier to a version file.
// It accepts callbacks to update progress and phase.
//
// Example usage:
//
//	ctx := context.Background()
//	versionsDir := "/path/to/installations"
//	installName := "v0.5.0"
//	assetURL := "https://example.com/neovim.archive"
//	checksumURL := "https://example.com/neovim.archive.sha256"
//	releaseIdentifier := "abcdef1234567890"
//	err := DownloadAndInstall(
//	    ctx,
//	    versionsDir,
//	    installName,
//	    assetURL,
//	    checksumURL,
//	    releaseIdentifier,
//	    func(progress int) {
//	        fmt.Printf("Download progress: %d%%\n", progress)
//	    },
//	    func(phase string) {
//	        fmt.Printf("Phase: %s\n", phase)
//	    },
//	)
//	if err != nil {
//	    // handle error
//	}
func DownloadAndInstall(ctx context.Context, versionsDir, installName, assetURL, checksumURL, releaseIdentifier string, progressCallback func(progress int), phaseCallback func(phase string)) error {
	tmpFile, err := os.CreateTemp("", "nvim-*.archive")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if phaseCallback != nil {
		phaseCallback("Downloading asset...")
	}

	if err := downloadFile(ctx, assetURL, tmpFile, progressCallback); err != nil {
		return fmt.Errorf("download error: %w", err)
	}

	if checksumURL != "" {
		if phaseCallback != nil {
			phaseCallback("Verifying checksum...")
		}

		logrus.Debug("Verifying checksum...")
		if err := verifyChecksum(ctx, tmpFile, checksumURL); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
		logrus.Debug("Checksum verified successfully")
	}

	if phaseCallback != nil {
		phaseCallback("Extracting Archive...")
	}

	versionDir := filepath.Join(versionsDir, installName)
	if err := archive.ExtractArchive(tmpFile, versionDir); err != nil {
		return fmt.Errorf("extraction error: %w", err)
	}

	if phaseCallback != nil {
		phaseCallback("Writing version file...")
	}

	versionFile := filepath.Join(versionDir, "version.txt")
	if err := os.WriteFile(versionFile, []byte(releaseIdentifier), 0644); err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}
	return nil
}

// downloadFile downloads the content from the given URL and writes it to dest.
// It uses a progressReader to report progress via the callback.
//
// Example usage:
//
//	ctx := context.Background()
//	dest, _ := os.Create("downloaded.archive")
//	defer dest.Close()
//	err := downloadFile(ctx, "https://example.com/neovim.archive", dest, func(progress int) {
//	    fmt.Printf("Progress: %d%%\n", progress)
//	})
//	if err != nil {
//	    // handle error
//	}
func downloadFile(ctx context.Context, url string, dest *os.File, callback func(progress int)) error {
	logrus.Debugf("Downloading asset from URL: %s", url)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	total := resp.ContentLength
	pr := &progressReader{
		reader:   resp.Body,
		total:    total,
		callback: callback,
	}

	if _, err := io.Copy(dest, pr); err != nil {
		return fmt.Errorf("failed to copy download content: %w", err)
	}
	return nil
}

// verifyChecksum downloads the expected checksum from checksumURL,
// computes the SHA256 checksum of the provided file, and compares them.
// The file pointer is reset after the computation.
//
// Example usage:
//
//	ctx := context.Background()
//	file, _ := os.Open("downloaded.archive")
//	defer file.Close()
//	err := verifyChecksum(ctx, file, "https://example.com/neovim.archive.sha256")
//	if err != nil {
//	    // handle error
//	}
func verifyChecksum(ctx context.Context, file *os.File, checksumURL string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", checksumURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create checksum request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download checksum file: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("checksum download failed with status %d", resp.StatusCode)
	}
	checksumData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read checksum data: %w", err)
	}
	expectedFields := strings.Fields(string(checksumData))
	if len(expectedFields) == 0 {
		return fmt.Errorf("checksum file is empty")
	}
	expectedHash := expectedFields[0]
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek file for checksum computation: %w", err)
	}
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to reset file pointer: %w", err)
	}
	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}
	return nil
}

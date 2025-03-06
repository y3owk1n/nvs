package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/h2non/filetype"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	versionsDir   string
	cacheFilePath string
	globalBinDir  string
	verbose       bool
	// Global context for cancellation.
	ctx, cancel = context.WithCancel(context.Background())
	// Global HTTP client with a timeout.
	client = &http.Client{
		Timeout: 15 * time.Second,
	}
	// Cache TTL for remote releases.
	releasesCacheTTL = 5 * time.Minute
)

var Version = "v0.0.0"

// Release represents a GitHub release.
type Release struct {
	TagName     string  `json:"tag_name"`
	Prerelease  bool    `json:"prerelease"`
	Assets      []Asset `json:"assets"`
	PublishedAt string  `json:"published_at"`
}

// Asset represents an asset in a release.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// progressWriter wraps an io.Writer to display download progress.
type progressWriter struct {
	total   int64
	current int64
	writer  io.Writer
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	pw.current += int64(n)
	percent := float64(pw.current) / float64(pw.total) * 100
	fmt.Printf("\rDownloading... %.2f%% complete", percent)
	return n, err
}

func init() {
	// Setup cancellation on OS interrupt signals.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		logrus.Info("Interrupt received, canceling operations...")
		cancel()
		os.Exit(1)
	}()

	// Initialize logger formatting.
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	// Determine the user's home directory.
	home, err := os.UserHomeDir()
	if err != nil {
		logrus.Fatalf("Failed to get user home directory: %v", err)
	}
	// Base configuration directory.
	baseDir := filepath.Join(home, ".nvs")
	// Versions are stored under ~/.nvs/versions.
	versionsDir = filepath.Join(baseDir, "versions")
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		logrus.Fatalf("Failed to create versions directory: %v", err)
	}

	// Global binary directory that will contain the symlink "nvim".
	globalBinDir = filepath.Join(baseDir, "bin")
	if err := os.MkdirAll(globalBinDir, 0755); err != nil {
		logrus.Fatalf("Failed to create global bin directory: %v", err)
	}

	// Create a cache directory to store remote releases.
	cacheDir := filepath.Join(filepath.Dir(versionsDir), "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		logrus.Fatalf("Failed to create cache directory: %v", err)
	}
	cacheFilePath = filepath.Join(cacheDir, "releases.json")
}

func main() {
	// Set up Cobra's persistent flags.
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	cobra.OnInitialize(func() {
		if verbose {
			logrus.SetLevel(logrus.DebugLevel)
		} else {
			logrus.SetLevel(logrus.InfoLevel)
		}
	})

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// -------------------- Cobra Commands Setup --------------------

var rootCmd = &cobra.Command{
	Use:     "nvs",
	Short:   "Neovim version switcher",
	Long:    "A CLI tool to install, switch, list, uninstall, and reset Neovim versions.",
	Version: Version,
}

var installCmd = &cobra.Command{
	Use:   "install <version|stable|nightly>",
	Short: "Install a Neovim version",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		versionArg := args[0]
		release, err := resolveVersion(versionArg)
		if err != nil {
			logrus.Fatalf("Error resolving version: %v", err)
		}

		if isInstalled(release.TagName) {
			fmt.Printf("Version %s is already installed\n", release.TagName)
			return
		}

		assetURL, assetPattern, err := getAssetURL(release)
		if err != nil {
			logrus.Fatalf("Error getting asset URL: %v", err)
		}
		logrus.Debugf("Asset URL: %s, Pattern: %s", assetURL, assetPattern)

		checksumURL, err := getChecksumURL(release, assetPattern)
		if err != nil {
			logrus.Fatalf("Error getting checksum URL: %v", err)
		}

		fmt.Printf("Installing Neovim %s...\n", release.TagName)
		if err := downloadAndInstall(release.TagName, assetURL, checksumURL); err != nil {
			logrus.Fatalf("Installation failed: %v", err)
		}
		fmt.Println("\nInstallation successful!")
	},
}

var useCmd = &cobra.Command{
	Use:   "use <version|stable|nightly>",
	Short: "Switch to a specific version",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		versionArg := args[0]
		targetVersion := versionArg
		if versionArg == "stable" || versionArg == "nightly" {
			release, err := resolveVersion(versionArg)
			if err != nil {
				logrus.Fatalf("Error resolving version: %v", err)
			}
			targetVersion = release.TagName
		}

		if !isInstalled(targetVersion) {
			logrus.Fatalf("Version %s is not installed", targetVersion)
		}

		// Update the "current" symlink inside the versions directory.
		symlinkPath := filepath.Join(versionsDir, "current")
		versionPath := filepath.Join(versionsDir, targetVersion)
		if err := updateSymlink(versionPath, symlinkPath); err != nil {
			logrus.Fatalf("Failed to switch version: %v", err)
		}

		fmt.Printf("Switched to Neovim %s\n", targetVersion)
		// Find the Neovim executable.
		nvimExec := findNvimBinary(versionPath)
		if nvimExec == "" {
			fmt.Printf("Warning: Could not find Neovim binary in %s. Please check the installation structure.\n", versionPath)
			return
		}

		// Create (or update) a global symlink in the global bin directory.
		targetBin := filepath.Join(globalBinDir, "nvim")
		if _, err := os.Lstat(targetBin); err == nil {
			os.Remove(targetBin)
		}
		if err := os.Symlink(nvimExec, targetBin); err != nil {
			logrus.Fatalf("Failed to create symlink in global bin: %v", err)
		}
		fmt.Printf("Global Neovim binary updated: %s -> %s\n", targetBin, nvimExec)

		// Only show the PATH message if the global bin directory is not already in PATH.
		pathEnv := os.Getenv("PATH")
		if !strings.Contains(pathEnv, globalBinDir) {
			fmt.Printf("Add this to your PATH: %s\n", globalBinDir)
		}
	},
}

// findNvimBinary searches under dir for an executable named "nvim" or starting with "nvim-".
func findNvimBinary(dir string) string {
	var binaryPath string
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip error
		}
		if !d.IsDir() && (d.Name() == "nvim" || strings.HasPrefix(d.Name(), "nvim-")) {
			info, err := d.Info()
			if err == nil && info.Mode()&0111 != 0 {
				binaryPath = path
				return io.EOF // break early
			}
		}
		return nil
	})
	return binaryPath
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed versions",
	Run: func(cmd *cobra.Command, args []string) {
		versions, err := listInstalledVersions()
		if err != nil {
			logrus.Fatalf("Error listing versions: %v", err)
		}
		for _, v := range versions {
			fmt.Println(v)
		}
	},
}

// listRemoteCmd lists remote releases with enhanced info.
// For stable releases it shows the actual version (if resolved) and for nightly it shows commit or published date.
var listRemoteCmd = &cobra.Command{
	Use:   "list-remote [force]",
	Short: "List available remote versions (cached for 5 minutes)",
	Run: func(cmd *cobra.Command, args []string) {
		force := len(args) > 0 && args[0] == "force"
		releases, err := getCachedReleases(force)
		if err != nil {
			logrus.Fatalf("Error fetching releases: %v", err)
		}

		stableRelease, err := findLatestStable()
		stableTag := ""
		if err == nil {
			stableTag = stableRelease.TagName
		} else {
			stableTag = "stable"
		}

		for _, r := range releases {
			if r.Prerelease {
				var commit string
				if strings.HasPrefix(r.TagName, "nightly-") {
					commit = strings.TrimPrefix(r.TagName, "nightly-")
				} else if r.TagName == "nightly" {
					commit = "published on " + r.PublishedAt
				} else {
					commit = r.TagName
				}
				fmt.Printf("%-10s (nightly commit: %s)\n", r.TagName, commit)
			} else {
				tagToShow := r.TagName
				if r.TagName == "stable" {
					tagToShow = stableTag
				}
				fmt.Printf("%-10s (stable version: %s)\n", r.TagName, tagToShow)
			}
		}
	},
}

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current active version",
	Run: func(cmd *cobra.Command, args []string) {
		current, err := getCurrentVersion()
		if err != nil {
			logrus.Fatalf("Error getting current version: %v", err)
		}
		fmt.Println(current)
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <version>",
	Short: "Uninstall a specific version",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		versionArg := args[0]
		versionPath := filepath.Join(versionsDir, versionArg)
		if !isInstalled(versionArg) {
			logrus.Fatalf("Version %s is not installed", versionArg)
		}
		if err := os.RemoveAll(versionPath); err != nil {
			logrus.Fatalf("Failed to uninstall version %s: %v", versionArg, err)
		}
		fmt.Printf("Uninstalled version %s\n", versionArg)
	},
}

// resetCmd deletes the entire ~/.nvs directory (removing symlinks, versions, cache, etc.).
var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset all data (remove symlinks, downloaded versions, cache, etc.)",
	Long:  "WARNING: This command will delete the entire ~/.nvs directory and all its contents. Use with caution.",
	Run: func(cmd *cobra.Command, args []string) {
		home, err := os.UserHomeDir()
		if err != nil {
			logrus.Fatalf("Failed to get home directory: %v", err)
		}
		baseDir := filepath.Join(home, ".nvs")
		fmt.Printf("WARNING: This will delete all data in %s. Are you sure? (y/N): ", baseDir)
		var answer string
		fmt.Scanln(&answer)
		if strings.ToLower(answer) != "y" {
			fmt.Println("Reset cancelled.")
			return
		}
		if err := os.RemoveAll(baseDir); err != nil {
			logrus.Fatalf("Failed to reset data: %v", err)
		}
		logrus.Info("Reset successful. All data has been cleared.")
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(useCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(listRemoteCmd)
	rootCmd.AddCommand(currentCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(resetCmd)
}

// -------------------- Caching Functions --------------------

// getCachedReleases returns cached remote releases if fresh (unless forced),
// otherwise fetches fresh data from GitHub.
func getCachedReleases(force bool) ([]Release, error) {
	if !force {
		if info, err := os.Stat(cacheFilePath); err == nil {
			if time.Since(info.ModTime()) < releasesCacheTTL {
				data, err := os.ReadFile(cacheFilePath)
				if err == nil {
					var releases []Release
					if err = json.Unmarshal(data, &releases); err == nil {
						logrus.Info("Using cached releases")
						return releases, nil
					}
				}
			}
		}
	}
	logrus.Info("Fetching fresh releases from GitHub")
	releases, err := getReleases()
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(releases)
	if err == nil {
		os.WriteFile(cacheFilePath, data, 0644)
	}
	return releases, nil
}

// -------------------- Release Handling & Download/Extraction --------------------

func resolveVersion(version string) (Release, error) {
	switch version {
	case "stable":
		return findLatestStable()
	case "nightly":
		return findLatestNightly()
	default:
		return findSpecificVersion(version)
	}
}

func getReleases() ([]Release, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/repos/neovim/neovim/releases", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "nvs")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 403 {
		body, _ := io.ReadAll(resp.Body)
		if strings.Contains(string(body), "rate limit") {
			return nil, fmt.Errorf("GitHub API rate limit exceeded. Please try again later.")
		}
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return releases, nil
}

func findLatestStable() (Release, error) {
	releases, err := getReleases()
	if err != nil {
		return Release{}, err
	}
	for _, r := range releases {
		if !r.Prerelease {
			return r, nil
		}
	}
	return Release{}, fmt.Errorf("no stable release found")
}

func findLatestNightly() (Release, error) {
	releases, err := getReleases()
	if err != nil {
		return Release{}, err
	}
	for _, r := range releases {
		if r.Prerelease {
			return r, nil
		}
	}
	return Release{}, fmt.Errorf("no nightly release found")
}

func findSpecificVersion(version string) (Release, error) {
	releases, err := getReleases()
	if err != nil {
		return Release{}, err
	}
	for _, r := range releases {
		if r.TagName == version {
			return r, nil
		}
	}
	return Release{}, fmt.Errorf("version %s not found", version)
}

// getAssetURL selects the correct asset URL based on OS and architecture.
// For darwin/arm64 it tries multiple patterns including tar.gz and zip.
func getAssetURL(release Release) (string, string, error) {
	var patterns []string
	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			patterns = []string{"linux64.tar.gz"}
		case "arm64":
			patterns = []string{"linux-arm64.tar.gz"}
		default:
			return "", "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			patterns = []string{"macos-arm64.tar.gz", "macos.tar.gz", "macos-arm64.zip", "macos.zip"}
		} else {
			patterns = []string{"macos.tar.gz", "macos.zip"}
		}
	case "windows":
		patterns = []string{"win64.zip"}
	default:
		return "", "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	for _, asset := range release.Assets {
		for _, pattern := range patterns {
			if strings.Contains(asset.Name, pattern) {
				return asset.BrowserDownloadURL, pattern, nil
			}
		}
	}
	return "", "", fmt.Errorf("no matching asset found for %s/%s", runtime.GOOS, runtime.GOARCH)
}

func getChecksumURL(release Release, assetPattern string) (string, error) {
	checksumPattern := assetPattern + ".sha256"
	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, checksumPattern) {
			return asset.BrowserDownloadURL, nil
		}
	}
	return "", nil
}

func downloadAndInstall(version, assetURL, checksumURL string) error {
	tmpFile, err := os.CreateTemp("", "nvim-*.archive")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if err := downloadFile(assetURL, tmpFile); err != nil {
		return fmt.Errorf("download error: %w", err)
	}

	if checksumURL != "" {
		logrus.Info("Verifying checksum...")
		if err := verifyChecksum(tmpFile, checksumURL); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
		logrus.Info("Checksum verified successfully")
	}

	versionDir := filepath.Join(versionsDir, version)
	if err := extractArchive(tmpFile, versionDir); err != nil {
		return fmt.Errorf("extraction error: %w", err)
	}
	return nil
}

func downloadFile(url string, dest *os.File) error {
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
	logrus.Debugf("Response status: %d, Content-Length: %d", resp.StatusCode, resp.ContentLength)
	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}
	n, err := io.Copy(dest, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy download content: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("downloaded file is empty; check asset URL: %s", url)
	}
	return nil
}

func detectArchiveFormat(f *os.File) (string, error) {
	buf := make([]byte, 262)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read file for type detection: %w", err)
	}
	if n == 0 {
		return "", fmt.Errorf("file type matching error: Empty buffer")
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

func extractArchive(src *os.File, dest string) error {
	// Reset the file pointer to the beginning before detecting the archive format.
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
			break // End of archive.
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

func verifyChecksum(file *os.File, checksumURL string) error {
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
	expected := strings.Fields(string(checksumData))
	if len(expected) == 0 {
		return fmt.Errorf("checksum file is empty")
	}
	expectedHash := expected[0]
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

func isInstalled(version string) bool {
	_, err := os.Stat(filepath.Join(versionsDir, version))
	return !os.IsNotExist(err)
}

func listInstalledVersions() ([]string, error) {
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read versions directory: %w", err)
	}
	var versions []string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "current" {
			versions = append(versions, entry.Name())
		}
	}
	return versions, nil
}

func updateSymlink(target, link string) error {
	if _, err := os.Lstat(link); err == nil {
		if err := os.Remove(link); err != nil {
			return fmt.Errorf("failed to remove existing symlink: %w", err)
		}
	}
	return os.Symlink(target, link)
}

func getCurrentVersion() (string, error) {
	link := filepath.Join(versionsDir, "current")
	target, err := os.Readlink(link)
	if err != nil {
		return "", fmt.Errorf("failed to read symlink: %w", err)
	}
	return filepath.Base(target), nil
}

package helpers

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

// Constants for utils operations.
const (
	DirPerm      = 0o755
	GoroutineNum = 2
	WindowsOS    = "windows"
)

// Errors for utils operations.
var (
	ErrNotSymlinkOrDir    = errors.New("current is not a symlink or directory")
	ErrNvimNotFound       = errors.New("neovim binary not found in")
	ErrConfigDoesNotExist = errors.New("configuration does not exist")
)

// Variables for utils operations.
var (
	UserHomeDir     = os.UserHomeDir
	LookPath        = exec.LookPath
	Fatalf          = logrus.Fatalf
	ExecCommandFunc = exec.CommandContext
)

// IsInstalled checks if a version directory exists within the versionsDir.
// It returns true if the directory exists, meaning that the version is installed.
//
// Example usage:
//
//	installed := IsInstalled("/path/to/versions", "v0.6.0")
//	if installed {
//	    fmt.Println("Version is installed")
//	}
func IsInstalled(versionsDir, version string) bool {
	_, err := os.Stat(filepath.Join(versionsDir, version))

	return !os.IsNotExist(err)
}

// ListInstalledVersions returns a list of installed version directory names
// found in the versionsDir, excluding any directory named "current".
//
// Example usage:
//
//	versions, err := ListInstalledVersions("/path/to/versions")
//	if err != nil {
//	    // handle error
//	}
//	fmt.Println("Installed versions:", versions)
func ListInstalledVersions(versionsDir string) ([]string, error) {
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "current" {
			versions = append(versions, entry.Name())
		}
	}

	return versions, nil
}

// UpdateSymlink creates a symlink (or junction/hardlink fallback on Windows).
// If isDir = true, fallback uses junctions (mklink /J).
// If isDir = false, fallback uses hardlinks (mklink /H).
//
// Example usage:
//
//	err := UpdateSymlink("/path/to/version/v0.6.0", "/path/to/versions/current", true)
//	if err != nil {
//	    // handle error
//	}
func UpdateSymlink(target, link string, isDir bool) error {
	// Remove old symlink if it exists.
	var err error

	_, err = os.Lstat(link)
	if err == nil {
		err = os.Remove(link)
		if err != nil {
			return err
		}
	}

	// Try normal symlink
	err = os.Symlink(target, link)
	if err == nil {
		return nil
	} else if runtime.GOOS != WindowsOS {
		// On non-Windows, fail fast
		return err
	}

	// Windows fallback
	if isDir {
		// Directory junction
		cmd := ExecCommandFunc(context.Background(), "cmd", "/C", "mklink", "/J", link, target)
		cmd.Stdout = os.Stdout

		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("failed to create junction for %s: %w", link, err)
		}

		logrus.Debugf("Created junction instead of symlink: %s -> %s", link, target)
	} else {
		// File hardlink
		cmd := ExecCommandFunc(context.Background(), "cmd", "/C", "mklink", "/H", link, target)
		cmd.Stdout = os.Stdout

		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("failed to create hardlink for %s: %w", link, err)
		}

		logrus.Debugf("Created hardlink instead of symlink: %s -> %s", link, target)
	}

	return nil
}

// GetCurrentVersion reads the "current" symlink (or junction on Windows)
// within versionsDir and returns the base name of its target directory,
// which indicates the currently active version.
//
// Example usage:
//
//	version, err := GetCurrentVersion("/path/to/versions")
//	if err != nil {
//	    // handle error
//	}
//	fmt.Println("Current version:", version)
func GetCurrentVersion(versionsDir string) (string, error) {
	link := filepath.Join(versionsDir, "current")

	info, err := os.Lstat(link)
	if err != nil {
		return "", fmt.Errorf("failed to lstat %s: %w", link, err)
	}

	// Case 1: it's a symlink
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(link)
		if err != nil {
			return "", fmt.Errorf("failed to read symlink %s: %w", link, err)
		}

		return filepath.Base(target), nil
	}

	// Case 2: it's a directory (could be junction or real dir)
	if info.IsDir() {
		return filepath.Base(link), nil
	}

	return "", fmt.Errorf("%w: %s", ErrNotSymlinkOrDir, link)
}

// GetNvimConfigBaseDir determines the canonical configuration directory
// used by Neovim according to its runtime path conventions.
//
// Resolution order:
//
//  1. If the environment variable XDG_CONFIG_HOME is set, Neovim looks under
//     $XDG_CONFIG_HOME/nvim. This is the highest-precedence override.
//     Example (Linux/macOS):
//     XDG_CONFIG_HOME="$HOME/.xdg"
//     Example (Windows, PowerShell):
//     $env:XDG_CONFIG_HOME="C:\xdg"
//
//  2. If XDG_CONFIG_HOME is not set, Neovim falls back to a platform-specific
//     default:
//
//     • Linux/macOS → $HOME/.config
//     Example: "/home/alice/.config"
//     "/Users/alice/.config"
//
//     • Windows → %LOCALAPPDATA%
//     Example: "C:\Users\alice\AppData\Local"
//
//  3. If LOCALAPPDATA is not set on Windows, this function falls back to
//     $HOME/.config/nvim for consistency with other platforms.
//
// Returns:
//   - The absolute path to the Neovim configuration directory.
//   - An error if the user’s home directory cannot be determined when required.
//
// Notes:
//   - This function does *not* consider tool-specific overrides such as
//     NVS_CONFIG_DIR, because it is intended to strictly reflect Neovim’s
//     own search path rules.
//   - Callers should ensure that the returned directory exists before use;
//     Neovim itself will create it lazily if needed.
func GetNvimConfigBaseDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return xdg, nil
	}

	if runtime.GOOS == WindowsOS {
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			return local, nil
		}
		// fallback to home/.config if LOCALAPPDATA is missing
	}

	home, err := UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config"), nil
}

// FindNvimBinary walks through the given directory to find a Neovim binary.
// On Windows it searches for "nvim.exe" (or prefixed names with .exe),
// while on other OSes it looks for "nvim" (or prefixed names) that are executable.
// The function returns the full path to the binary or an empty string if not found.
//
// Example usage:
//
//	binaryPath := FindNvimBinary("/path/to/version/v0.6.0")
//	if binaryPath == "" {
//	    fmt.Println("Neovim binary not found")
//	} else {
//	    fmt.Println("Found Neovim binary at:", binaryPath)
//	}
func FindNvimBinary(dir string) string {
	var binaryPath string

	err := filepath.WalkDir(dir, func(path string, dirEntry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !dirEntry.IsDir() {
			name := dirEntry.Name()
			if runtime.GOOS == WindowsOS {
				if strings.EqualFold(name, "nvim.exe") ||
					(strings.HasPrefix(strings.ToLower(name), "nvim-") && filepath.Ext(name) == ".exe") {
					// Go two levels up: ../../nvim-win64
					binaryPath = filepath.Dir(filepath.Dir(path))

					return io.EOF // break early
				}
			} else {
				if name == "nvim" || strings.HasPrefix(name, "nvim-") {
					info, err := dirEntry.Info()
					if err == nil && info.Mode()&0o111 != 0 {
						binaryPath = path

						return io.EOF // break early
					}
				}
			}
		}

		return nil
	})
	if err != nil && !errors.Is(err, io.EOF) {
		logrus.Fatalf("Failed to walk through nvim directory: %v", err)
	}

	return binaryPath
}

// UseVersion switches to the specified targetVersion by updating the "current" link,
// locating the Neovim binary in the version directory, and creating a global link
// in globalBinDir pointing to that binary. On Windows, falls back to junctions (dirs)
// or hardlinks (files) if symlinks are not allowed.
//
// Example usage:
//
//	err := UseVersion("v0.6.0", "/path/to/versions/current", "/path/to/versions", "/usr/local/bin")
//	if err != nil {
//	    // handle error
//	}
func UseVersion(
	targetVersion string,
	currentSymlink string,
	versionsDir string,
	globalBinDir string,
) error {
	var err error

	versionPath := filepath.Join(versionsDir, targetVersion)
	logrus.Debugf("Updating symlink to point to: %s", versionPath)

	err = UpdateSymlink(versionPath, currentSymlink, true)
	if err != nil {
		return fmt.Errorf("failed to switch version: %w", err)
	}

	nvimExec := FindNvimBinary(versionPath)
	if nvimExec == "" {
		_, err = fmt.Fprintf(os.Stdout,
			"%s Could not find Neovim binary in %s. Please check the installation structure.\n",
			ErrorIcon(),
			CyanText(versionPath),
		)
		if err != nil {
			logrus.Warnf("Failed to write to stdout: %v", err)
		}

		return fmt.Errorf("%w: %s", ErrNvimNotFound, versionPath)
	}

	targetBin := filepath.Join(globalBinDir, "nvim")

	_, err = os.Lstat(targetBin)
	if err == nil {
		err = os.Remove(targetBin)
		if err != nil {
			logrus.Warnf(
				"Failed to remove existing global bin symlink: %s, error: %v",
				targetBin,
				err,
			)
		} else {
			logrus.Debugf("Removed existing global bin symlink: %s", targetBin)
		}
	}

	if runtime.GOOS == WindowsOS {
		err = UpdateSymlink(nvimExec, targetBin, true)
		if err != nil {
			return fmt.Errorf("failed to create global nvim link: %w", err)
		}
	} else {
		err = UpdateSymlink(nvimExec, targetBin, false)
		if err != nil {
			return fmt.Errorf("failed to create global nvim link: %w", err)
		}
	}

	logrus.Debugf("Global Neovim binary updated: %s -> %s", targetBin, nvimExec)

	switchMsg := "Switched to Neovim " + CyanText(targetVersion)

	_, err = fmt.Fprintf(os.Stdout, "%s %s\n", SuccessIcon(), WhiteText(switchMsg))
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	if pathEnv := os.Getenv("PATH"); !strings.Contains(pathEnv, globalBinDir) {
		if runtime.GOOS == WindowsOS {
			// windows needs the whole directory to be linked
			nvimBinDir := filepath.Join(globalBinDir, "nvim", "bin")

			_, err = fmt.Fprintf(
				os.Stdout,
				"%s Run `nvs path` or manually add this directory to your PATH for convenience: %s\n",
				WarningIcon(),
				CyanText(nvimBinDir),
			)
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			logrus.Debugf("Global bin directory not found in PATH: %s", nvimBinDir)
		} else {
			_, err = fmt.Fprintf(os.Stdout, "%s Run `nvs path` or manually add this directory to your PATH for convenience: %s\n", WarningIcon(), CyanText(globalBinDir))
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			logrus.Debugf("Global bin directory not found in PATH: %s", globalBinDir)
		}
	}

	return nil
}

// GetInstalledReleaseIdentifier reads the version.txt file from the installed release
// directory (specified by alias) within versionsDir and returns the trimmed content.
//
// Example usage:
//
//	id, err := GetInstalledReleaseIdentifier("/path/to/versions", "v0.6.0")
//	if err != nil {
//	    // handle error
//	}
//	fmt.Println("Installed release identifier:", id)
func GetInstalledReleaseIdentifier(versionsDir, alias string) (string, error) {
	versionFile := filepath.Join(versionsDir, alias, "version.txt")

	data, err := os.ReadFile(versionFile)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

// LaunchNvimWithConfig launches Neovim using the provided configuration name.
// It sets the NVIM_APPNAME environment variable, locates the nvim executable in PATH,
// and runs Neovim with the current process's stdio.
//
// Example usage:
//
//	err := helpers.LaunchNvimWithConfig("myconfig")
//	if err != nil {
//		// handle error
//	}
func LaunchNvimWithConfig(configName string) error {
	var err error

	baseConfigDir, err := GetNvimConfigBaseDir()
	if err != nil {
		return fmt.Errorf("failed to determine config base dir: %w", err)
	}

	configDir := filepath.Join(baseConfigDir, configName)

	info, err := os.Stat(configDir)
	if err != nil || !info.IsDir() {
		_, err = fmt.Fprintf(os.Stdout,
			"%s %s\n",
			ErrorIcon(),
			WhiteText(
				fmt.Sprintf(
					"configuration '%s' does not exist in %s",
					CyanText(configName),
					CyanText(baseConfigDir),
				),
			),
		)
		if err != nil {
			logrus.Warnf("Failed to write to stdout: %v", err)
		}

		return fmt.Errorf(
			"configuration '%s' does not exist: %w",
			configName,
			ErrConfigDoesNotExist,
		)
	}

	err = os.Setenv("NVIM_APPNAME", configName)
	if err != nil {
		return fmt.Errorf("failed to set NVIM_APPNAME: %w", err)
	}

	nvimExec, err := LookPath("nvim")
	if err != nil {
		return fmt.Errorf("nvim not found in PATH: %w", err)
	}

	launch := ExecCommandFunc(context.Background(), nvimExec)

	launch.Env = append(os.Environ(), "NVIM_APPNAME="+configName)
	launch.Stdin = os.Stdin
	launch.Stdout = os.Stdout

	launch.Stderr = os.Stderr

	err = launch.Run()
	if err != nil {
		return fmt.Errorf("failed to launch nvim: %w", err)
	}

	return nil
}

// ClearDirectory removes all contents within the specified directory.
// It returns an error if any removal fails.
//
// Example usage:
//
//	err := ClearDirectory("/path/to/temp")
//	if err != nil {
//	    // handle error
//	}
func ClearDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())

		err := os.RemoveAll(path)
		if err != nil {
			return err
		}
	}

	return nil
}

// TimeFormat converts an ISO 8601 timestamp to a human-friendly date (YYYY-MM-DD).
// If the input cannot be parsed, it returns the original string.
//
// Example usage:
//
//	formatted := TimeFormat("2023-03-25T14:30:00Z")
//	fmt.Println("Formatted date:", formatted)
func TimeFormat(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}

	return t.Format("2006-01-02")
}

// ColorizeRow applies the given color to each cell in the row and returns a new slice
// with the colorized strings.
//
// Example usage:
//
//	row := []string{"Name", "Version", "Date"}
//	coloredRow := ColorizeRow(row, color.New(color.FgGreen))
//	fmt.Println("Colored row:", coloredRow)
func ColorizeRow(row []string, c *color.Color) []string {
	colored := make([]string, len(row))
	for i, cell := range row {
		colored[i] = c.Sprint(cell)
	}

	return colored
}

// CopyFile copies the content of the file from src to dst,
// sets the destination file's permissions to 0755, and returns an error if any step fails.
//
// Example usage:
//
//	err := CopyFile("/path/to/source", "/path/to/destination")
//	if err != nil {
//	    // handle error
//	}
func CopyFile(src, dst string) error {
	var err error

	inputFile, err := os.Open(src)
	if err != nil {
		return err
	}

	defer func() {
		cerr := inputFile.Close()
		if cerr != nil {
			logrus.Warnf("Failed to close source file %s: %v", src, cerr)
		}
	}()

	out, err := os.Create(dst)
	// Return err
	if err != nil {
		return err
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()

	_, err = io.Copy(out, inputFile)
	if err != nil {
		return err
	}

	err = os.Chmod(dst, DirPerm)
	if err != nil {
		return err
	}

	return nil
}

// RunCommandWithSpinner executes the provided command with an active spinner that updates its suffix
// based on the command's output. It captures both stdout and stderr and returns an error if the command fails.
//
// Example usage:
//
//	ctx := context.Background()
//	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
//	s.Start()
//	defer s.Stop()
//	cmd := exec.CommandContext(ctx, "echo", "Hello, world!")
//	if err := RunCommandWithSpinner(ctx, s, cmd); err != nil {
//	    // handle error
//	}
func RunCommandWithSpinner(ctx context.Context, spinner *spinner.Spinner, cmd *exec.Cmd) error {
	var err error

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// updateSpinner reads from the given pipe and updates the spinner's suffix based on the output.
	updateSpinner := func(pipeOutput io.Reader, waitGroup *sync.WaitGroup) {
		defer waitGroup.Done()

		scanner := bufio.NewScanner(pipeOutput)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				spinner.Suffix = " " + line
			}
		}
	}

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	var waitGroup sync.WaitGroup
	waitGroup.Add(GoroutineNum)

	go updateSpinner(stdoutPipe, &waitGroup)
	go updateSpinner(stderrPipe, &waitGroup)

	// Channel to capture command completion.
	cmdErrChan := make(chan error, 1)
	go func() {
		cmdErrChan <- cmd.Wait()
	}()

	// Wait for either the command to finish or the context to be canceled.
	select {
	case <-ctx.Done():
		return fmt.Errorf("command canceled: %w", ctx.Err())
	case err := <-cmdErrChan:
		// Wait for spinner update routines to finish.
		waitGroup.Wait()

		if err != nil {
			return err
		}
	}

	return nil
}

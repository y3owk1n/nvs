package utils

import (
	"bufio"
	"context"
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

var (
	userHomeDir     = os.UserHomeDir
	lookPath        = exec.LookPath
	fatalf          = logrus.Fatalf
	execCommandFunc = exec.CommandContext
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

// UpdateSymlink creates or updates a symbolic link at link pointing to target.
// If a link already exists, it will be removed before creating a new one.
//
// Example usage:
//
//	err := UpdateSymlink("/path/to/version/v0.6.0", "/path/to/versions/current")
//	if err != nil {
//	    // handle error
//	}
func UpdateSymlink(target, link string) error {
	if _, err := os.Lstat(link); err == nil {
		if err := os.Remove(link); err != nil {
			return err
		}
	}
	return os.Symlink(target, link)
}

// GetCurrentVersion reads the "current" symlink within versionsDir and returns
// the base name of its target directory, which indicates the currently active version.
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
	target, err := os.Readlink(link)
	if err != nil {
		return "", err
	}
	return filepath.Base(target), nil
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
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			name := d.Name()
			if runtime.GOOS == "windows" {
				if name == "nvim.exe" || (strings.HasPrefix(name, "nvim-") && filepath.Ext(name) == ".exe") {
					binaryPath = path
					return io.EOF // break early
				}
			} else {
				if name == "nvim" || strings.HasPrefix(name, "nvim-") {
					info, err := d.Info()
					if err == nil && info.Mode()&0111 != 0 {
						binaryPath = path
						return io.EOF // break early
					}
				}
			}
		}
		return nil
	})
	if err != nil && err != io.EOF {
		logrus.Fatalf("Failed to walk through nvim directory: %v", err)
	}

	return binaryPath
}

// UseVersion switches to the specified targetVersion by updating the "current" symlink,
// locating the Neovim binary in the version directory, and creating a global symlink
// in globalBinDir pointing to that binary. It prints messages to indicate success or warnings.
//
// Example usage:
//
//	err := UseVersion("v0.6.0", "/path/to/versions/current", "/path/to/versions", "/usr/local/bin")
//	if err != nil {
//	    // handle error
//	}
func UseVersion(targetVersion string, currentSymlink string, versionsDir string, globalBinDir string) error {
	versionPath := filepath.Join(versionsDir, targetVersion)
	logrus.Debugf("Updating symlink to point to: %s", versionPath)
	if err := UpdateSymlink(versionPath, currentSymlink); err != nil {
		return fmt.Errorf("failed to switch version: %v", err)
	}

	nvimExec := FindNvimBinary(versionPath)
	if nvimExec == "" {
		fmt.Printf("%s Could not find Neovim binary in %s. Please check the installation structure.\n", ErrorIcon(), CyanText(versionPath))
		return fmt.Errorf("neovim binary not found in: %s", versionPath)
	}

	targetBin := filepath.Join(globalBinDir, "nvim")
	if _, err := os.Lstat(targetBin); err == nil {
		os.Remove(targetBin)
		logrus.Debugf("Removed existing global bin symlink: %s", targetBin)
	}
	if err := os.Symlink(nvimExec, targetBin); err != nil {
		return fmt.Errorf("failed to create symlink in global bin: %v", err)
	}

	logrus.Debugf("Global Neovim binary updated: %s -> %s", targetBin, nvimExec)
	switchMsg := fmt.Sprintf("Switched to Neovim %s", CyanText(targetVersion))
	fmt.Printf("%s %s\n", SuccessIcon(), WhiteText(switchMsg))

	if pathEnv := os.Getenv("PATH"); !strings.Contains(pathEnv, globalBinDir) {
		fmt.Printf("%s Run `nvs path` or manually add this directory to your PATH for convenience: %s\n", WarningIcon(), CyanText(globalBinDir))
		logrus.Debugf("Global bin directory not found in PATH: %s", globalBinDir)
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
//	LaunchNvimWithConfig("myconfig")
//	// Neovim will be launched with configuration located at ~/.config/myconfig
func LaunchNvimWithConfig(configName string) {
	home, err := userHomeDir()
	if err != nil {
		fatalf("Failed to get home directory: %v", err)
	}
	configDir := filepath.Join(home, ".config", configName)

	info, err := os.Stat(configDir)
	if os.IsNotExist(err) || !info.IsDir() {
		fmt.Printf("%s %s\n", ErrorIcon(), WhiteText(fmt.Sprintf("configuration '%s' does not exist in %s", CyanText(configName), CyanText("~/.config"))))
		return
	}

	os.Setenv("NVIM_APPNAME", configName)

	nvimExec, err := lookPath("nvim")
	if err != nil {
		fatalf("nvim not found in PATH: %v", err)
	}
	launch := exec.Command(nvimExec)
	launch.Env = append(os.Environ(), "NVIM_APPNAME="+configName)
	launch.Stdin = os.Stdin
	launch.Stdout = os.Stdout
	launch.Stderr = os.Stderr
	if err := launch.Run(); err != nil {
		fatalf("Failed to launch nvim: %v", err)
	}
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
		if err := os.RemoveAll(path); err != nil {
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
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}

	if err = os.Chmod(dst, 0755); err != nil {
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
func RunCommandWithSpinner(ctx context.Context, s *spinner.Spinner, cmd *exec.Cmd) error {
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %v", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %v", err)
	}

	// updateSpinner reads from the given pipe and updates the spinner's suffix based on the output.
	updateSpinner := func(pipeOutput io.Reader, wg *sync.WaitGroup) {
		defer wg.Done()
		scanner := bufio.NewScanner(pipeOutput)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				s.Suffix = " " + line
			}
		}
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go updateSpinner(stdoutPipe, &wg)
	go updateSpinner(stderrPipe, &wg)

	// Channel to capture command completion.
	cmdErrChan := make(chan error, 1)
	go func() {
		cmdErrChan <- cmd.Wait()
	}()

	// Wait for either the command to finish or the context to be cancelled.
	select {
	case <-ctx.Done():
		return fmt.Errorf("command cancelled: %v", ctx.Err())
	case err := <-cmdErrChan:
		// Wait for spinner update routines to finish.
		wg.Wait()
		if err != nil {
			return err
		}
	}

	return nil
}

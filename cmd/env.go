package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	// ErrRequiredDirsNotDetermined is returned when required directories cannot be determined.
	ErrRequiredDirsNotDetermined = errors.New(
		"one or more required directories could not be determined",
	)
	// ErrUnsupportedShell is returned when the shell type is not supported.
	ErrUnsupportedShell = errors.New("unsupported shell type")
)

// envCmd represents the "env" command.
// It prints the NVS environment configuration variables: NVS_CONFIG_DIR, NVS_CACHE_DIR, and NVS_BIN_DIR.
// If these variables are not explicitly set, default locations are determined using the user's system directories.
//
// Example usage:
//
//	nvs env
//
// This command will output a table displaying the values for NVS_CONFIG_DIR, NVS_CACHE_DIR, and NVS_BIN_DIR.
var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Print NVS env configurations",
	Long:  "Prints the env configuration used by NVS (NVS_CONFIG_DIR, NVS_CACHE_DIR, and NVS_BIN_DIR).",
	RunE:  RunEnv,
}

// RunEnv executes the env command.
func RunEnv(cmd *cobra.Command, _ []string) error {
	logrus.Debug("Executing env command")

	// Determine NVS_CONFIG_DIR from environment or default to <UserConfigDir>/nvs
	configDir := os.Getenv("NVS_CONFIG_DIR")
	if configDir == "" {
		c, err := os.UserConfigDir()
		if err == nil {
			configDir = filepath.Join(c, "nvs")
		} else {
			logrus.Warn("Failed to retrieve user config directory")

			configDir = "Unavailable"
		}
	}

	logrus.Debugf("Resolved configDir: %s", configDir)

	// Determine NVS_CACHE_DIR from environment or default to <UserCacheDir>/nvs
	cacheDir := os.Getenv("NVS_CACHE_DIR")
	if cacheDir == "" {
		c, err := os.UserCacheDir()
		if err == nil {
			cacheDir = filepath.Join(c, "nvs")
		} else {
			logrus.Warn("Failed to retrieve user cache directory")

			cacheDir = "Unavailable"
		}
	}

	logrus.Debugf("Resolved cacheDir: %s", cacheDir)

	// Determine NVS_BIN_DIR from environment or default to <UserHomeDir>/.local/bin
	binDir := os.Getenv("NVS_BIN_DIR")
	if binDir == "" {
		if runtime.GOOS == windows {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get user home directory: %w", err)
			}

			binDir = filepath.Join(home, "AppData", "Local", "Programs")
			logrus.Debugf("Using Windows binary directory: %s", binDir)
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get user home directory: %w", err)
			}

			binDir = filepath.Join(home, ".local", "bin")
			logrus.Debugf("Using default binary directory: %s", binDir)
		}
	}

	logrus.Debugf("Resolved binDir: %s", binDir)

	source, _ := cmd.Flags().GetBool("source")
	shell, _ := cmd.Flags().GetString("shell")
	logrus.Debugf("--source: %v, --shell: %q", source, shell)

	if source {
		// Let's try to detect the shell we're running in
		if shell == "" {
			shell = DetectShell()
		}

		logrus.Debugf("Using shell for output: %q", shell)

		var err error

		// fail if we can't determine the required directories
		if configDir == "" || cacheDir == "" || binDir == "" {
			logrus.Error("One or more required directories could not be determined")

			return ErrRequiredDirsNotDetermined
		}

		// add binDir to PATH if it's not already there, avoid duplicates
		addPath := !strings.Contains(os.Getenv("PATH"), binDir)
		logrus.Debugf("binDir already in PATH: %v (addPath=%v)", !addPath, addPath)

		// explicitly default to error `unsupported`, add in more shell in future
		switch shell {
		case "fish":
			_, err = fmt.Fprintf(os.Stdout, "set -gx NVS_CONFIG_DIR %q;\n", configDir)
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			_, err = fmt.Fprintf(os.Stdout, "set -gx NVS_CACHE_DIR %q;\n", cacheDir)
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			_, err = fmt.Fprintf(os.Stdout, "set -gx NVS_BIN_DIR %q;\n", binDir)
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			if addPath {
				_, err = fmt.Fprintf(os.Stdout, "set -gx PATH %q $PATH;\n", binDir)
				if err != nil {
					logrus.Warnf("Failed to write to stdout: %v", err)
				}
			}
		case "bash", "zsh", "sh", "":
			_, err = fmt.Fprintf(os.Stdout, "export NVS_CONFIG_DIR=%q\n", configDir)
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			_, err = fmt.Fprintf(os.Stdout, "export NVS_CACHE_DIR=%q\n", cacheDir)
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			_, err = fmt.Fprintf(os.Stdout, "export NVS_BIN_DIR=%q\n", binDir)
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			if addPath {
				_, err = fmt.Fprintf(os.Stdout, "export PATH=%q:$PATH\n", binDir)
				if err != nil {
					logrus.Warnf("Failed to write to stdout: %v", err)
				}
			}
		default:
			logrus.Errorf("Unsupported shell type %q", shell)

			return fmt.Errorf("%q: %w", shell, ErrUnsupportedShell)
		}

		return nil
	}

	// Create a table to display the configuration variables.
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Variable", "Value"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
	)
	table.SetTablePadding("1")
	table.SetBorder(false)
	table.SetRowLine(false)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetAutoWrapText(false)

	// Append each configuration variable and its value (with colored output).
	table.Append([]string{
		"NVS_CONFIG_DIR",
		color.New(color.Bold, color.FgCyan).Sprint(configDir),
	})
	table.Append([]string{
		"NVS_CACHE_DIR",
		color.New(color.Bold, color.FgCyan).Sprint(cacheDir),
	})
	table.Append([]string{
		"NVS_BIN_DIR",
		color.New(color.Bold, color.FgCyan).Sprint(binDir),
	})

	// Render the table to stdout.
	table.Render()

	return nil
}

// DetectShell detects the current shell.
func DetectShell() string {
	logrus.Debug("Attempting to detect shell via parent process")
	// Check parent process command (ps -p $$)
	cmd := exec.CommandContext(
		context.Background(),
		"ps",
		"-p",
		strconv.Itoa(os.Getppid()),
		"-o",
		"comm=",
	)

	out, err := cmd.Output()
	if err == nil {
		shell := strings.TrimSpace(string(out))
		logrus.Debugf("ps output: %q", shell)

		shell = filepath.Base(shell)

		// remove login shell dash
		shell = strings.TrimPrefix(shell, "-")

		// normalize the case
		shell = strings.ToLower(shell)

		if shell != "" {
			logrus.Debugf("Detected shell from ps: %q", shell)

			return shell
		}
	} else {
		logrus.Warnf("ps command failed: %v", err)
	}

	// Fallback to SHELL env var
	logrus.Debug("Falling back to SHELL env var")

	if sh := os.Getenv("SHELL"); sh != "" {
		base := filepath.Base(sh)
		logrus.Debugf("Detected shell from $SHELL: %q", base)

		return base
	}

	logrus.Warn("Could not detect shell")

	return ""
}

// init registers the envCmd with the root command.
func init() {
	rootCmd.AddCommand(envCmd)
	envCmd.Flags().
		Bool("source", false, "Export environment variables so that they can be piped in source")
	envCmd.Flags().
		String("shell", "", "Shell type for --source output (bash|zsh|sh|fish). Auto-detected if not provided.")
}

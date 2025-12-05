package cmd

import (
	"context"
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
	"github.com/y3owk1n/nvs/internal/constants"
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

	// Determine directories using getter functions
	// NVS_CONFIG_DIR is the parent of versions directory
	configDir := filepath.Dir(GetVersionsDir())
	logrus.Debugf("Resolved configDir: %s", configDir)

	// NVS_CACHE_DIR is the directory containing the cache file
	cacheDir := filepath.Dir(GetCacheFilePath())
	logrus.Debugf("Resolved cacheDir: %s", cacheDir)

	// NVS_BIN_DIR is the global binary directory
	binDir := GetGlobalBinDir()
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
		if configDir == "" || configDir == constants.UnavailableDir ||
			cacheDir == "" || cacheDir == constants.UnavailableDir ||
			binDir == "" {
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
	if runtime.GOOS == constants.WindowsOS {
		return detectShellWindows()
	}

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

// detectShellWindows detects the shell on Windows systems.
func detectShellWindows() string {
	logrus.Debug("Detecting shell on Windows")

	// Check for PowerShell
	if psModulePath := os.Getenv("PSModulePath"); psModulePath != "" {
		logrus.Debug("Detected PowerShell via PSModulePath")

		return "powershell"
	}

	// Check COMSPEC for cmd.exe
	if comspec := os.Getenv("COMSPEC"); comspec != "" {
		base := strings.ToLower(filepath.Base(comspec))
		if base == "cmd.exe" {
			logrus.Debug("Detected cmd.exe via COMSPEC")

			return "cmd"
		}
	}

	// Try to get parent process name using tasklist (Windows equivalent of ps)
	cmd := exec.CommandContext(
		context.Background(),
		"tasklist",
		"/FI",
		fmt.Sprintf("PID eq %d", os.Getppid()),
		"/FO",
		"CSV",
		"/NH",
	)

	out, err := cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) > 0 {
			// Parse CSV: "Image Name","PID","Session Name","Session#","Mem Usage"
			fields := strings.Split(lines[0], ",")
			if len(fields) >= 1 {
				processName := strings.Trim(strings.TrimSpace(fields[0]), "\"")
				processName = strings.ToLower(processName)

				logrus.Debugf("Parent process: %s", processName)

				switch processName {
				case "powershell.exe":
					return "powershell"
				case "pwsh.exe":
					return "pwsh"
				case "cmd.exe":
					return "cmd"
				}
			}
		}
	} else {
		logrus.Warnf("tasklist command failed: %v", err)
	}

	logrus.Warn("Could not detect shell on Windows")

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

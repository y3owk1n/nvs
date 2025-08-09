package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
	Run: func(cmd *cobra.Command, args []string) {
		logrus.Debug("Executing env command")

		// Determine NVS_CONFIG_DIR from environment or default to <UserConfigDir>/nvs
		configDir := os.Getenv("NVS_CONFIG_DIR")
		if configDir == "" {
			if c, err := os.UserConfigDir(); err == nil {
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
			if c, err := os.UserCacheDir(); err == nil {
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
			if home, err := os.UserHomeDir(); err == nil {
				binDir = filepath.Join(home, ".local", "bin")
			} else {
				logrus.Warn("Failed to retrieve user home directory")
				binDir = "Unavailable"
			}
		}
		logrus.Debugf("Resolved binDir: %s", binDir)

		source, _ := cmd.Flags().GetBool("source")
		shell, _ := cmd.Flags().GetString("shell")
		logrus.Debugf("--source: %v, --shell: %q", source, shell)

		if source {
			// Let's try to detect the shell we're running in
			if shell == "" {
				shell = detectShell()
			}
			logrus.Debugf("Using shell for output: %q", shell)

			// fail if we can't determine the required directories
			if configDir == "" || cacheDir == "" || binDir == "" {
				logrus.Error("One or more required directories could not be determined")
				os.Exit(1)
			}

			// add binDir to PATH if it's not already there, avoid duplicates
			addPath := !strings.Contains(os.Getenv("PATH"), binDir)
			logrus.Debugf("binDir already in PATH: %v (addPath=%v)", !addPath, addPath)

			// explicitly default to error `unsupported`, add in more shell in future
			switch shell {
			case "fish":
				fmt.Printf("set -gx NVS_CONFIG_DIR %q;\n", configDir)
				fmt.Printf("set -gx NVS_CACHE_DIR %q;\n", cacheDir)
				fmt.Printf("set -gx NVS_BIN_DIR %q;\n", binDir)
				if addPath {
					fmt.Printf("set -gx PATH %q $PATH;\n", binDir)
				}
			case "bash", "zsh", "sh", "":
				fmt.Printf("export NVS_CONFIG_DIR=%q\n", configDir)
				fmt.Printf("export NVS_CACHE_DIR=%q\n", cacheDir)
				fmt.Printf("export NVS_BIN_DIR=%q\n", binDir)
				if addPath {
					fmt.Printf("export PATH=%q:$PATH\n", binDir)
				}
			default:
				logrus.Errorf("Unsupported shell type %q", shell)
				os.Exit(1)
			}
			return
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
	},
}

func detectShell() string {
	logrus.Debug("Attempting to detect shell via parent process")
	// Check parent process command (ps -p $$)
	out, err := exec.Command("ps", "-p", fmt.Sprint(os.Getppid()), "-o", "comm=").Output()
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
	envCmd.Flags().Bool("source", false, "Export environment variables so that they can be piped in source")
	envCmd.Flags().String("shell", "", "Shell type for --source output (bash|zsh|sh|fish). Auto-detected if not provided.")
}

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/ui"
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
	jsonOutput, _ := cmd.Flags().GetBool("json")
	logrus.Debugf("--source: %v, --shell: %q, --json: %v", source, shell, jsonOutput)

	if source && jsonOutput {
		return ErrMutuallyExclusiveFlags
	}

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
		//
		// The previous code used strings.Contains on the raw PATH
		// string, which produces false positives when binDir is a
		// prefix or substring of some other PATH entry (e.g.
		// binDir = "/home/u/bin", PATH = "/home/u/bin-extra:/usr/bin"
		// would have matched). The correct check splits PATH on
		// the platform's path-list separator and does exact-match
		// comparison on each entry.
		addPath := !pathListContains(os.Getenv("PATH"), binDir)
		logrus.Debugf("binDir already in PATH: %v (addPath=%v)", !addPath, addPath)

		// explicitly default to error `unsupported`, add in more shell in future
		switch shell {
		case "fish":
			// shellQuote uses POSIX-style single-quote escaping
			// (works in fish too). It is required because Go's
			// %q produces a double-quoted Go string that fish
			// would interpret as allowing $-expansion, e.g. a
			// path containing a literal '$' would be re-expanded
			// by the shell. Single-quote escaping prevents all
			// expansion in both POSIX shells and fish.
			_, err = fmt.Fprintf(
				os.Stdout,
				"set -gx NVS_CONFIG_DIR %s;\n",
				shellQuote(configDir),
			)
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			_, err = fmt.Fprintf(
				os.Stdout,
				"set -gx NVS_CACHE_DIR %s;\n",
				shellQuote(cacheDir),
			)
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			_, err = fmt.Fprintf(
				os.Stdout,
				"set -gx NVS_BIN_DIR %s;\n",
				shellQuote(binDir),
			)
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			if addPath {
				_, err = fmt.Fprintf(
					os.Stdout,
					"set -gx PATH %s $PATH;\n",
					shellQuote(binDir),
				)
				if err != nil {
					logrus.Warnf("Failed to write to stdout: %v", err)
				}
			}
		case "bash", "zsh", "sh", "":
			_, err = fmt.Fprintf(
				os.Stdout,
				"export NVS_CONFIG_DIR=%s\n",
				shellQuote(configDir),
			)
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			_, err = fmt.Fprintf(
				os.Stdout,
				"export NVS_CACHE_DIR=%s\n",
				shellQuote(cacheDir),
			)
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			_, err = fmt.Fprintf(
				os.Stdout,
				"export NVS_BIN_DIR=%s\n",
				shellQuote(binDir),
			)
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			if addPath {
				_, err = fmt.Fprintf(
					os.Stdout,
					"export PATH=%s:$PATH\n",
					shellQuote(binDir),
				)
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

	if jsonOutput {
		data := map[string]string{
			"NVS_CONFIG_DIR": configDir,
			"NVS_CACHE_DIR":  cacheDir,
			"NVS_BIN_DIR":    binDir,
		}

		return outputJSON(data)
	}

	// Create a table to display the configuration variables.
	//
	// The default (no --source, no --json) view is the
	// human-readable summary, so it routes through the new
	// ui.Table primitive for visual consistency with the
	// other commands. Each row is "VARIABLE  <value>", with
	// the value rendered in the Accent (primary) color so the
	// path is the data the user is actually reading.
	tbl := ui.Table.New("Variable", "Value")

	tbl.Row("NVS_CONFIG_DIR", ui.Message.Accent(configDir))
	tbl.Row("NVS_CACHE_DIR", ui.Message.Accent(cacheDir))
	tbl.Row("NVS_BIN_DIR", ui.Message.Accent(binDir))

	_, _ = fmt.Fprint(os.Stdout, ui.Banner.Logo())
	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprint(os.Stdout, tbl.Render(ui.Style.Palette()))

	return nil
}

// pathListContains reports whether item is a path-list entry in
// list. The list is split on the platform's path-list separator
// (':' on Unix, ';' on Windows) and each entry is compared for
// exact equality. This is the correct semantic for checking
// whether a directory is already on PATH; substring matching
// (strings.Contains) yields false positives whenever item is a
// prefix or substring of any other entry.
func pathListContains(list, item string) bool {
	if list == "" {
		return false
	}

	return slices.Contains(strings.Split(list, string(os.PathListSeparator)), item)
}

// shellQuote returns a POSIX-shell-safe single-quoted form of s
// that is also valid in fish. The single-quote escape works by
// closing the current single-quoted string, inserting a literal
// backslash-escaped single quote, and re-opening: '\”. The result
// is interpreted by the shell as a literal string with no
// expansion of $, `, \, or any other metacharacter.
//
// This is used in place of Go's %q verb because %q produces a
// double-quoted Go string that fish would interpret as allowing
// $-expansion — a path containing a literal '$' would be silently
// re-expanded by the shell.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
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
	envCmd.Flags().
		Bool("json", false, "Output in JSON format")
}

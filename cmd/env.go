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

	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/log"
	"github.com/y3owk1n/nvs/internal/ui"
)

// envCmd represents the "env" command.
// It prints the NVS environment configuration variables and
// their resolved values: paths (NVS_CONFIG_DIR, NVS_CACHE_DIR,
// NVS_BIN_DIR), behavior toggles (NVS_GITHUB_MIRROR,
// NVS_USE_GLOBAL_CACHE), and logger settings (NVS_LOG,
// NVS_LOG_FILE).
//
// Example usage:
//
//	nvs env                 # human-readable table
//	nvs env --json          # machine-readable
//	nvs env --source        # shell-eval'd export statements (paths only)
//
// The --source mode emits only the path variables (the ones
// nvs needs to find its on-disk state). Behavior and log
// settings are user preferences and are deliberately not
// exported, so a user who runs `eval "$(nvs env --source)"`
// at shell startup does not unintentionally pin a debug log
// level into every subsequent invocation.
var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Print NVS env configurations",
	Long: `Prints the env configuration used by NVS.

Variables shown:
  Paths     NVS_CONFIG_DIR, NVS_CACHE_DIR, NVS_BIN_DIR
  Behavior  NVS_GITHUB_MIRROR, NVS_USE_GLOBAL_CACHE
  Logging   NVS_LOG, NVS_LOG_FILE`,
	RunE: RunEnv,
}

// envVar is one row in the env table. A separate struct (rather
// than parallel slices) keeps the rendering loop readable and
// lets us add per-row hints later (e.g. a "Default" column) with
// only the affected call sites changing.
type envVar struct {
	Name  string
	Value string
	// IsPath is true for the three NVS_*_DIR vars. The
	// --source path emits only these (see envCmd doc).
	IsPath bool
}

// RunEnv executes the env command.
func RunEnv(cmd *cobra.Command, _ []string) error {
	log.Debug("executing env command")

	source, _ := cmd.Flags().GetBool("source")
	shell, _ := cmd.Flags().GetString("shell")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	log.Debug("flags", "source", source, "shell", shell, "json", jsonOutput)

	if source && jsonOutput {
		return ErrMutuallyExclusiveFlags
	}

	vars := collectEnvVars()

	if source {
		return renderEnvSource(vars, shell)
	}

	if jsonOutput {
		data := make(map[string]string, len(vars))
		for _, v := range vars {
			data[v.Name] = v.Value
		}

		return outputJSON(data)
	}

	return renderEnvText(vars)
}

// collectEnvVars resolves every env var nvs cares about to its
// current effective value. For unset optional vars we substitute
// a human-readable placeholder ("(unset)" or the default) so the
// table never has empty cells — a blank value is easy to misread
// as "set to empty string", which is a different thing.
func collectEnvVars() []envVar {
	configDir := filepath.Dir(GetVersionsDir())
	cacheDir := filepath.Dir(GetCacheFilePath())
	binDir := GetGlobalBinDir()

	log.Debug("resolved paths", "config", configDir, "cache", cacheDir, "bin", binDir)

	githubMirror := os.Getenv("NVS_GITHUB_MIRROR")
	if githubMirror == "" {
		githubMirror = "(unset, using github.com)"
	}

	useGlobalCache := "false"
	if envValue := os.Getenv("NVS_USE_GLOBAL_CACHE"); strings.EqualFold(envValue, "true") ||
		envValue == "1" {
		useGlobalCache = "true"
	}

	// Show the EFFECTIVE log level (after parsing, after
	// fallbacks) rather than the raw env var, so an invalid
	// value like NVS_LOG=potato reports the level that is
	// actually in use (warn) rather than the value the user
	// typed.
	logLevel := log.GetLevel().String()

	logFile := os.Getenv("NVS_LOG_FILE")
	if logFile == "" {
		logFile = "(unset, stderr only)"
	}

	return []envVar{
		{Name: "NVS_CONFIG_DIR", Value: configDir, IsPath: true},
		{Name: "NVS_CACHE_DIR", Value: cacheDir, IsPath: true},
		{Name: "NVS_BIN_DIR", Value: binDir, IsPath: true},
		{Name: "NVS_GITHUB_MIRROR", Value: githubMirror},
		{Name: "NVS_USE_GLOBAL_CACHE", Value: useGlobalCache},
		{Name: "NVS_LOG", Value: logLevel},
		{Name: "NVS_LOG_FILE", Value: logFile},
	}
}

// renderEnvText writes the default human-readable view: a
// banner followed by a two-column table. Values are rendered in
// the Accent color so the data the user is looking for stands
// out from the variable names.
func renderEnvText(vars []envVar) error {
	tbl := ui.Table.New("Variable", "Value")
	for _, v := range vars {
		tbl.Row(v.Name, ui.Message.Accent(v.Value))
	}

	_, _ = fmt.Fprint(os.Stdout, ui.Banner.Logo())
	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprint(os.Stdout, tbl.Render(ui.Style.Palette()))

	return nil
}

// renderEnvSource emits shell-eval'd export statements for the
// path variables. Only path vars are exported — behavior and
// log settings are user preferences that should be set in the
// user's shell profile, not pinned by `eval "$(nvs env --source)"`.
func renderEnvSource(vars []envVar, shell string) error {
	if shell == "" {
		shell = DetectShell()
	}

	log.Debug("source mode shell", "shell", shell)

	// Validate that every path var was resolvable. A missing
	// path here means InitConfig found no usable filesystem
	// location for one of the three nvs dirs — surfacing it
	// loudly is better than silently writing a broken eval.
	for _, envVarEntry := range vars {
		if !envVarEntry.IsPath {
			continue
		}

		if envVarEntry.Value == "" || envVarEntry.Value == constants.UnavailableDir {
			log.Debug("required directory could not be determined", "var", envVarEntry.Name)

			return ErrRequiredDirsNotDetermined
		}
	}

	binDir := envVarValue(vars, "NVS_BIN_DIR")
	addPath := !pathListContains(os.Getenv("PATH"), binDir)

	log.Debug("source mode PATH state", "bin_dir_in_path", !addPath, "will_prepend", addPath)

	switch shell {
	case "fish":
		return emitFishSource(vars, binDir, addPath)
	case "bash", "zsh", "sh", "":
		return emitPosixSource(vars, binDir, addPath)
	default:
		// Don't log+return: cobra prints the returned error
		// once on stderr. Tracing it here would duplicate
		// the output. Operator-grade trace stays at debug.
		log.Debug("unsupported shell", "shell", shell)

		return fmt.Errorf("%q: %w", shell, ErrUnsupportedShell)
	}
}

// envVarValue returns the value for the named variable. Used by
// renderEnvSource so the path-export loop can stay generic and
// the PATH-prepend logic can still pick out the bin dir.
func envVarValue(vars []envVar, name string) string {
	for _, envVarEntry := range vars {
		if envVarEntry.Name == name {
			return envVarEntry.Value
		}
	}

	return ""
}

// emitFishSource writes fish-style `set -gx NAME VALUE;` lines
// for every path var, then prepends binDir to PATH if it is not
// already on the list. Errors writing to stdout are logged but
// not returned: stdout is a pipe to the user's shell, and there
// is nothing useful the caller can do if the pipe has closed.
func emitFishSource(vars []envVar, binDir string, addPath bool) error {
	for _, envVarEntry := range vars {
		if !envVarEntry.IsPath {
			continue
		}

		_, err := fmt.Fprintf(
			os.Stdout,
			"set -gx %s %s;\n",
			envVarEntry.Name,
			shellQuote(envVarEntry.Value),
		)
		if err != nil {
			log.Warn("write stdout failed", "err", err)
		}
	}

	if addPath {
		_, err := fmt.Fprintf(
			os.Stdout,
			"set -gx PATH %s $PATH;\n",
			shellQuote(binDir),
		)
		if err != nil {
			log.Warn("write stdout failed", "err", err)
		}
	}

	return nil
}

// emitPosixSource is the POSIX-shell (bash/zsh/sh) counterpart
// of emitFishSource.
func emitPosixSource(vars []envVar, binDir string, addPath bool) error {
	for _, envVarEntry := range vars {
		if !envVarEntry.IsPath {
			continue
		}

		_, err := fmt.Fprintf(
			os.Stdout,
			"export %s=%s\n",
			envVarEntry.Name,
			shellQuote(envVarEntry.Value),
		)
		if err != nil {
			log.Warn("write stdout failed", "err", err)
		}
	}

	if addPath {
		_, err := fmt.Fprintf(
			os.Stdout,
			"export PATH=%s:$PATH\n",
			shellQuote(binDir),
		)
		if err != nil {
			log.Warn("write stdout failed", "err", err)
		}
	}

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

	log.Debug("Attempting to detect shell via parent process")
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
		log.Debugf("ps output: %q", shell)

		shell = filepath.Base(shell)

		// remove login shell dash
		shell = strings.TrimPrefix(shell, "-")

		// normalize the case
		shell = strings.ToLower(shell)

		if shell != "" {
			log.Debugf("Detected shell from ps: %q", shell)

			return shell
		}
	} else {
		log.Warnf("ps command failed: %v", err)
	}

	// Fallback to SHELL env var
	log.Debug("Falling back to SHELL env var")

	if sh := os.Getenv("SHELL"); sh != "" {
		base := filepath.Base(sh)
		log.Debugf("Detected shell from $SHELL: %q", base)

		return base
	}

	log.Warn("Could not detect shell")

	return ""
}

// detectShellWindows detects the shell on Windows systems.
func detectShellWindows() string {
	log.Debug("Detecting shell on Windows")

	// Check for PowerShell
	if psModulePath := os.Getenv("PSModulePath"); psModulePath != "" {
		log.Debug("Detected PowerShell via PSModulePath")

		return "powershell"
	}

	// Check COMSPEC for cmd.exe
	if comspec := os.Getenv("COMSPEC"); comspec != "" {
		base := strings.ToLower(filepath.Base(comspec))
		if base == "cmd.exe" {
			log.Debug("Detected cmd.exe via COMSPEC")

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

				log.Debugf("Parent process: %s", processName)

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
		log.Warnf("tasklist command failed: %v", err)
	}

	log.Warn("Could not detect shell on Windows")

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

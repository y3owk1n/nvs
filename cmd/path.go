package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/log"
	"github.com/y3owk1n/nvs/internal/ui"
)

// pathCmd represents the "path" command.
// It automatically adds the global binary directory to the user's PATH by modifying the appropriate shell configuration file.
// Depending on the operating system and shell, it determines the proper rc file (e.g. ~/.bashrc, ~/.zshrc, or ~/.config/fish/config.fish)
// and outputs a diff of the changes that will be applied. The user is then prompted to confirm the modification.
//
// Example usage:
//
//	nvs path
//
// On non-Windows systems, if the global binary directory is not already in the PATH, this command displays a diff (the new export command)
// and asks the user to confirm. If confirmed, the export command is added to the rc file. On Windows or Nix-managed shells, the command
// advises manual configuration.
var pathCmd = &cobra.Command{
	Use:   "path",
	Short: "Automatically add the global binary directory to your PATH",
	RunE:  RunPath,
}

// RunPath executes the path command.
func RunPath(_ *cobra.Command, _ []string) error {
	log.Debug("Running path command")

	// On Windows, automatic PATH modifications are not implemented.
	if runtime.GOOS == constants.WindowsOS {
		// Use GetGlobalBinDir() to get the path
		nvimBinDir := filepath.Join(GetGlobalBinDir(), "nvim", "bin")

		log.Debug("Detected Windows OS")

		ui.Message.Warnf("Automatic PATH setup is not implemented for Windows.")
		ui.Message.Infof(
			"Please add %s to your PATH environment variable manually.",
			ui.Message.Accent(nvimBinDir),
		)

		return nil
	}

	// Check if the global binary directory is already in the PATH.
	pathEnv := os.Getenv("PATH")
	log.Debug("Current PATH: ", pathEnv)

	// Check if GetGlobalBinDir() is already in PATH. Hoist the
	// Clean() of GetGlobalBinDir() out of the loop — it is
	// loop-invariant, and Clean() walks the path string on every
	// call.
	binDirClean := filepath.Clean(GetGlobalBinDir())

	pathSeparator := string(os.PathListSeparator)
	paths := strings.Split(pathEnv, pathSeparator)

	found := false
	for _, p := range paths {
		if filepath.Clean(p) == binDirClean {
			found = true

			break
		}
	}

	if found {
		log.Debugf("PATH already contains %s", GetGlobalBinDir())

		ui.Message.Infof(
			"Your PATH already contains %s.",
			ui.Message.Accent(GetGlobalBinDir()),
		)

		return nil
	}

	// Determine the user's shell; default to /bin/bash if not set.
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
		// Verify the default shell exists
		_, err := os.Stat(shell)
		if os.IsNotExist(err) {
			ui.Message.Warnf(
				"Default shell %s does not exist, PATH setup may not work correctly",
				shell,
			)
		}
	}

	// If running in a Nix-managed shell, advise manual configuration.
	isNixShell := os.Getenv("NIX_SHELL") != "" || strings.Contains(shell, "/nix/store")
	if isNixShell {
		log.Debug("Detected Nix shell environment")

		ui.Message.Warnf(
			"It appears your shell is managed by Nix. Automatic PATH modifications may not work as expected.",
		)
		ui.Message.Infof(
			"Please update your Nix configuration manually to include %s in your PATH.",
			ui.Message.Accent(GetGlobalBinDir()),
		)

		return nil
	}

	log.Debug("Detected shell: ", shell)

	// Get the base name of the shell executable (e.g. bash, zsh, fish).
	shellName := filepath.Base(shell)
	log.Debug("Shell name: ", shellName)

	// Determine the rc file path and export command based on the shell.
	var rcFile, exportCmd string

	exportCmdComment := "# Added by nvs"

	// Get home directory, preferring HOME env var but falling back to os.UserHomeDir
	home := os.Getenv("HOME")
	if home == "" {
		var err error

		home, err = os.UserHomeDir()
		if err != nil {
			log.Warnf("Failed to get home directory: %v", err)

			ui.Message.Warnf(
				"Cannot determine home directory. Please set HOME environment variable.",
			)

			return nil
		}
	}

	switch shellName {
	case constants.ShellBash, constants.ShellZsh:
		rcFile = filepath.Join(home, fmt.Sprintf(".%src", shellName))
		exportCmd = fmt.Sprintf("export PATH=\"$PATH:%s\"", GetGlobalBinDir())
	case constants.ShellFish:
		rcFile = filepath.Join(home, ".config", "fish", "config.fish")
		// Ensure parent directory exists for fish config
		err := os.MkdirAll(filepath.Dir(rcFile), constants.DirPerm)
		if err != nil {
			return fmt.Errorf("failed to create fish config directory: %w", err)
		}

		exportCmd = "set -gx PATH $PATH " + GetGlobalBinDir()
	default:
		log.Debug("Unsupported shell: ", shellName)

		ui.Message.Warnf(
			"Shell '%s' is not automatically supported. Please add %s to your PATH manually.",
			ui.Message.Accent(shellName),
			ui.Message.Accent(GetGlobalBinDir()),
		)

		return nil
	}

	log.Debug("Using rcFile: ", rcFile)
	log.Debug("Export command: ", exportCmd)

	// Display the diff of the changes that will be applied.
	//
	// The "diff" is two lines — a comment and an export — that
	// would be appended to the user's rc file. We render it
	// inline (no Panel) because the panel border on a 2-line
	// diff is more visual weight than the data warrants. The
	// "+" prefix is styled with ui.Message.Success (green) so
	// the user reads the addition as "what will be added" at
	// a glance; the path inside the export line is Accent
	// (primary) so the new path stands out as the actual
	// data.
	ui.Message.Infof(
		"The following diff will be applied to %s:",
		ui.Message.Accent(rcFile),
	)

	_, _ = fmt.Fprintf(
		os.Stdout,
		"  %s %s\n  %s export PATH=\"$PATH:%s\"\n",
		ui.Message.Success("+"),
		exportCmdComment,
		ui.Message.Success("+"),
		ui.Message.Accent(GetGlobalBinDir()),
	)

	// Prompt the user for confirmation.
	//
	// ConfirmScriptable auto-detects TTY vs piped input (see
	// ui.Picker.ConfirmScriptable). In a TTY, the user gets
	// the huh Yes/No form; in a pipe, the scriptable y/N
	// fallback. Either way, the destructive "modify my shell
	// rc file" intent is confirmed or denied cleanly.
	confirmed, err := ui.Picker.ConfirmScriptable(
		"Do you want to proceed?",
	)
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}

	if !confirmed {
		ui.Message.Infof("Aborted by user.")

		return nil
	}

	// If the rc file does not exist, create it with the export command.
	_, statErr := os.Stat(rcFile)
	switch {
	case os.IsNotExist(statErr):
		log.Debug("Creating new rcFile")

		err := os.WriteFile(
			rcFile,
			[]byte(exportCmdComment+"\n"+exportCmd+"\n"),
			constants.FilePerm,
		)
		if err != nil {
			return fmt.Errorf("failed to create %s: %w", rcFile, err)
		}
	case statErr != nil:
		return fmt.Errorf("failed to stat %s: %w", rcFile, statErr)
	default:
		// Otherwise, append the export command if it is not already present.
		log.Debug("Appending to existing rcFile")

		data, err := os.ReadFile(rcFile)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", rcFile, err)
		}

		// Check if the global bin directory is already in PATH
		globalBinDir := GetGlobalBinDir()
		if !rcFileContainsPathComponent(string(data), globalBinDir) {
			file, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY, constants.FilePerm)
			if err != nil {
				return fmt.Errorf("failed to open %s: %w", rcFile, err)
			}

			defer func() {
				err := file.Close()
				if err != nil {
					log.Errorf("Failed to close %s: %v", rcFile, err)
				}
			}()

			_, err = file.WriteString("\n" + exportCmdComment + "\n" + exportCmd + "\n")
			if err != nil {
				return fmt.Errorf("failed to update %s: %w", rcFile, err)
			}
		}
	}

	ui.Message.Successf(
		"Done applying changes to %s.",
		ui.Message.Accent(rcFile),
	)
	ui.Message.Warnf(
		"Please restart your terminal or source %s to apply changes.",
		ui.Message.Accent(rcFile),
	)

	return nil
}

// init registers the pathCmd with the root command.
func init() {
	rootCmd.AddCommand(pathCmd)
}

// rcFileContainsPathComponent reports whether target appears in
// content as a distinct path component. Using a raw strings.Contains
// is wrong: if the rc file has a different path of which target is
// a prefix or substring (e.g. target=/home/u/.local/bin, rc has
// `export PATH="$PATH:/home/u/.local/bin-extra"`) or if target
// appears in a comment, the substring check would return true and
// we'd skip appending, leaving the user without a working PATH
// entry.
//
// The correct check requires target to be delimited on both sides
// by characters that cannot extend a path component. We treat
// ASCII letters/digits, '_', '-', and '.' as path-component
// characters (delimiters are everything else, including '/', ' ',
// '"', "'", '$', ':', '=', and the string boundaries). This
// matches the same set of characters that are valid in PATH
// components and in the contents of typical rc-file PATH
// assignments.
//
// The check iterates line-by-line and only considers lines that
// look PATH-related (contain 'PATH' or 'path') to keep
// false-positives from comments minimal.
func rcFileContainsPathComponent(content, target string) bool {
	if target == "" {
		return false
	}

	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		if !strings.Contains(trimmed, "PATH") && !strings.Contains(trimmed, "path") {
			continue
		}

		if lineHasPathComponent(trimmed, target) {
			return true
		}
	}

	return false
}

// lineHasPathComponent reports whether target appears in line
// bounded by non-path-component characters (or string boundaries)
// on both sides.
func lineHasPathComponent(line, target string) bool {
	if target == "" {
		return false
	}

	for idx := strings.Index(line, target); idx >= 0; idx = nextIndex(line, target, idx) {
		beforeOK := idx == 0 || !isPathComponentByte(line[idx-1])
		afterIdx := idx + len(target)

		afterOK := afterIdx >= len(line) || !isPathComponentByte(line[afterIdx])
		if beforeOK && afterOK {
			return true
		}
	}

	return false
}

// nextIndex returns the next index in line at or after 'from'
// where target appears. It is used to advance past the
// occurrence we just inspected.
func nextIndex(line, target string, from int) int {
	if from < 0 {
		return strings.Index(line, target)
	}

	// Move one byte past the previous match's start so we don't
	// re-match at the same position.
	start := from + 1
	if start >= len(line) {
		return -1
	}

	rel := strings.Index(line[start:], target)
	if rel < 0 {
		return -1
	}

	return start + rel
}

// isPathComponentByte reports whether ch is a byte that can extend
// a path component. Matches the ASCII letters/digits, '_', '-',
// and '.' that show up in real filesystem paths and in
// shell-tokenized PATH entries; everything else is treated as a
// delimiter.
func isPathComponentByte(chr byte) bool {
	switch {
	case chr >= 'a' && chr <= 'z':
		return true
	case chr >= 'A' && chr <= 'Z':
		return true
	case chr >= '0' && chr <= '9':
		return true
	case chr == '_' || chr == '-' || chr == '.':
		return true
	default:
		return false
	}
}

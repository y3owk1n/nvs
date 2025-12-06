package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/constants"
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
//
//nolint:funlen
func RunPath(_ *cobra.Command, _ []string) error {
	logrus.Debug("Running path command")

	var err error

	// On Windows, automatic PATH modifications are not implemented.
	if runtime.GOOS == constants.WindowsOS {
		// Use GetGlobalBinDir() to get the path
		nvimBinDir := filepath.Join(GetGlobalBinDir(), "nvim", "bin")

		logrus.Debug("Detected Windows OS")

		_, err = fmt.Fprintf(os.Stdout,
			"%s %s\n",
			ui.WarningIcon(),
			ui.WhiteText("Automatic PATH setup is not implemented for Windows."),
		)
		if err != nil {
			logrus.Warnf("Failed to write to stdout: %v", err)
		}

		_, err = fmt.Fprintf(os.Stdout,
			"%s %s\n",
			ui.InfoIcon(),
			ui.WhiteText(
				fmt.Sprintf(
					"Please add %s to your PATH environment variable manually.",
					ui.CyanText(nvimBinDir),
				),
			),
		)
		if err != nil {
			logrus.Warnf("Failed to write to stdout: %v", err)
		}

		return nil
	}

	// Check if the global binary directory is already in the PATH.
	pathEnv := os.Getenv("PATH")
	logrus.Debug("Current PATH: ", pathEnv)

	// Check if GetGlobalBinDir() is already in PATH
	pathSeparator := string(os.PathListSeparator)
	paths := strings.Split(pathEnv, pathSeparator)

	found := false
	for _, p := range paths {
		if filepath.Clean(p) == filepath.Clean(GetGlobalBinDir()) {
			found = true

			break
		}
	}

	if found {
		logrus.Debugf("PATH already contains %s", GetGlobalBinDir())

		_, err = fmt.Fprintf(os.Stdout,
			"%s %s\n",
			ui.InfoIcon(),
			ui.WhiteText(
				fmt.Sprintf("Your PATH already contains %s.", ui.CyanText(GetGlobalBinDir())),
			),
		)
		if err != nil {
			logrus.Warnf("Failed to write to stdout: %v", err)
		}

		return nil
	}

	// Determine the user's shell; default to /bin/bash if not set.
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
		// Verify the default shell exists
		_, err := os.Stat(shell)
		if os.IsNotExist(err) {
			logrus.Warnf(
				"Default shell %s does not exist, PATH setup may not work correctly",
				shell,
			)
		}
	}

	// If running in a Nix-managed shell, advise manual configuration.
	isNixShell := os.Getenv("NIX_SHELL") != "" || strings.Contains(shell, "/nix/store")
	if isNixShell {
		logrus.Debug("Detected Nix shell environment")

		_, err = fmt.Fprintf(os.Stdout,
			"%s %s\n",
			ui.WarningIcon(),
			ui.WhiteText(
				"It appears your shell is managed by Nix. Automatic PATH modifications may not work as expected.",
			),
		)
		if err != nil {
			logrus.Warnf("Failed to write to stdout: %v", err)
		}

		_, err = fmt.Fprintf(os.Stdout,
			"%s %s\n",
			ui.InfoIcon(),
			ui.WhiteText(
				fmt.Sprintf(
					"Please update your Nix configuration manually to include %s in your PATH.",
					ui.CyanText(GetGlobalBinDir()),
				),
			),
		)
		if err != nil {
			logrus.Warnf("Failed to write to stdout: %v", err)
		}

		return nil
	}

	logrus.Debug("Detected shell: ", shell)

	// Get the base name of the shell executable (e.g. bash, zsh, fish).
	shellName := filepath.Base(shell)
	logrus.Debug("Shell name: ", shellName)

	// Determine the rc file path and export command based on the shell.
	var rcFile, exportCmd string

	exportCmdComment := "# Added by nvs"

	// Get home directory, preferring HOME env var but falling back to os.UserHomeDir
	home := os.Getenv("HOME")
	if home == "" {
		var err error

		home, err = os.UserHomeDir()
		if err != nil {
			logrus.Warnf("Failed to get home directory: %v", err)

			_, err = fmt.Fprintf(
				os.Stdout,
				"%s %s\n",
				ui.WarningIcon(),
				ui.WhiteText(
					"Cannot determine home directory. Please set HOME environment variable.",
				),
			)
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

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
		logrus.Debug("Unsupported shell: ", shellName)

		_, err = fmt.Fprintf(os.Stdout,
			"%s %s\n",
			ui.WarningIcon(),
			ui.WhiteText(
				fmt.Sprintf(
					"Shell '%s' is not automatically supported. Please add %s to your PATH manually.",
					ui.CyanText(shellName),
					ui.CyanText(GetGlobalBinDir()),
				),
			),
		)
		if err != nil {
			logrus.Warnf("Failed to write to stdout: %v", err)
		}

		return nil
	}

	logrus.Debug("Using rcFile: ", rcFile)
	logrus.Debug("Export command: ", exportCmd)

	// Display the diff of the changes that will be applied.
	_, err = fmt.Fprintf(os.Stdout,
		"%s %s\n\n",
		ui.InfoIcon(),
		ui.WhiteText(
			fmt.Sprintf("The following diff will be applied to %s:", ui.CyanText(rcFile)),
		),
	)
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	_, err = fmt.Fprintf(
		os.Stdout,
		"%s\n",
		ui.GreenText(fmt.Sprintf("+ %s\n+ %s", exportCmdComment, exportCmd)),
	)
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	// Prompt the user for confirmation.
	_, err = fmt.Fprintf(
		os.Stdout,
		"\n%s %s ",
		ui.PromptIcon(),
		"Do you want to proceed? (y/N): ",
	)
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	reader := bufio.NewReader(os.Stdin)

	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(strings.ToLower(input))
	logrus.Debug("User input: ", input)

	if input != "y" {
		_, err = fmt.Fprintf(
			os.Stdout,
			"%s %s\n",
			ui.InfoIcon(),
			ui.WhiteText("Aborted by user."),
		)
		if err != nil {
			logrus.Warnf("Failed to write to stdout: %v", err)
		}

		return nil
	}

	// If the rc file does not exist, create it with the export command.
	_, statErr := os.Stat(rcFile)
	switch {
	case os.IsNotExist(statErr):
		logrus.Debug("Creating new rcFile")

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
		logrus.Debug("Appending to existing rcFile")

		data, err := os.ReadFile(rcFile)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", rcFile, err)
		}

		// Check if the global bin directory is already in PATH
		globalBinDir := GetGlobalBinDir()
		if !strings.Contains(string(data), globalBinDir) {
			file, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY, constants.FilePerm)
			if err != nil {
				return fmt.Errorf("failed to open %s: %w", rcFile, err)
			}

			defer func() {
				err := file.Close()
				if err != nil {
					logrus.Errorf("Failed to close %s: %v", rcFile, err)
				}
			}()

			_, err = file.WriteString("\n" + exportCmdComment + "\n" + exportCmd + "\n")
			if err != nil {
				return fmt.Errorf("failed to update %s: %w", rcFile, err)
			}
		}
	}

	_, err = fmt.Fprintf(
		os.Stdout,
		"%s %s\n",
		ui.SuccessIcon(),
		ui.WhiteText(
			fmt.Sprintf("Done applying changes to %s:", ui.CyanText(rcFile)),
		),
	)
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	_, err = fmt.Fprintf(os.Stdout,
		"%s Please restart your terminal or source %s to apply changes.\n",
		ui.WarningIcon(),
		ui.CyanText(rcFile),
	)
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	return nil
}

// init registers the pathCmd with the root command.
func init() {
	rootCmd.AddCommand(pathCmd)
}

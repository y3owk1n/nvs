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
	"github.com/y3owk1n/nvs/pkg/utils"
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
	Run: func(cmd *cobra.Command, args []string) {
		logrus.Debug("Running path command")

		// On Windows, automatic PATH modifications are not implemented.
		if runtime.GOOS == "windows" {
			nvimBinDir := filepath.Join(globalBinDir, "nvim", "bin")

			logrus.Debug("Detected Windows OS")
			fmt.Printf("%s %s\n", utils.WarningIcon(), utils.WhiteText("Automatic PATH setup is not implemented for Windows."))
			fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText(fmt.Sprintf("Please add %s to your PATH environment variable manually.", utils.CyanText(nvimBinDir))))
			return
		}

		// Check if the global binary directory is already in the PATH.
		pathEnv := os.Getenv("PATH")
		logrus.Debug("Current PATH: ", pathEnv)
		if strings.Contains(pathEnv, globalBinDir) {
			logrus.Debug("PATH already contains globalBinDir")
			fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText(fmt.Sprintf("Your PATH already contains %s.", utils.CyanText(globalBinDir))))
			return
		}

		// If running in a Nix-managed shell, advise manual configuration.
		if os.Getenv("NIX_SHELL") != "" {
			logrus.Debug("Detected Nix shell environment")
			fmt.Printf("%s %s\n", utils.WarningIcon(), utils.WhiteText("It appears your shell is managed by Nix. Automatic PATH modifications may not work as expected."))
			fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText(fmt.Sprintf("Please update your Nix configuration manually to include %s in your PATH.", utils.CyanText(globalBinDir))))
			return
		}

		// Determine the user's shell; default to /bin/bash if not set.
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/bash"
		}
		logrus.Debug("Detected shell: ", shell)

		// Get the base name of the shell executable (e.g. bash, zsh, fish).
		shellName := filepath.Base(shell)
		logrus.Debug("Shell name: ", shellName)

		// Determine the rc file path and export command based on the shell.
		var rcFile, exportCmd string
		exportCmdComment := "# Added by nvs"

		switch shellName {
		case "bash", "zsh":
			rcFile = filepath.Join(os.Getenv("HOME"), fmt.Sprintf(".%src", shellName))
			exportCmd = fmt.Sprintf("export PATH=\"$PATH:%s\"", globalBinDir)
		case "fish":
			rcFile = filepath.Join(os.Getenv("HOME"), ".config", "fish", "config.fish")
			exportCmd = fmt.Sprintf("set -gx PATH $PATH %s", globalBinDir)
		default:
			logrus.Debug("Unsupported shell: ", shellName)
			fmt.Printf("%s %s\n", utils.WarningIcon(), utils.WhiteText(fmt.Sprintf("Shell '%s' is not automatically supported. Please add %s to your PATH manually.", utils.CyanText(shellName), utils.CyanText(globalBinDir))))
			return
		}

		logrus.Debug("Using rcFile: ", rcFile)
		logrus.Debug("Export command: ", exportCmd)

		// If the shell is managed by Nix, check if the rc file already contains the PATH setting.
		if strings.Contains(shell, "/nix/store") {
			logrus.Debug("Detected Nix-managed shell")
			if data, err := os.ReadFile(rcFile); err == nil {
				if strings.Contains(string(data), globalBinDir) {
					fmt.Printf("%s %s\n", utils.WarningIcon(), utils.WhiteText(fmt.Sprintf("%s already contains the PATH setting.", utils.CyanText(rcFile))))
				} else {
					fmt.Printf("%s %s\n", utils.WarningIcon(), utils.WhiteText(fmt.Sprintf("Your shell (%s) is managed by Nix and %s does not appear to contain the PATH setting.", utils.CyanText(shell), utils.CyanText(rcFile))))
					fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText(fmt.Sprintf("Please update your Nix configuration manually to include %s in your PATH.", utils.CyanText(globalBinDir))))
				}
			} else {
				logrus.Errorf("Unable to read %s: %v", rcFile, err)
			}
			return
		}

		// Display the diff of the changes that will be applied.
		fmt.Printf("%s %s\n\n", utils.InfoIcon(), utils.WhiteText(fmt.Sprintf("The following diff will be applied to %s:", utils.CyanText(rcFile))))
		fmt.Printf("%s\n", utils.GreenText(fmt.Sprintf("+ %s\n+ %s", exportCmdComment, exportCmd)))

		// Prompt the user for confirmation.
		fmt.Printf("\n%s %s ", utils.PromptIcon(), "Do you want to proceed? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			logrus.Fatalf("Failed to read input: %v", err)
		}
		input = strings.TrimSpace(strings.ToLower(input))
		logrus.Debug("User input: ", input)
		if input != "y" {
			fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText("Aborted by user."))
			return
		}

		// If the rc file does not exist, create it with the export command.
		if _, err := os.Stat(rcFile); os.IsNotExist(err) {
			logrus.Debug("Creating new rcFile")
			if err := os.WriteFile(rcFile, []byte(exportCmdComment+"\n"+exportCmd+"\n"), 0644); err != nil {
				logrus.Fatalf("Failed to create %s: %v", rcFile, err)
			}
		} else {
			// Otherwise, append the export command if it is not already present.
			logrus.Debug("Appending to existing rcFile")
			data, err := os.ReadFile(rcFile)
			if err != nil {
				logrus.Fatalf("Failed to read %s: %v", rcFile, err)
			}
			if !strings.Contains(string(data), globalBinDir) {
				f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					logrus.Fatalf("Failed to open %s: %v", rcFile, err)
				}
				defer func() {
					if err := f.Close(); err != nil {
						logrus.Errorf("Failed to close %s: %v", rcFile, err)
					}
				}()
				if _, err := f.WriteString("\n" + exportCmdComment + "\n" + exportCmd + "\n"); err != nil {
					logrus.Fatalf("Failed to update %s: %v", rcFile, err)
				}
			}
		}

		fmt.Printf("%s %s\n", utils.SuccessIcon(), utils.WhiteText(fmt.Sprintf("Done applying changes to %s:", utils.CyanText(rcFile))))
		fmt.Printf("%s Please restart your terminal or source %s to apply changes.\n", utils.WarningIcon(), utils.CyanText(rcFile))
	},
}

// init registers the pathCmd with the root command.
func init() {
	rootCmd.AddCommand(pathCmd)
}

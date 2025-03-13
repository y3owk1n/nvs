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

var pathCmd = &cobra.Command{
	Use:   "path",
	Short: "Automatically add the global binary directory to your PATH",
	Run: func(cmd *cobra.Command, args []string) {
		logrus.Debug("Running path command")

		if runtime.GOOS == "windows" {
			logrus.Debug("Detected Windows OS")
			fmt.Printf("%s %s\n", utils.WarningIcon(), utils.WhiteText("Automatic PATH setup is not implemented for Windows."))
			fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText(fmt.Sprintf("Please add %s to your PATH environment variable manually.", utils.CyanText(globalBinDir))))
			return
		}

		pathEnv := os.Getenv("PATH")
		logrus.Debug("Current PATH: ", pathEnv)
		if strings.Contains(pathEnv, globalBinDir) {
			logrus.Debug("PATH already contains globalBinDir")
			fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText(fmt.Sprintf("Your PATH already contains %s.", utils.CyanText(globalBinDir))))
			return
		}

		if os.Getenv("NIX_SHELL") != "" {
			logrus.Debug("Detected Nix shell environment")
			fmt.Printf("%s %s\n", utils.WarningIcon(), utils.WhiteText("It appears your shell is managed by Nix. Automatic PATH modifications may not work as expected."))
			fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText(fmt.Sprintf("Please update your Nix configuration manually to include %s in your PATH.", utils.CyanText(globalBinDir))))
			return
		}

		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/bash"
		}
		logrus.Debug("Detected shell: ", shell)

		shellName := filepath.Base(shell)
		logrus.Debug("Shell name: ", shellName)

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

		fmt.Printf("%s %s\n\n", utils.InfoIcon(), utils.WhiteText(fmt.Sprintf("The following diff will be applied to %s:", utils.CyanText(rcFile))))
		fmt.Printf("%s\n", utils.GreenText(fmt.Sprintf("+ %s\n+ %s", exportCmdComment, exportCmd)))

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

		if _, err := os.Stat(rcFile); os.IsNotExist(err) {
			logrus.Debug("Creating new rcFile")
			if err := os.WriteFile(rcFile, []byte(exportCmdComment+"\n"+exportCmd+"\n"), 0644); err != nil {
				logrus.Fatalf("Failed to create %s: %v", rcFile, err)
			}
		} else {
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
				defer f.Close()
				if _, err := f.WriteString("\n" + exportCmdComment + "\n" + exportCmd + "\n"); err != nil {
					logrus.Fatalf("Failed to update %s: %v", rcFile, err)
				}
			}
		}
		fmt.Printf("%s %s\n", utils.SuccessIcon(), utils.WhiteText(fmt.Sprintf("Done applying changes to %s:", utils.CyanText(rcFile))))
		fmt.Printf("%s Please restart your terminal or source %s to apply changes.\n", utils.WarningIcon(), utils.CyanText(rcFile))
	},
}

func init() {
	rootCmd.AddCommand(pathCmd)
}

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/releases"
	"github.com/y3owk1n/nvs/pkg/utils"
)

// uninstallCmd represents the "uninstall" command (aliases: rm, remove, un).
// It uninstalls a specific installed Neovim version by removing its directory.
// If the version to be uninstalled is currently active, it prompts for confirmation before
// removing the "current" symlink and then proceeds to uninstall the version.
// After uninstalling the current version, if other versions exist, it allows the user to switch
// to a different installed version.
//
// Example usage:
//
//	nvs uninstall v0.6.0
//	nvs rm stable
var uninstallCmd = &cobra.Command{
	Use:     "uninstall <version>",
	Aliases: []string{"rm", "remove", "un"},
	Short:   "Uninstall a specific version",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		logrus.Debug("Running uninstall command")

		// Normalize the provided version argument (e.g. add "v" prefix if missing)
		versionArg := releases.NormalizeVersion(args[0])
		logrus.Debug("Normalized version: ", versionArg)

		// Compute the path where the version is installed.
		versionPath := filepath.Join(versionsDir, versionArg)
		logrus.Debug("Computed version path: ", versionPath)

		// Check if the version is installed.
		if !utils.IsInstalled(versionsDir, versionArg) {
			logrus.Fatalf("Version %s is not installed", versionArg)
		}

		// Determine the path of the "current" symlink.
		currentSymlink := filepath.Join(versionsDir, "current")

		// Check if the version to uninstall is currently active.
		isCurrent := false
		if current, err := utils.GetCurrentVersion(versionsDir); err == nil {
			if current == versionArg {
				isCurrent = true
			}
		}

		// If the version is currently active, prompt for confirmation.
		if isCurrent {
			fmt.Printf("%s The version %s is currently in use. Do you really want to uninstall it? (y/N): ", utils.WarningIcon(), utils.CyanText(versionArg))
			reader := bufio.NewReader(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				logrus.Fatalf("Failed to read input: %v", err)
			}
			input = strings.TrimSpace(input)
			if strings.ToLower(input) != "y" {
				fmt.Println(utils.InfoIcon(), utils.WhiteText("Aborted uninstall."))
				logrus.Debug("Uninstall cancelled by user")
				return
			}

			logrus.Debugf("User confirmed removal of current version %s", versionArg)
			// Remove the current symlink/junction if uninstalling the active version.
			info, err := os.Lstat(currentSymlink)
			if err != nil {
				logrus.Fatalf("Failed to lstat current symlink: %v", err)
			}

			if info.Mode()&os.ModeSymlink != 0 {
				// POSIX symlink
				if err := os.Remove(currentSymlink); err != nil {
					logrus.Fatalf("Failed to remove current symlink: %v", err)
				}
				logrus.Debug("Removed current symlink")
			} else if info.IsDir() {
				// Likely a Windows junction — requires RemoveAll
				if err := os.RemoveAll(currentSymlink); err != nil {
					logrus.Fatalf("Failed to remove current junction: %v", err)
				}
				logrus.Debug("Removed current junction")
			} else {
				logrus.Warnf("Current entry is neither a symlink nor a directory: %s", currentSymlink)
			}
		}

		logrus.Debug("Version is installed, proceeding with removal")
		// Remove the version's directory.
		if err := os.RemoveAll(versionPath); err != nil {
			logrus.Fatalf("Failed to uninstall version %s: %v", versionArg, err)
		}

		successMsg := fmt.Sprintf("Uninstalled version: %s", utils.CyanText(versionArg))
		logrus.Debug(successMsg)
		fmt.Printf("%s %s\n", utils.SuccessIcon(), utils.WhiteText(successMsg))

		// If the uninstalled version was the current version,
		// prompt the user to switch to a different installed version.
		if isCurrent {
			versions, err := utils.ListInstalledVersions(versionsDir)
			if err != nil {
				logrus.Fatalf("Error listing versions: %v", err)
			}

			var availableVersions []string
			// Exclude the "current" symlink entry.
			for _, entry := range versions {
				if entry != "current" {
					availableVersions = append(availableVersions, entry)
				}
			}

			if len(availableVersions) == 0 {
				fmt.Printf("%s %s\n", utils.WarningIcon(), utils.WhiteText("No other versions available. Your current version has been unset."))
			} else {
				logrus.Debugf("Switchable Installed Neovim Versions: %v", availableVersions)
				prompt := promptui.Select{
					Label: "Switchable Installed Neovim Versions",
					Items: availableVersions,
				}

				logrus.Debug("Displaying selection prompt")
				_, selectedVersion, err := prompt.Run()
				if err != nil {
					if err == promptui.ErrInterrupt {
						logrus.Debug("User cancelled selection")
						fmt.Printf("%s %s\n", utils.WarningIcon(), utils.WhiteText("Selection cancelled."))
						return
					}
					logrus.Fatalf("Prompt failed: %v", err)
				}

				// Use the selected version as the new current version.
				if err := utils.UseVersion(selectedVersion, currentSymlink, versionsDir, globalBinDir); err != nil {
					logrus.Fatalf("%v", err)
				}
			}
		}
	},
}

// init registers the uninstallCmd with the root command.
func init() {
	rootCmd.AddCommand(uninstallCmd)
}

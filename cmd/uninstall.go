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

var uninstallCmd = &cobra.Command{
	Use:     "uninstall <version>",
	Aliases: []string{"rm", "remove", "un"},
	Short:   "Uninstall a specific version",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		logrus.Debug("Running uninstall command")

		versionArg := releases.NormalizeVersion(args[0])
		logrus.Debug("Normalized version: ", versionArg)

		versionPath := filepath.Join(versionsDir, versionArg)
		logrus.Debug("Computed version path: ", versionPath)

		if !utils.IsInstalled(versionsDir, versionArg) {
			logrus.Fatalf("Version %s is not installed", versionArg)
		}

		currentSymlink := filepath.Join(versionsDir, "current")

		isCurrent := false
		if current, err := utils.GetCurrentVersion(versionsDir); err == nil {
			if current == versionArg {
				isCurrent = true
			}
		}

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
			if err := os.Remove(currentSymlink); err != nil {
				logrus.Fatalf("Failed to remove current symlink: %v", err)
			}
		}

		logrus.Debug("Version is installed, proceeding with removal")
		if err := os.RemoveAll(versionPath); err != nil {
			logrus.Fatalf("Failed to uninstall version %s: %v", versionArg, err)
		}

		successMsg := fmt.Sprintf("Uninstalled version: %s", utils.CyanText(versionArg))
		logrus.Debug(successMsg)
		fmt.Printf("%s %s\n", utils.SuccessIcon(), utils.WhiteText(successMsg))

		if isCurrent {
			versions, err := utils.ListInstalledVersions(versionsDir)
			if err != nil {
				logrus.Fatalf("Error listing versions: %v", err)
			}

			var availableVersions []string
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

				if err := utils.UseVersion(selectedVersion, currentSymlink, versionsDir, globalBinDir); err != nil {
					logrus.Fatalf("%v", err)
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}

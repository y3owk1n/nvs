package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}

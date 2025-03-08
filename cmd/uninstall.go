package cmd

import (
	"fmt"
	"os"
	"path/filepath"

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
		versionArg := releases.NormalizeVersion(args[0])
		versionPath := filepath.Join(versionsDir, versionArg)

		if !utils.IsInstalled(versionsDir, versionArg) {
			logrus.Fatalf("Version %s is not installed", versionArg)
		}

		if err := os.RemoveAll(versionPath); err != nil {
			logrus.Fatalf("Failed to uninstall version %s: %v", versionArg, err)
		}

		successMsg := fmt.Sprintf("Uninstalled version %s", versionArg)
		fmt.Printf("%s %s\n", utils.SuccessIcon(), utils.WhiteText(successMsg))
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}

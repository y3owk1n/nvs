package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/utils"
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset all data (remove symlinks, downloaded versions, cache, etc.) except bin structure",
	Long:  "WARNING: This command will delete all data in ~/.nvs including items inside the bin directory, but will preserve the bin directory structure. Use with caution.",
	Run: func(cmd *cobra.Command, args []string) {
		logrus.Debug("Starting reset command")

		home, err := os.UserHomeDir()
		if err != nil {
			logrus.Fatalf("Failed to get home directory: %v", err)
		}
		baseDir := filepath.Join(home, ".nvs")
		logrus.Debugf("Base directory resolved: %s", baseDir)

		fmt.Printf("%s %s\n\n", utils.WarningIcon(), fmt.Sprintf("WARNING: This will delete all data in %s, including items inside the bin directory, but will preserve the bin directory structure.", utils.CyanText(baseDir)))
		fmt.Printf("%s ", "Are you sure? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			logrus.Fatalf("Failed to read input: %v", err)
		}
		input = strings.TrimSpace(input)
		if strings.ToLower(input) != "y" {
			fmt.Println(utils.InfoIcon(), utils.WhiteText("Reset cancelled."))
			logrus.Debug("Reset cancelled by user")
			return
		}

		entries, err := os.ReadDir(baseDir)
		if err != nil {
			logrus.Fatalf("Failed to read base directory: %v", err)
		}

		logrus.Debug("Starting directory cleanup")
		for _, entry := range entries {
			fullPath := filepath.Join(baseDir, entry.Name())
			if entry.Name() == "bin" {
				logrus.Debugf("Clearing contents of bin directory: %s", fullPath)
				if err := utils.ClearDirectory(fullPath); err != nil {
					logrus.Fatalf("Failed to clear bin directory: %v", err)
				}
			} else {
				logrus.Debugf("Removing directory: %s", fullPath)
				if err := os.RemoveAll(fullPath); err != nil {
					logrus.Fatalf("Failed to remove %s: %v", fullPath, err)
				}
			}
		}
		fmt.Println(utils.SuccessIcon(), utils.WhiteText("Reset successful. All data has been cleared, but the bin structure has been preserved."))
		logrus.Debug("Reset completed successfully")
	},
}

func init() {
	rootCmd.AddCommand(resetCmd)
}

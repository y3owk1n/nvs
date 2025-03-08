package cmd

import (
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
		home, err := os.UserHomeDir()
		if err != nil {
			logrus.Fatalf("Failed to get home directory: %v", err)
		}
		baseDir := filepath.Join(home, ".nvs")
		fmt.Printf("WARNING: This will delete all data in %s, including items inside the bin directory, but will preserve the bin directory structure. Are you sure? (y/N): ", baseDir)
		var answer string
		_, err = fmt.Scanln(&answer)
		if err != nil {
			logrus.Fatalf("Failed to read input: %v", err)
		}
		if strings.ToLower(answer) != "y" {
			fmt.Println("Reset cancelled.")
			return
		}

		entries, err := os.ReadDir(baseDir)
		if err != nil {
			logrus.Fatalf("Failed to read base directory: %v", err)
		}

		for _, entry := range entries {
			fullPath := filepath.Join(baseDir, entry.Name())
			if entry.Name() == "bin" {
				// Clear the contents of the bin directory while preserving the directory itself.
				if err := utils.ClearDirectory(fullPath); err != nil {
					logrus.Fatalf("Failed to clear bin directory: %v", err)
				}
			} else {
				// Remove all other entries.
				if err := os.RemoveAll(fullPath); err != nil {
					logrus.Fatalf("Failed to remove %s: %v", fullPath, err)
				}
			}
		}
		logrus.Info("Reset successful. All data has been cleared, but the bin structure has been preserved.")
	},
}

func init() {
	rootCmd.AddCommand(resetCmd)
}

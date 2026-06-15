package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/domain/vtypes"
	"github.com/y3owk1n/nvs/internal/log"
	"github.com/y3owk1n/nvs/internal/ui"
)

// listCmd represents the "list" command (aliases: ls).
// It lists all installed Neovim versions found in the versions directory and marks the current active version.
// If no versions are installed, it informs the user.
//
// Example usage:
//
//	nvs list
//	nvs ls
var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List installed versions",
	RunE:    RunList,
}

// versionInfo is the structured result of RunList. The shape
// is the public JSON contract (TestRunList_JSON asserts on
// it), so the struct fields and their JSON tags must not
// change shape.
type versionInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Type   string `json:"type"`
}

// RunList executes the list command.
func RunList(cmd *cobra.Command, _ []string) error {
	log.Debug("Executing list command")

	// Retrieve installed versions from the version service.
	versions, err := GetVersionService().List()
	if err != nil {
		return fmt.Errorf("error listing versions: %w", err)
	}

	log.Debugf("Found %d installed versions", len(versions))

	// If no versions are installed, display a message and exit.
	if len(versions) == 0 {
		ui.Message.Infof("No installed versions.")

		log.Debug("No installed versions found")

		return nil
	}

	// Get the current active version.
	current, err := GetVersionService().Current()
	if err != nil {
		log.Warn("No current version set or unable to determine the current version")
	} else {
		log.Debugf("Current version: %s", current.Name())
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		return renderListJSON(versions, current)
	}

	return renderListText(versions, current)
}

// renderListJSON emits the --json contract: an object with
// "versions" (one VersionInfo per installed version, with
// "status" set to "current" or "installed"). It is preserved
// byte-for-byte from the pre-refactor implementation.
func renderListJSON(versions []vtypes.Version, current vtypes.Version) error {
	infos := make([]versionInfo, 0, len(versions))
	for _, version := range versions {
		status := "installed"
		if current.Name() != "" && version.Name() == current.Name() {
			status = "current"
		}

		infos = append(infos, versionInfo{
			Name:   version.Name(),
			Status: status,
			Type:   version.Type().String(),
		})
	}

	return outputJSON(map[string]any{"versions": infos})
}

// renderListText renders the human-readable list view: a
// banner, a one-line summary, and a data table with one row
// per installed version. The current version is rendered
// with an "→ " prefix and the primary color so the user
// can spot it at a glance.
func renderListText(versions []vtypes.Version, current vtypes.Version) error {
	currentName := current.Name()

	tbl := ui.Table.New("VERSION", "STATUS")

	for _, version := range versions {
		isCurrent := currentName != "" && version.Name() == currentName

		if isCurrent {
			tbl.Row(
				ui.Message.Highlight("→ "+version.Name()),
				ui.Message.Highlight("Current"),
			)
		} else {
			tbl.Row(
				ui.Message.Text(version.Name()),
				ui.Message.Text("Installed"),
			)
		}
	}

	_, _ = fmt.Fprint(os.Stdout, ui.Banner.Logo())
	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprint(os.Stdout, tbl.Render(ui.Style.Palette()))

	return nil
}

// init registers the listCmd with the root command.
func init() {
	listCmd.Flags().Bool("json", false, "Output in JSON format")
	rootCmd.AddCommand(listCmd)
}

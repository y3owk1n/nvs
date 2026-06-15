package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/log"
	"github.com/y3owk1n/nvs/internal/ui"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check your system for potential problems",
	Long:  "Check your system for potential problems with nvs installation and environment.",
	RunE:  RunDoctor,
}

// CheckResult represents the result of a system check.
type CheckResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// checkOutcome is the in-memory representation of a check
// after it has run. The Status field mirrors what is emitted
// in --json mode (so the public JSON contract is preserved),
// and the Error and Warning fields drive the text rendering
// (a row's icon is Error-driven; a sub-line is Warning- or
// Error-driven).
type checkOutcome struct {
	Name    string
	Status  string
	Error   error
	Warning string
}

// RunDoctor executes the doctor command.
func RunDoctor(cmd *cobra.Command, _ []string) error {
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		log.Warnf("Failed to read json flag: %v", err)
	}

	checks := []struct {
		name  string
		check func() (string, error)
	}{
		{"Shell", checkShell},
		{"Environment variables", checkEnvVars},
		{"PATH", checkPath},
		{"Dependencies", checkDependencies},
		{"Permissions", checkPermissions},
	}

	outcomes := make([]checkOutcome, 0, len(checks))
	for _, check := range checks {
		log.Debugf("Running doctor check: %s", check.name)

		warning, checkErr := check.check()

		outcome := checkOutcome{Name: check.name}
		if checkErr != nil {
			outcome.Status = "error"
			outcome.Error = checkErr
		} else {
			outcome.Status = "ok"
		}

		outcome.Warning = warning

		outcomes = append(outcomes, outcome)
	}

	if jsonOutput {
		return renderDoctorJSON(outcomes)
	}

	return renderDoctorText(outcomes)
}

// renderDoctorJSON emits the --json contract: an object with
// "checks" (one CheckResult per outcome) and "issues" (a list
// of "name: error" strings, one per failed check). It is
// preserved byte-for-byte from the pre-refactor implementation.
func renderDoctorJSON(outcomes []checkOutcome) error {
	results := make([]CheckResult, 0, len(outcomes))
	issues := make([]string, 0, len(outcomes))

	for _, outcome := range outcomes {
		results = append(results, CheckResult{
			Name:   outcome.Name,
			Status: outcome.Status,
		})

		if outcome.Error != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", outcome.Name, outcome.Error))
		}
	}

	jsonErr := outputJSON(map[string]any{"checks": results, "issues": issues})
	if jsonErr != nil {
		return jsonErr
	}

	if len(issues) > 0 {
		return fmt.Errorf("%w: %d issue(s)", ErrIssuesFound, len(issues))
	}

	return nil
}

// renderDoctorText renders the human-readable doctor view:
// a banner, a single section panel listing each check as a
// status row with optional indented sub-line, and a summary
// line (success or "N issues found:" + bullet list).
//
// The summary line writes to stdout and the function returns a
// non-nil error to signal the non-zero exit code (matching the
// pre-refactor behavior).
func renderDoctorText(outcomes []checkOutcome) error {
	_, _ = fmt.Fprint(os.Stdout, ui.Banner.Logo())
	_, _ = fmt.Fprint(os.Stdout, ui.Panel.Section("System health", renderDoctorBody(outcomes)))

	issues := collectDoctorIssues(outcomes)
	if len(issues) > 0 {
		ui.Message.Warnf("%d issue(s) found:", len(issues))

		for _, issue := range issues {
			ui.Message.Bulletf("%s", issue)
		}

		return fmt.Errorf("%w: %d issue(s)", ErrIssuesFound, len(issues))
	}

	ui.Message.Successf("No issues found. You are ready to go.")

	return nil
}

// renderDoctorBody builds the multi-line panel body for the
// system-health section. The icon for each row is driven by
// the outcome (Error beats Warning beats none) and the
// optional sub-line is rendered as a 4-space-indented muted
// line under the row.
func renderDoctorBody(outcomes []checkOutcome) string {
	var body strings.Builder

	for _, outcome := range outcomes {
		switch {
		case outcome.Error != nil:
			body.WriteString(ui.Message.ErrorRow(outcome.Name))
			body.WriteString(ui.Message.Detail(outcome.Error.Error()))
		case outcome.Warning != "":
			body.WriteString(ui.Message.WarnRow(outcome.Name))
			body.WriteString(ui.Message.Detail(outcome.Warning))
		default:
			body.WriteString(ui.Message.SuccessRow(outcome.Name))
		}
	}

	return body.String()
}

// collectDoctorIssues returns the "name: error" strings for
// every failed check, in check order. It is shared between the
// text and JSON paths so the issue list is identical across
// both modes.
func collectDoctorIssues(outcomes []checkOutcome) []string {
	issues := make([]string, 0, len(outcomes))

	for _, outcome := range outcomes {
		if outcome.Error != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", outcome.Name, outcome.Error))
		}
	}

	return issues
}

func checkShell() (string, error) {
	shell := DetectShell()
	if shell == "" {
		return "", ErrCouldNotDetectShell
	}

	return "", nil
}

func checkEnvVars() (string, error) {
	// Just check if we can resolve them, RunEnv logic handles defaults
	if GetVersionsDir() == "" {
		return "", ErrCouldNotResolveVersionsDir
	}

	return "", nil
}

func checkPath() (string, error) {
	binDir := GetGlobalBinDir()
	if binDir == "" {
		return "", fmt.Errorf("%w: empty global bin dir", ErrBinDirNotInPath)
	}

	path := os.Getenv("PATH")
	binClean := filepath.Clean(binDir)

	for p := range strings.SplitSeq(path, string(os.PathListSeparator)) {
		pClean := filepath.Clean(p)
		if runtime.GOOS == constants.WindowsOS {
			if strings.EqualFold(pClean, binClean) {
				return "", nil
			}
		} else if pClean == binClean {
			return "", nil
		}
	}

	return "", fmt.Errorf("%w: %s", ErrBinDirNotInPath, binDir)
}

func checkDependencies() (string, error) {
	// Base dependencies needed for general nvs operation
	baseDeps := []string{"git", "curl", "tar"}
	if runtime.GOOS == constants.WindowsOS {
		baseDeps = []string{"git", "tar"} // curl might be alias in PS
	}

	// Build dependencies needed only for building from source
	buildDeps := []string{"make", "cmake", "gettext", "ninja", "curl"}

	var (
		missingBase  []string
		missingBuild []string
	)

	// Check base dependencies

	for _, dep := range baseDeps {
		_, err := exec.LookPath(dep)
		if err != nil {
			missingBase = append(missingBase, dep)
		}
	}

	// Check build dependencies
	for _, dep := range buildDeps {
		_, err := exec.LookPath(dep)
		if err != nil {
			missingBuild = append(missingBuild, dep)
		}
	}

	// Report missing base dependencies as errors
	if len(missingBase) > 0 {
		return "", fmt.Errorf(
			"%w: Missing base dependencies: %s",
			ErrMissingDependency,
			strings.Join(missingBase, ", "),
		)
	}

	// Report missing build dependencies as warnings (don't fail the check)
	if len(missingBuild) > 0 {
		warning := "Missing build dependencies (needed for building from source): " +
			strings.Join(missingBuild, ", ")

		return warning, nil
	}

	return "", nil
}

func checkPermissions() (string, error) {
	versionsDir := GetVersionsDir()
	if versionsDir == "" {
		// Env resolution check already reports this; avoid probing CWD here.
		return "", ErrCouldNotResolveVersionsDir
	}

	dirs := []string{
		versionsDir,
		filepath.Dir(versionsDir), // Config dir
	}

	for _, dir := range dirs {
		if dir == "" {
			continue
		}

		err := os.MkdirAll(dir, constants.DirPerm)
		if err != nil {
			return "", fmt.Errorf("cannot create/write to %s: %w", dir, err)
		}

		// Probe write permission by creating a uniquely-named
		// temp file (avoids the previous collision hazard where
		// a hardcoded ".perm-test" could be clobbered by a
		// concurrent 'nvs doctor' run, or fail if an attacker
		// pre-created a read-only file at that path). We write
		// a small payload via the returned handle to exercise
		// the full open/write path, not just create.
		file, err := os.CreateTemp(dir, ".nvs-perm-*.tmp")
		if err != nil {
			return "", fmt.Errorf("cannot create/write to %s: %w", dir, err)
		}

		testFile := file.Name()

		_, writeErr := file.WriteString("test")
		closeErr := file.Close()

		defer func() {
			removeErr := os.Remove(testFile)
			if removeErr != nil && !os.IsNotExist(removeErr) {
				log.Warnf("Failed to remove temp file %s: %v", testFile, removeErr)
			}
		}()

		if writeErr != nil {
			return "", fmt.Errorf("cannot write to %s: %w", dir, writeErr)
		}

		if closeErr != nil {
			return "", fmt.Errorf("cannot close temp file in %s: %w", dir, closeErr)
		}
	}

	return "", nil
}

func init() {
	doctorCmd.Flags().Bool("json", false, "Output in JSON format")
	rootCmd.AddCommand(doctorCmd)
}

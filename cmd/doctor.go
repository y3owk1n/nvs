package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/infra/filesystem"
	"github.com/y3owk1n/nvs/internal/ui"
)

const (
	// SpinnerInterval is the interval for the spinner.
	SpinnerInterval = 100 * time.Millisecond
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check your system for potential problems",
	Long:  "Check your system for potential problems with nvs installation and environment.",
	RunE:  runDoctor,
}

func runDoctor(cmd *cobra.Command, args []string) error {
	spin := spinner.New(spinner.CharSets[14], SpinnerInterval)
	spin.Suffix = " Checking system..."
	spin.Start()

	checks := []struct {
		name  string
		check func() error
	}{
		{"OS/Arch", checkOSArch},
		{"Shell", checkShell},
		{"Environment variables", checkEnvVars},
		{"PATH", checkPath},
		{"Dependencies", checkDependencies},
		{"Permissions", checkPermissions},
	}

	var issues []string

	spin.Stop()

	_, _ = os.Stdout.Write([]byte("\n"))

	for _, check := range checks {
		_, _ = fmt.Fprintf(os.Stdout, "Checking %s... ", check.name)

		err := check.check()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stdout, "%s\n", ui.ErrorIcon())

			issues = append(issues, fmt.Sprintf("%s: %v", check.name, err))
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "%s\n", ui.SuccessIcon())
		}
	}

	_, _ = os.Stdout.Write([]byte("\n"))

	if len(issues) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "%s\n", ui.RedText("Issues found:"))

		for _, issue := range issues {
			_, _ = fmt.Fprintf(os.Stdout, "  - %s\n", issue)
		}

		return fmt.Errorf("%w: %d issue(s)", ErrIssuesFound, len(issues))
	}

	_, _ = fmt.Fprintf(os.Stdout,
		"%s\n", ui.GreenText("No issues found! You are ready to go."))

	return nil
}

func checkOSArch() error {
	if runtime.GOOS == "" || runtime.GOARCH == "" {
		return fmt.Errorf("%w: %s/%s", ErrUnknownOSArch, runtime.GOOS, runtime.GOARCH)
	}

	return nil
}

func checkShell() error {
	shell := DetectShell()
	if shell == "" {
		return ErrCouldNotDetectShell
	}

	return nil
}

func checkEnvVars() error {
	// Just check if we can resolve them, RunEnv logic handles defaults
	if GetVersionsDir() == "" {
		return ErrCouldNotResolveVersionsDir
	}

	return nil
}

func checkPath() error {
	binDir := GetGlobalBinDir()

	path := os.Getenv("PATH")
	if !strings.Contains(path, binDir) {
		return fmt.Errorf("%w: %s", ErrBinDirNotInPath, binDir)
	}

	return nil
}

func checkDependencies() error {
	deps := []string{"git", "curl", "tar"}
	if runtime.GOOS == "windows" {
		deps = []string{"git", "tar"} // curl might be alias in PS
	}

	for _, dep := range deps {
		_, err := exec.LookPath(dep)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrMissingDependency, dep)
		}
	}

	return nil
}

func checkPermissions() error {
	dirs := []string{
		GetVersionsDir(),
		filepath.Dir(GetVersionsDir()), // Config dir
	}

	for _, dir := range dirs {
		err := os.MkdirAll(dir, filesystem.DirPerm)
		if err != nil {
			return fmt.Errorf("cannot create/write to %s: %w", dir, err)
		}

		// Try writing a temp file
		testFile := filepath.Join(dir, ".perm-test")

		err = os.WriteFile(testFile, []byte("test"), FilePerm)
		if err != nil {
			return fmt.Errorf("cannot write to %s: %w", dir, err)
		}

		_ = os.Remove(testFile)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

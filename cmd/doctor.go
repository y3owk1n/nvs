package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/infra/filesystem"
	"github.com/y3owk1n/nvs/internal/ui"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check your system for potential problems",
	Long:  "Check your system for potential problems with nvs installation and environment.",
	RunE:  runDoctor,
}

func runDoctor(cmd *cobra.Command, args []string) error {
	checks := []struct {
		name  string
		check func() error
	}{
		{"Shell", checkShell},
		{"Environment variables", checkEnvVars},
		{"PATH", checkPath},
		{"Dependencies", checkDependencies},
		{"Permissions", checkPermissions},
	}

	var issues []string

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
	if binDir == "" {
		return fmt.Errorf("%w: empty global bin dir", ErrBinDirNotInPath)
	}

	path := os.Getenv("PATH")
	binClean := filepath.Clean(binDir)

	for p := range strings.SplitSeq(path, string(os.PathListSeparator)) {
		pClean := filepath.Clean(p)
		if runtime.GOOS == windows {
			if strings.EqualFold(pClean, binClean) {
				return nil
			}
		} else if pClean == binClean {
			return nil
		}
	}

	return fmt.Errorf("%w: %s", ErrBinDirNotInPath, binDir)
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
	versionsDir := GetVersionsDir()
	if versionsDir == "" {
		// Env resolution check already reports this; avoid probing CWD here.
		return ErrCouldNotResolveVersionsDir
	}

	dirs := []string{
		versionsDir,
		filepath.Dir(versionsDir), // Config dir
	}

	for _, dir := range dirs {
		if dir == "" {
			continue
		}

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

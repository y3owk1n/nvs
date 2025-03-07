package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Note: For testing LaunchNvimWithConfig, we need to abstract exec.LookPath.
// If you can modify your utils package, add the following at package scope:
//
//    var lookPath = exec.LookPath
//
// And then in LaunchNvimWithConfig, use lookPath("nvim") instead of exec.LookPath("nvim").
// For this example, we assume that change was made. If not, consider skipping that test.

// TestIsInstalled verifies that IsInstalled returns true when a directory exists.
func TestIsInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	versionDir := filepath.Join(tmpDir, "v1.0.0")
	if err := os.Mkdir(versionDir, 0755); err != nil {
		t.Fatalf("Failed to create version directory: %v", err)
	}
	if !IsInstalled(tmpDir, "v1.0.0") {
		t.Error("IsInstalled returned false for an existing version")
	}
	if IsInstalled(tmpDir, "nonexistent") {
		t.Error("IsInstalled returned true for a non-existent version")
	}
}

// TestListInstalledVersions verifies that ListInstalledVersions returns only directories that are not named "current".
func TestListInstalledVersions(t *testing.T) {
	tmpDir := t.TempDir()
	versions := []string{"v1.0.0", "v1.1.0", "current"}
	for _, v := range versions {
		dir := filepath.Join(tmpDir, v)
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %q: %v", v, err)
		}
	}
	list, err := ListInstalledVersions(tmpDir)
	if err != nil {
		t.Fatalf("ListInstalledVersions error: %v", err)
	}
	for _, v := range list {
		if v == "current" {
			t.Error("ListInstalledVersions should not include 'current'")
		}
	}
	if len(list) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(list))
	}
}

// TestUpdateSymlink verifies that UpdateSymlink creates a symlink pointing to the target.
func TestUpdateSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target")
	// Create a dummy file to act as the target.
	if err := os.WriteFile(target, []byte("dummy"), 0644); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}
	link := filepath.Join(tmpDir, "link")
	if err := UpdateSymlink(target, link); err != nil {
		t.Fatalf("UpdateSymlink error: %v", err)
	}
	resolved, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("os.Readlink error: %v", err)
	}
	if resolved != target {
		t.Errorf("Symlink points to %q; want %q", resolved, target)
	}

	// Test updating an existing symlink.
	newTarget := filepath.Join(tmpDir, "newTarget")
	if err := os.WriteFile(newTarget, []byte("new dummy"), 0644); err != nil {
		t.Fatalf("Failed to create new target file: %v", err)
	}
	if err := UpdateSymlink(newTarget, link); err != nil {
		t.Fatalf("UpdateSymlink error when updating: %v", err)
	}
	resolved, err = os.Readlink(link)
	if err != nil {
		t.Fatalf("os.Readlink error after updating: %v", err)
	}
	if resolved != newTarget {
		t.Errorf("Symlink after update points to %q; want %q", resolved, newTarget)
	}
}

// TestGetCurrentVersion creates a "current" symlink and verifies that GetCurrentVersion returns its base name.
func TestGetCurrentVersion(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a version directory.
	versionName := "v1.0.0"
	versionDir := filepath.Join(tmpDir, versionName)
	if err := os.Mkdir(versionDir, 0755); err != nil {
		t.Fatalf("Failed to create version directory: %v", err)
	}
	// Create the "current" symlink pointing to the version directory.
	currentLink := filepath.Join(tmpDir, "current")
	if err := os.Symlink(versionDir, currentLink); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}
	currentVersion, err := GetCurrentVersion(tmpDir)
	if err != nil {
		t.Fatalf("GetCurrentVersion error: %v", err)
	}
	if currentVersion != versionName {
		t.Errorf("GetCurrentVersion returned %q; want %q", currentVersion, versionName)
	}
}

// TestFindNvimBinary creates a temporary directory with a dummy executable and verifies the lookup.
func TestFindNvimBinary(t *testing.T) {
	tmpDir := t.TempDir()
	var binaryName string
	if runtime.GOOS == "windows" {
		binaryName = "nvim.exe"
	} else {
		binaryName = "nvim"
	}
	binaryPath := filepath.Join(tmpDir, binaryName)
	// Create a dummy file.
	if err := os.WriteFile(binaryPath, []byte("dummy"), 0755); err != nil {
		t.Fatalf("Failed to create dummy nvim binary: %v", err)
	}
	found := FindNvimBinary(tmpDir)
	if found == "" {
		t.Error("FindNvimBinary did not find the dummy binary")
	} else if filepath.Base(found) != binaryName {
		t.Errorf("FindNvimBinary returned %q; want %q", found, binaryName)
	}
}

// TestGetInstalledReleaseIdentifier creates a dummy version file and verifies its content is read correctly.
func TestGetInstalledReleaseIdentifier(t *testing.T) {
	tmpDir := t.TempDir()
	version := "v1.0.0"
	dir := filepath.Join(tmpDir, version)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create version directory: %v", err)
	}
	expected := "release-identifier"
	versionFile := filepath.Join(dir, "version.txt")
	if err := os.WriteFile(versionFile, []byte(expected+"\n"), 0644); err != nil {
		t.Fatalf("Failed to write version file: %v", err)
	}
	identifier, err := GetInstalledReleaseIdentifier(tmpDir, version)
	if err != nil {
		t.Fatalf("GetInstalledReleaseIdentifier error: %v", err)
	}
	if identifier != expected {
		t.Errorf("GetInstalledReleaseIdentifier returned %q; want %q", identifier, expected)
	}
}

// TestClearDirectory creates a directory with files and subdirectories and verifies ClearDirectory removes them.
func TestClearDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	// Create some files and directories.
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "subfile.txt"), []byte("subcontent"), 0644); err != nil {
		t.Fatalf("Failed to create subfile: %v", err)
	}
	if err := ClearDirectory(tmpDir); err != nil {
		t.Fatalf("ClearDirectory error: %v", err)
	}
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected directory to be empty after ClearDirectory, found %d entries", len(entries))
	}
}

// TestTimeFormat verifies that TimeFormat converts an ISO time string to the expected format.
func TestTimeFormat(t *testing.T) {
	iso := "2025-03-07T15:04:05Z"
	expected := "2025-03-07"
	formatted := TimeFormat(iso)
	if formatted != expected {
		t.Errorf("TimeFormat(%q) = %q; want %q", iso, formatted, expected)
	}

	// If the input cannot be parsed, it should return the original string.
	invalid := "not-a-time"
	if TimeFormat(invalid) != invalid {
		t.Errorf("TimeFormat(%q) = %q; want %q", invalid, TimeFormat(invalid), invalid)
	}
}

// TestLaunchNvimWithConfig demonstrates how you might test LaunchNvimWithConfig if you refactor to use a
// package-level variable for LookPath. If that refactoring isn't possible, you can consider skipping
// this test or refactoring your code further to improve testability.
func TestLaunchNvimWithConfig(t *testing.T) {
	// Check if our package has an overridable lookPath variable.
	// If not, skip the test.
	type lookPathType func(string) (string, error)
	// Using a type assertion to see if we can get a pointer to the lookPath variable.
	// If your code doesn't expose such a variable, skip this test.
	//
	// For this example, we assume that the refactoring was done and a package-level variable named "lookPath"
	// exists. Uncomment the following lines if that's the case.
	//
	// origLookPath := lookPath
	// defer func() { lookPath = origLookPath }()
	// lookPath = func(file string) (string, error) {
	//     if file == "nvim" {
	//         return "/dummy/nvim", nil
	//     }
	//     return "", os.ErrNotExist
	// }
	//
	// Since we can't override exec.LookPath directly, if you haven't refactored, skip this test:
	t.Skip("Skipping TestLaunchNvimWithConfig because exec.LookPath cannot be overridden without refactoring")
}

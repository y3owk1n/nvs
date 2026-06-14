package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/y3owk1n/nvs/internal/constants"
)

// withVersionsDir swaps the package-level versionsDir to a
// per-test temp dir and restores it on cleanup. Every test that
// exercises code path reading GetVersionsDir (backup, lock,
// sentinel placement, ...) must go through this helper, otherwise
// it would mutate the user's real ~/.config/nvs/versions.
func withVersionsDir(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()

	original := versionsDir
	versionsDir = tempDir

	t.Cleanup(func() {
		versionsDir = original
	})

	return tempDir
}

// withNightly creates a fake nightly install dir under
// versionsDir with a small set of regular files. It returns the
// absolute path to the nightly dir.
func withNightly(t *testing.T, versionsDir string) string {
	t.Helper()

	nightlyDir := filepath.Join(versionsDir, constants.Nightly)

	err := os.MkdirAll(nightlyDir, constants.DirPerm)
	if err != nil {
		t.Fatalf("create nightly dir: %v", err)
	}

	files := map[string]string{
		"bin/nvim":           "elf-binary-stub",
		"share/nvim/runtime": "lua-stuff",
		"VERSION":            "0.12.3",
	}

	for relPath, content := range files {
		absPath := filepath.Join(nightlyDir, relPath)

		err := os.MkdirAll(filepath.Dir(absPath), constants.DirPerm)
		if err != nil {
			t.Fatalf("create parent for %s: %v", relPath, err)
		}

		err = os.WriteFile(absPath, []byte(content), constants.FilePerm)
		if err != nil {
			t.Fatalf("write %s: %v", relPath, err)
		}
	}

	return nightlyDir
}

func TestBackupNightlyUnderLock_HappyPath(t *testing.T) {
	versionsDir := withVersionsDir(t)

	nightlyDir := withNightly(t, versionsDir)

	backupDir := filepath.Join(
		versionsDir,
		"nightly-"+shortHash("abc123def456", constants.ShortHashLength),
	)

	err := backupNightlyUnderLock(nightlyDir, backupDir)
	if err != nil {
		t.Fatalf("backupNightlyUnderLock failed: %v", err)
	}

	// backupDir must exist with the sentinel and the copied files.
	sentinel := filepath.Join(backupDir, ".nvs-backup-owner")

	_, err = os.Stat(sentinel)
	if err != nil {
		t.Errorf("sentinel %s missing: %v", sentinel, err)
	}

	for relPath, wantContent := range map[string]string{
		"bin/nvim":           "elf-binary-stub",
		"share/nvim/runtime": "lua-stuff",
		"VERSION":            "0.12.3",
	} {
		gotBytes, readErr := os.ReadFile(filepath.Join(backupDir, relPath))
		if readErr != nil {
			t.Errorf("read %s: %v", relPath, readErr)

			continue
		}

		if string(gotBytes) != wantContent {
			t.Errorf("%s content = %q, want %q", relPath, gotBytes, wantContent)
		}
	}
}

func TestBackupNightlyUnderLock_NoTempLeftover(t *testing.T) {
	versionsDir := withVersionsDir(t)

	nightlyDir := withNightly(t, versionsDir)

	backupDir := filepath.Join(
		versionsDir,
		"nightly-"+shortHash("abc123def456", constants.ShortHashLength),
	)

	err := backupNightlyUnderLock(nightlyDir, backupDir)
	if err != nil {
		t.Fatalf("backupNightlyUnderLock failed: %v", err)
	}

	// No .nightly-backup-* staging dirs should remain under
	// versionsDir. The "nightly-*" dir is the published backup
	// and is expected to stay.
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".nightly-backup-") {
			t.Errorf("staging dir %s left behind after success", name)
		}

		// The previous failure mode also left `copy-temp-*` dirs.
		if strings.HasPrefix(name, "copy-temp-") {
			t.Errorf("staging dir %s left behind after success", name)
		}
	}
}

func TestBackupNightlyUnderLock_AlreadyClaimed(t *testing.T) {
	versionsDir := withVersionsDir(t)

	nightlyDir := withNightly(t, versionsDir)

	backupDir := filepath.Join(
		versionsDir,
		"nightly-"+shortHash("abc123def456", constants.ShortHashLength),
	)

	// First call: build the backup.
	err := backupNightlyUnderLock(nightlyDir, backupDir)
	if err != nil {
		t.Fatalf("first backup failed: %v", err)
	}

	// Corrupt the copy so we can detect a re-copy. If the
	// sentinel is honored, the corruption stays.
	versionFile := filepath.Join(backupDir, "VERSION")

	err = os.WriteFile(versionFile, []byte("CORRUPTED"), constants.FilePerm)
	if err != nil {
		t.Fatalf("corrupt VERSION: %v", err)
	}

	// Second call: must short-circuit on the sentinel and not
	// overwrite the corruption.
	err = backupNightlyUnderLock(nightlyDir, backupDir)
	if err != nil {
		t.Fatalf("second backup failed: %v", err)
	}

	gotBytes, err := os.ReadFile(versionFile)
	if err != nil {
		t.Fatalf("read VERSION: %v", err)
	}

	if string(gotBytes) != "CORRUPTED" {
		t.Errorf("second call overwrote the backup: VERSION = %q, want CORRUPTED", gotBytes)
	}
}

func TestBackupNightlyUnderLock_StaleDirRecovery(t *testing.T) {
	versionsDir := withVersionsDir(t)

	nightlyDir := withNightly(t, versionsDir)

	backupDir := filepath.Join(
		versionsDir,
		"nightly-"+shortHash("abc123def456", constants.ShortHashLength),
	)

	// Simulate a prior interrupted run: backupDir exists with
	// junk content but no sentinel.
	err := os.MkdirAll(backupDir, constants.DirPerm)
	if err != nil {
		t.Fatalf("create stale backupDir: %v", err)
	}

	err = os.WriteFile(
		filepath.Join(backupDir, "stale-junk.txt"),
		[]byte("leftover"),
		constants.FilePerm,
	)
	if err != nil {
		t.Fatalf("write stale junk: %v", err)
	}

	// The new run must replace the stale dir with a fully-formed
	// backup (sentinel + nightly contents). This is the exact
	// failure mode the user hit: backupDir pre-existing caused
	// the atomic rename inside copyDir to fail with "file exists".
	err = backupNightlyUnderLock(nightlyDir, backupDir)
	if err != nil {
		t.Fatalf("backupNightlyUnderLock failed: %v", err)
	}

	// Sentinel must be present.
	_, err = os.Stat(filepath.Join(backupDir, ".nvs-backup-owner"))
	if err != nil {
		t.Errorf("sentinel missing after recovery: %v", err)
	}

	// Stale junk must be gone.
	_, err = os.Stat(filepath.Join(backupDir, "stale-junk.txt"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("stale-junk.txt still present after recovery: %v", err)
	}

	// Nightly contents must be present.
	_, err = os.Stat(filepath.Join(backupDir, "bin", "nvim"))
	if err != nil {
		t.Errorf("bin/nvim missing after recovery: %v", err)
	}
}

func TestBackupNightlyUnderLock_SourceMissing(t *testing.T) {
	versionsDir := withVersionsDir(t)

	// No nightlyDir created — backup should fail cleanly with
	// no backupDir and no temp leftovers.
	missingDir := filepath.Join(versionsDir, constants.Nightly)

	backupDir := filepath.Join(
		versionsDir,
		"nightly-"+shortHash("abc123def456", constants.ShortHashLength),
	)

	err := backupNightlyUnderLock(missingDir, backupDir)
	if err == nil {
		t.Fatal("expected error when nightly dir is missing")
	}

	// backupDir must not be created on the failure path.
	_, err = os.Stat(backupDir)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("backupDir created despite source missing: stat err = %v", err)
	}

	// And no staging leftovers.
	entries, readErr := os.ReadDir(versionsDir)
	if readErr != nil {
		t.Fatalf("ReadDir: %v", readErr)
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".nightly-backup-") {
			t.Errorf("staging dir %s left behind after failure", entry.Name())
		}
	}
}

func TestCopyDirContents_Basic(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	files := map[string]string{
		"a.txt":     "alpha",
		"sub/b.txt": "bravo",
		"sub/c.txt": "charlie",
	}

	for rel, content := range files {
		abs := filepath.Join(src, rel)

		err := os.MkdirAll(filepath.Dir(abs), constants.DirPerm)
		if err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		err = os.WriteFile(abs, []byte(content), constants.FilePerm)
		if err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	err := copyDirContents(src, dst)
	if err != nil {
		t.Fatalf("copyDirContents failed: %v", err)
	}

	for rel, want := range files {
		got, err := os.ReadFile(filepath.Join(dst, rel))
		if err != nil {
			t.Errorf("read %s: %v", rel, err)

			continue
		}

		if string(got) != want {
			t.Errorf("%s = %q, want %q", rel, got, want)
		}
	}
}

func TestCopyDirContents_PreservesSymlink(t *testing.T) {
	if os.Getenv("NVS_TEST_SKIP_SYMLINK") == "1" {
		t.Skip("NVS_TEST_SKIP_SYMLINK=1")
	}

	src := t.TempDir()
	dst := t.TempDir()

	// File inside src, then a symlink inside src pointing at it.
	target := filepath.Join(src, "real.txt")

	err := os.WriteFile(target, []byte("payload"), constants.FilePerm)
	if err != nil {
		t.Fatalf("write target: %v", err)
	}

	link := filepath.Join(src, "link.txt")

	err = os.Symlink("real.txt", link)
	if err != nil {
		// Some environments (e.g. Windows without admin) cannot
		// create symlinks. Skip rather than fail.
		t.Skipf("symlink unsupported in this env: %v", err)
	}

	err = copyDirContents(src, dst)
	if err != nil {
		t.Fatalf("copyDirContents failed: %v", err)
	}

	// The symlink must still be a symlink after the copy (i.e.
	// we did not silently fall through to copying target
	// content on platforms that support symlinks).
	linkInfo, err := os.Lstat(filepath.Join(dst, "link.txt"))
	if err != nil {
		t.Fatalf("lstat link: %v", err)
	}

	if linkInfo.Mode()&os.ModeSymlink == 0 {
		t.Errorf("link.txt is no longer a symlink after copy")
	}
}

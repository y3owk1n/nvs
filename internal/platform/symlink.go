// Package platform provides platform-specific utilities.
// Package platform provides platform-specific utilities.
package platform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/sirupsen/logrus"
)

const (
	// WindowsOS is the string representation of the Windows operating system.
	WindowsOS = "windows"
)

// UpdateSymlink creates a symlink (or junction/hardlink fallback on Windows).
// If isDir = true, fallback uses junctions (mklink /J).
// If isDir = false, fallback uses hardlinks (mklink /H).
func UpdateSymlink(target, link string, isDir bool) error {
	// Remove old symlink if it exists.
	var err error

	_, err = os.Lstat(link)
	if err == nil {
		err = os.Remove(link)
		if err != nil {
			return err
		}
	}

	// Try normal symlink
	err = os.Symlink(target, link)
	if err == nil {
		return nil
	} else if runtime.GOOS != WindowsOS {
		// On non-Windows, fail fast
		return err
	}

	// Windows fallback
	if isDir {
		// Directory junction
		cmd := exec.CommandContext(context.Background(), "cmd", "/C", "mklink", "/J", link, target)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("failed to create junction for %s: %w", link, err)
		}

		logrus.Debugf("Created junction instead of symlink: %s -> %s", link, target)
	} else {
		// File hardlink
		cmd := exec.CommandContext(context.Background(), "cmd", "/C", "mklink", "/H", link, target)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("failed to create hardlink for %s: %w", link, err)
		}

		logrus.Debugf("Created hardlink instead of symlink: %s -> %s", link, target)
	}

	return nil
}

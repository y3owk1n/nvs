// Package platform provides platform-specific utilities.
package platform

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/internal/constants"
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
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat existing link %s: %w", link, err)
	}

	// Try normal symlink
	err = os.Symlink(target, link)
	if err == nil {
		return nil
	} else if runtime.GOOS != constants.WindowsOS {
		// On non-Windows, fail fast
		return err
	}

	// Windows fallback
	flag := "/H"

	linkType := "hardlink"
	if isDir {
		flag = "/J"
		linkType = "junction"
	}

	cmd := exec.CommandContext(context.Background(), "cmd", "/C", "mklink", flag, link, target)
	cmd.Stdout = os.Stdout

	var stderrBuf bytes.Buffer

	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	if err != nil {
		if stderrBuf.Len() > 0 {
			return fmt.Errorf(
				"failed to create %s for %s: %s: %w",
				linkType,
				link,
				stderrBuf.String(),
				err,
			)
		}

		return fmt.Errorf("failed to create %s for %s: %w", linkType, link, err)
	}

	logrus.Debugf("Created %s instead of symlink: %s -> %s", linkType, link, target)

	return nil
}

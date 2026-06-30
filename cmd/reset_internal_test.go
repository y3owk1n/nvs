package cmd

import (
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/y3owk1n/nvs/internal/constants"
)

const (
	testEtc      = "/etc"
	testDriveC   = `C:\`
	testDriveD   = `D:/`
	testUsersDir = `C:\Users`
)

func TestAssertSafeToRemovePath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		path    string
		wantErr bool
	}{
		// Legitimate NVS paths — must be accepted
		{
			name:    "macOS config dir",
			path:    "/Users/jane/Library/Application Support/nvs",
			wantErr: false,
		},
		{
			name:    "Linux XDG config",
			path:    "/home/jane/.config/nvs",
			wantErr: false,
		},
		{
			name:    "Linux fallback home config",
			path:    "/home/jane/.nvs",
			wantErr: false,
		},
		{
			name:    "cache dir deep",
			path:    "/home/jane/.cache/nvs",
			wantErr: false,
		},

		// Empty / degenerate
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "current dir",
			path:    ".",
			wantErr: true,
		},

		// Filesystem root
		{
			name:    "Unix root",
			path:    "/",
			wantErr: true,
		},
		{
			name:    "root with repeated separator",
			path:    string(filepath.Separator) + string(filepath.Separator),
			wantErr: true,
		},

		// Top-level system directories
		{
			name:    "Unix top-level /etc",
			path:    testEtc,
			wantErr: true,
		},
		{
			name:    "Unix top-level /Users",
			path:    "/Users",
			wantErr: true,
		},
		{
			name:    "Unix top-level /var",
			path:    "/var",
			wantErr: true,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := assertSafeToRemovePath(test.path)

			gotErr := got != nil
			if gotErr != test.wantErr {
				t.Errorf(
					"assertSafeToRemovePath(%q) error = %v, wantErr %v",
					test.path,
					got,
					test.wantErr,
				)
			}

			if gotErr && !errors.Is(got, errUnsafeResetPath) {
				t.Errorf(
					"assertSafeToRemovePath(%q) error = %v, expected to wrap errUnsafeResetPath",
					test.path,
					got,
				)
			}
		})
	}
}

func TestAssertSafeToRemovePath_Windows(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != constants.WindowsOS {
		t.Skip("Windows-only test")
	}

	cases := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "drive root C:\\",
			path:    testDriveC,
			wantErr: true,
		},
		{
			name:    "drive root D:/",
			path:    testDriveD,
			wantErr: true,
		},
		{
			name:    "top-level C:\\Users",
			path:    testUsersDir,
			wantErr: true,
		},
		{
			name:    "deep C:\\Users\\jane\\AppData\\Roaming\\nvs",
			path:    `C:\Users\jane\AppData\Roaming\nvs`,
			wantErr: false,
		},
		{
			name:    "UNC root \\\\",
			path:    `\\`,
			wantErr: true,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := assertSafeToRemovePath(test.path)

			gotErr := got != nil
			if gotErr != test.wantErr {
				t.Errorf(
					"assertSafeToRemovePath(%q) error = %v, wantErr %v",
					test.path,
					got,
					test.wantErr,
				)
			}
		})
	}
}

func TestAssertSafeToRemoveParent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "deep symlink path",
			path:    "/home/jane/.local/bin/nvim",
			wantErr: false,
		},
		{
			name:    "symlink at filesystem root",
			path:    "/nvim",
			wantErr: true,
		},
		{
			name:    "symlink under top-level /etc",
			path:    "/etc/nvim",
			wantErr: true,
		},
		{
			name:    "symlink under top-level /Users",
			path:    "/Users/nvim",
			wantErr: true,
		},
		{
			name:    "current dir symlink",
			path:    "./nvim",
			wantErr: true,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := assertSafeToRemoveParent(test.path)

			gotErr := got != nil
			if gotErr != test.wantErr {
				t.Errorf(
					"assertSafeToRemoveParent(%q) error = %v, wantErr %v",
					test.path,
					got,
					test.wantErr,
				)
			}
		})
	}
}

func TestIsDriveRoot(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != constants.WindowsOS {
		// isDriveRoot always returns false on non-Windows.
		for _, sample := range []string{`C:\`, `D:/`, `\\`, `C:`} {
			if isDriveRoot(sample) {
				t.Errorf("isDriveRoot(%q) on non-Windows should be false", sample)
			}
		}

		return
	}

	cases := []struct {
		path string
		want bool
	}{
		{testDriveC, true},
		{testDriveD, true},
		{`Z:\`, true},
		{testUsersDir, false},
		{testEtc, false},
		{`C:`, false},
		{``, false},
		{`\\`, false},
	}

	for _, test := range cases {
		t.Run(test.path, func(t *testing.T) {
			t.Parallel()

			if got := isDriveRoot(test.path); got != test.want {
				t.Errorf("isDriveRoot(%q) = %v, want %v", test.path, got, test.want)
			}
		})
	}
}

func TestIsFilesystemRoot(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == constants.WindowsOS {
		cases := []struct {
			path string
			want bool
		}{
			{testDriveC, true},
			{`/`, false}, // / is not a Windows drive root
			{`\`, true},
			{`\\`, true}, // UNC root
			{testUsersDir, false},
		}

		for _, test := range cases {
			t.Run(test.path, func(t *testing.T) {
				t.Parallel()

				if got := isFilesystemRoot(test.path); got != test.want {
					t.Errorf(
						"isFilesystemRoot(%q) = %v, want %v",
						test.path,
						got,
						test.want,
					)
				}
			})
		}

		return
	}

	// Unix.
	for _, sample := range []string{"/", "//", "/*not-root*", testEtc} {
		want := sample == "/" || sample == "//"
		if got := isFilesystemRoot(sample); got != want {
			t.Errorf("isFilesystemRoot(%q) = %v, want %v", sample, got, want)
		}
	}
}

// TestAssertSafeToRemovePath_FilepathCleanNormalization verifies that
// the safety check normalizes paths before inspecting them, so a
// caller passing a path with redundant slashes or trailing slashes
// gets the same answer as the cleaned form.
func TestAssertSafeToRemovePath_FilepathCleanNormalization(t *testing.T) {
	t.Parallel()

	// "/etc/" cleans to "/etc" (top-level, should be rejected).
	cleaned := filepath.Clean(testEtc + "/")

	got := assertSafeToRemovePath(cleaned)
	if got == nil {
		t.Errorf("assertSafeToRemovePath(filepath.Clean(%q)) = nil, want error", testEtc+"/")
	}
}

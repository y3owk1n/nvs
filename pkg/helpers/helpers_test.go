package helpers_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/y3owk1n/nvs/pkg/helpers"
)

func TestIcons(t *testing.T) {
	tests := []struct {
		name string
		icon string
	}{
		{"SuccessIcon", helpers.SuccessIcon()},
		{"ErrorIcon", helpers.ErrorIcon()},
		{"WarningIcon", helpers.WarningIcon()},
		{"InfoIcon", helpers.InfoIcon()},
		{"UpgradeIcon", helpers.UpgradeIcon()},
		{"PromptIcon", helpers.PromptIcon()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.icon == "" {
				t.Errorf("%s returned empty string", tt.name)
			}
		})
	}
}

func TestTextColors(t *testing.T) {
	tests := []struct {
		name   string
		result string
	}{
		{"GreenText", helpers.GreenText("test")},
		{"RedText", helpers.RedText("test")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result == "" {
				t.Errorf("%s returned empty string", tt.name)
			}
		})
	}
}

func TestTimeFormat(t *testing.T) {
	result := helpers.TimeFormat("2023-01-01T00:00:00Z")
	if result == "" {
		t.Errorf("TimeFormat returned empty string")
	}
}

// func TestColorizeRow(t *testing.T) {
// 	result := ColorizeRow("test", "value")
// 	if len(result) == 0 {
// 		t.Errorf("ColorizeRow returned empty slice")
// 	}
// }

func TestGetStandardNvimConfigDir_XDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")

	result, err := helpers.GetNvimConfigBaseDir()
	if err != nil {
		t.Fatalf("GetNvimConfigBaseDir failed: %v", err)
	}

	expected := "/tmp/xdg"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestGetStandardNvimConfigDir_windowsOSLocalAppData(t *testing.T) {
	//nolint:goconst
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}
	// Test would require setting LOCALAPPDATA
}

func TestGetStandardNvimConfigDir_windowsOSFallback(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}
	// Test would require unsetting LOCALAPPDATA
}

func TestGetStandardNvimConfigDir_UnixDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}

	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }() //nolint:usetesting

	_ = os.Unsetenv("XDG_CONFIG_HOME")

	result, err := helpers.GetNvimConfigBaseDir()
	if err != nil {
		t.Fatalf("GetNvimConfigBaseDir failed: %v", err)
	}

	home, _ := os.UserHomeDir()

	expected := filepath.Join(home, ".config")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

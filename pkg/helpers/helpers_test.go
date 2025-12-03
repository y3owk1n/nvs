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
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	// Test LOCALAPPDATA path
	t.Setenv("LOCALAPPDATA", "C:\\Users\\TestUser\\AppData\\Local")

	result, err := helpers.GetNvimConfigBaseDir()
	if err != nil {
		t.Fatalf("GetNvimConfigBaseDir failed: %v", err)
	}

	expected := "C:\\Users\\TestUser\\AppData\\Local"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestGetStandardNvimConfigDir_windowsOSFallback(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	// Test fallback when LOCALAPPDATA is not set
	t.Setenv("LOCALAPPDATA", "")

	result, err := helpers.GetNvimConfigBaseDir()
	if err != nil {
		t.Fatalf("GetNvimConfigBaseDir failed: %v", err)
	}

	home := os.Getenv("USERPROFILE")
	if home == "" {
		home = "C:\\Users\\Default"
	}
	expected := filepath.Join(home, ".config")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestGetStandardNvimConfigDir_UnixDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}

	result, err := helpers.GetNvimConfigBaseDir()
	if err != nil {
		t.Fatalf("GetNvimConfigBaseDir failed: %v", err)
	}

	expected := filepath.Join(os.Getenv("HOME"), ".config")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

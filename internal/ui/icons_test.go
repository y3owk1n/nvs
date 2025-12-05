package ui_test

import (
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/y3owk1n/nvs/internal/ui"
)

func TestIconConstants(t *testing.T) {
	// Verify icon constants are set correctly
	tests := []struct {
		name string
		icon string
		want string
	}{
		{"Checkmark", ui.Checkmark, "✓"},
		{"Cross", ui.Cross, "✖"},
		{"Info", ui.Info, "ℹ"},
		{"Warn", ui.Warn, "⚠"},
		{"Upgrade", ui.Upgrade, "↑"},
		{"Prompt", ui.Prompt, "?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.icon != tt.want {
				t.Errorf("%s icon = %q, want %q", tt.name, tt.icon, tt.want)
			}
		})
	}
}

func TestColoredIcon(t *testing.T) {
	tests := []struct {
		name  string
		icon  string
		color color.Attribute
	}{
		{"green checkmark", ui.Checkmark, color.FgGreen},
		{"red cross", ui.Cross, color.FgRed},
		{"yellow warning", ui.Warn, color.FgYellow},
		{"blue info", ui.Info, color.FgBlue},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := ui.ColoredIcon(testCase.icon, testCase.color)
			// Result should not be empty
			if len(result) == 0 {
				t.Error("ColoredIcon() returned empty string")
			}
			// Result should contain the original icon (possibly with ANSI codes)
			if !strings.Contains(result, testCase.icon) {
				t.Errorf(
					"ColoredIcon() result %q does not contain original icon %q",
					result,
					testCase.icon,
				)
			}
			// Note: In CI, colors may be disabled, but the base icon should be present
		})
	}
}

func TestIconFunctions(t *testing.T) {
	tests := []struct {
		name string
		fn   func() string
	}{
		{"SuccessIcon", ui.SuccessIcon},
		{"ErrorIcon", ui.ErrorIcon},
		{"WarningIcon", ui.WarningIcon},
		{"InfoIcon", ui.InfoIcon},
		{"UpgradeIcon", ui.UpgradeIcon},
		{"PromptIcon", ui.PromptIcon},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn()
			if len(result) == 0 {
				t.Errorf("%s() returned empty string", tt.name)
			}
		})
	}
}

func TestTextColorFunctions(t *testing.T) {
	tests := []struct {
		name string
		fn   func(string) string
	}{
		{"WhiteText", ui.WhiteText},
		{"CyanText", ui.CyanText},
		{"GreenText", ui.GreenText},
		{"RedText", ui.RedText},
		{"YellowText", ui.YellowText},
		{"MagentaText", ui.MagentaText},
	}

	input := "test text"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn(input)
			if len(result) == 0 {
				t.Errorf("%s(%q) returned empty string", tt.name, input)
			}
		})
	}
}

func TestTextColorFunctions_EmptyString(t *testing.T) {
	tests := []struct {
		name string
		fn   func(string) string
	}{
		{"WhiteText", ui.WhiteText},
		{"CyanText", ui.CyanText},
		{"GreenText", ui.GreenText},
		{"RedText", ui.RedText},
		{"YellowText", ui.YellowText},
		{"MagentaText", ui.MagentaText},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic with empty input
			_ = tt.fn("")
		})
	}
}

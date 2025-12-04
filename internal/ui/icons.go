package ui

import "github.com/fatih/color"

// Constants for icons.
const (
	Checkmark = "✓"
	Cross     = "✖"
	Info      = "ℹ"
	Warn      = "⚠"
	Upgrade   = "↑"
	Prompt    = "?"
)

// ColoredIcon colors an icon with the given color.
func ColoredIcon(icon string, fgColor color.Attribute) string {
	return color.New(fgColor, color.Bold).Sprint(icon)
}

// SuccessIcon returns a colored success icon.
func SuccessIcon() string {
	return ColoredIcon(Checkmark, color.FgGreen)
}

// ErrorIcon returns a colored error icon.
func ErrorIcon() string {
	return ColoredIcon(Cross, color.FgRed)
}

// WarningIcon returns a colored warning icon.
func WarningIcon() string {
	return ColoredIcon(Warn, color.FgYellow)
}

// InfoIcon returns a colored info icon.
func InfoIcon() string {
	return ColoredIcon(Info, color.FgBlue)
}

// UpgradeIcon returns a colored upgrade icon.
func UpgradeIcon() string {
	return ColoredIcon(Upgrade, color.FgYellow)
}

// PromptIcon returns a colored prompt icon.
func PromptIcon() string {
	return ColoredIcon(Prompt, color.FgCyan)
}

// WhiteText colors text white.
func WhiteText(text string) string {
	return color.New(color.FgWhite).Sprint(text)
}

// CyanText colors text cyan.
func CyanText(text string) string {
	return color.New(color.FgCyan).Sprint(text)
}

// GreenText colors text green.
func GreenText(text string) string {
	return color.New(color.FgGreen).Sprint(text)
}

// RedText colors text red.
func RedText(text string) string {
	return color.New(color.FgRed).Sprint(text)
}

// YellowText colors text yellow.
func YellowText(text string) string {
	return color.New(color.FgYellow).Sprint(text)
}

// MagentaText colors text magenta.
func MagentaText(text string) string {
	return color.New(color.FgMagenta).Sprint(text)
}

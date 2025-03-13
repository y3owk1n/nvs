package utils

import "github.com/fatih/color"

const (
	Checkmark = "✓"
	Cross     = "✖"
	Info      = "ℹ"
	Warn      = "⚠"
	Upgrade   = "↑"
	Prompt    = "?"
)

func ColoredIcon(icon string, fgColor color.Attribute) string {
	return color.New(fgColor, color.Bold).Sprint(icon)
}

func SuccessIcon() string {
	return ColoredIcon(Checkmark, color.FgGreen)
}

func ErrorIcon() string {
	return ColoredIcon(Cross, color.FgRed)
}

func WarningIcon() string {
	return ColoredIcon(Warn, color.FgYellow)
}

func InfoIcon() string {
	return ColoredIcon(Info, color.FgBlue)
}

func UpgradeIcon() string {
	return ColoredIcon(Upgrade, color.FgYellow)
}

func PromptIcon() string {
	return ColoredIcon(Prompt, color.FgCyan)
}

func WhiteText(text string) string {
	return color.New(color.FgWhite).Sprint(text)
}

func CyanText(text string) string {
	return color.New(color.FgCyan).Sprint(text)
}

func GreenText(text string) string {
	return color.New(color.FgGreen).Sprint(text)
}

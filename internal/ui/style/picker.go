package style

// PickerPalette is the set of colors used by the nvs-themed
// huh picker. huh themes are evaluated on a single background
// (not adaptive), so this is a flat color set rather than the
// Light/Dark pair used by Palette.
//
// The values are derived from the active Palette:
//
//   - Primary:    palette.Primary.Dark   (NVS_COLOR_PRIMARY[_DARK])
//   - Muted:      palette.Subtle.Dark    (NVS_COLOR_SUBTLE[_DARK])
//   - Text:       palette.Text.Dark      (NVS_COLOR_TEXT[_DARK])
//   - Background: palette.Text.Light     (NVS_COLOR_TEXT_LIGHT)
//
// There are no separate NVS_PICKER_* environment variables:
// every picker color maps to an existing palette slot, and
// overriding that slot (e.g. NVS_COLOR_PRIMARY=#abcdef) flows
// through the picker automatically. This keeps the design
// system as a single source of truth and avoids the "I changed
// NVS_COLOR_PRIMARY but the picker still looks old" footgun.
type PickerPalette struct {
	// Primary is the focus / accent color (titles, selectors,
	// the focused button background).
	Primary string

	// Muted is the unfocused / hint color (blurred title, item
	// description separators).
	Muted string

	// Text is the unselected-option color.
	Text string

	// Background is the focused-button foreground / contrast
	// color rendered on top of the primary background.
	Background string
}

// PickerColors returns the picker theme colors derived from
// the active Palette. Because every picker slot is a
// palette-derived value, no env-var lookup is needed here:
// callers that want a custom picker appearance should
// override the corresponding NVS_COLOR_* slot instead.
func PickerColors() PickerPalette {
	palette := Default()

	return PickerPalette{
		Primary:    palette.Primary.Dark,
		Muted:      palette.Subtle.Dark,
		Text:       palette.Text.Dark,
		Background: palette.Text.Light,
	}
}

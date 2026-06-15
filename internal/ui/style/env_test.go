package style_test

import (
	"testing"

	"github.com/y3owk1n/nvs/internal/ui/style"
)

// unsetColorEnv removes every NVS_COLOR_<NAME> variable
// (plus its _LIGHT / _DARK variants) from the process
// environment for the duration of the test. It is the
// t.Setenv-friendly counterpart of the "no overrides"
// baseline: every test that wants to assert a clean baseline
// can call it and then set only the variables it cares about.
//
// Tests in this file must NOT use t.Parallel(); t.Setenv
// forbids it because env mutation is process-global.
func unsetColorEnv(t *testing.T) {
	t.Helper()

	// Palette slot names — kept in sync with the override
	// calls inside style.Default().
	slots := []string{
		"PRIMARY", "TEXT", "MUTED", "SUBTLE",
		"BORDER", "ACCENT", "SUCCESS", "WARNING", "ERROR",
	}

	for _, slot := range slots {
		t.Setenv("NVS_COLOR_"+slot, "")
		t.Setenv("NVS_COLOR_"+slot+"_LIGHT", "")
		t.Setenv("NVS_COLOR_"+slot+"_DARK", "")
	}
}

func TestDefaultPaletteHonorsBaseOverride(t *testing.T) {
	unsetColorEnv(t)

	t.Setenv("NVS_COLOR_PRIMARY", "#FF00FF")

	palette := style.Default()

	if got := palette.Primary.Light; got != "#FF00FF" {
		t.Errorf("Primary.Light = %q, want %q (base override)", got, "#FF00FF")
	}

	if got := palette.Primary.Dark; got != "#FF00FF" {
		t.Errorf(
			"Primary.Dark = %q, want %q (base override should affect both variants)",
			got,
			"#FF00FF",
		)
	}
}

func TestDefaultPaletteHonorsVariantOverrides(t *testing.T) {
	unsetColorEnv(t)

	t.Setenv("NVS_COLOR_ACCENT_LIGHT", "#111111")
	t.Setenv("NVS_COLOR_ACCENT_DARK", "#EEEEEE")

	palette := style.Default()

	if got := palette.Accent.Light; got != "#111111" {
		t.Errorf("Accent.Light = %q, want %q", got, "#111111")
	}

	if got := palette.Accent.Dark; got != "#EEEEEE" {
		t.Errorf("Accent.Dark = %q, want %q", got, "#EEEEEE")
	}
}

func TestDefaultPaletteVariantOverridesBeatBase(t *testing.T) {
	unsetColorEnv(t)

	// Base sets both; _LIGHT / _DARK must override the matching
	// side without affecting the other. This is the common
	// "give me a one-liner for the easy case, but let me
	// fine-tune the dark variant" workflow.
	t.Setenv("NVS_COLOR_SUCCESS", "#AABBCC")
	t.Setenv("NVS_COLOR_SUCCESS_DARK", "#DDEEFF")

	palette := style.Default()

	if got := palette.Success.Light; got != "#AABBCC" {
		t.Errorf("Success.Light = %q, want %q (base value)", got, "#AABBCC")
	}

	if got := palette.Success.Dark; got != "#DDEEFF" {
		t.Errorf("Success.Dark = %q, want %q (_DARK should win over base)", got, "#DDEEFF")
	}
}

func TestDefaultPaletteTrimsWhitespace(t *testing.T) {
	unsetColorEnv(t)

	t.Setenv("NVS_COLOR_WARNING_LIGHT", "  #ABC123  ")

	palette := style.Default()

	if got := palette.Warning.Light; got != "#ABC123" {
		t.Errorf("Warning.Light = %q, want trimmed %q", got, "#ABC123")
	}
}

func TestDefaultPaletteEmptyValueIsIgnored(t *testing.T) {
	unsetColorEnv(t)

	// An explicitly empty value is treated as "no override" so
	// users can safely source a file that exports every slot
	// and then unset just the ones they care about.
	t.Setenv("NVS_COLOR_ERROR", "")

	palette := style.Default()

	if palette.Error.Light == "" || palette.Error.Dark == "" {
		t.Error("Error color should keep its defaults when NVS_COLOR_ERROR is empty")
	}
}

func TestDefaultPaletteOnlyTouchesTheConfiguredSlot(t *testing.T) {
	unsetColorEnv(t)

	t.Setenv("NVS_COLOR_BORDER", "#FF0000")

	palette := style.Default()

	if got := palette.Border.Light; got != "#FF0000" {
		t.Errorf("Border.Light = %q, want %q", got, "#FF0000")
	}

	// The other slots must be untouched. We assert on Primary
	// (one of the slots we never set) rather than the full
	// palette — the full check is the responsibility of
	// TestDefaultPalette in style_test.go.
	if palette.Primary.Light == "#FF0000" {
		t.Error("Primary.Light should not be affected by NVS_COLOR_BORDER")
	}
}

func TestPickerColorsDefaultToPaletteDarkVariants(t *testing.T) {
	unsetColorEnv(t)

	palette := style.Default()
	picker := style.PickerColors()

	if got, want := picker.Primary, palette.Primary.Dark; got != want {
		t.Errorf("PickerColors.Primary = %q, want %q (palette.Primary.Dark)", got, want)
	}

	if got, want := picker.Muted, palette.Subtle.Dark; got != want {
		t.Errorf("PickerColors.Muted = %q, want %q (palette.Subtle.Dark)", got, want)
	}

	if got, want := picker.Text, palette.Text.Dark; got != want {
		t.Errorf("PickerColors.Text = %q, want %q (palette.Text.Dark)", got, want)
	}

	if got, want := picker.Background, palette.Text.Light; got != want {
		t.Errorf("PickerColors.Background = %q, want %q (palette.Text.Light)", got, want)
	}
}

func TestPickerColorsFollowPaletteOverride(t *testing.T) {
	unsetColorEnv(t)

	// Changing the palette should automatically flow through
	// to the picker — that's the whole point of a single
	// source of truth.
	t.Setenv("NVS_COLOR_PRIMARY", "#FACADE")

	picker := style.PickerColors()

	if got, want := picker.Primary, "#FACADE"; got != want {
		t.Errorf("PickerColors.Primary = %q, want %q (should follow NVS_COLOR_PRIMARY)", got, want)
	}
}

func TestPickerColorsFollowVariantOverride(t *testing.T) {
	unsetColorEnv(t)

	// Picker.Text is derived from palette.Text.Dark, so
	// NVS_COLOR_TEXT_DARK must drive the picker too.
	t.Setenv("NVS_COLOR_TEXT_DARK", "#0F0F0F")

	picker := style.PickerColors()

	if got, want := picker.Text, "#0F0F0F"; got != want {
		t.Errorf("PickerColors.Text = %q, want %q (should follow NVS_COLOR_TEXT_DARK)", got, want)
	}
}

func TestPickerColorsBackgroundFollowsTextLight(t *testing.T) {
	unsetColorEnv(t)

	// Picker.Background is derived from palette.Text.Light, so
	// NVS_COLOR_TEXT_LIGHT must drive the picker too.
	t.Setenv("NVS_COLOR_TEXT_LIGHT", "#FAFAFA")

	picker := style.PickerColors()

	if got, want := picker.Background, "#FAFAFA"; got != want {
		t.Errorf(
			"PickerColors.Background = %q, want %q (should follow NVS_COLOR_TEXT_LIGHT)",
			got,
			want,
		)
	}
}

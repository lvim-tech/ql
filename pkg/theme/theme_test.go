package theme

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// hasColor reports whether a lipgloss colour was actually set (an unset entry
// is lipgloss.NoColor).
func hasColor(c lipgloss.TerminalColor) bool {
	_, isNone := c.(lipgloss.NoColor)
	return c != nil && !isNone
}

// An arbitrary custom palette must reach the styles: every role that was given
// a colour ends up applied, and the title badge carries both a background
// (accent) and its own foreground (title_fg) so it stays legible.
func TestNewStylesArbitraryPalette(t *testing.T) {
	theme := Theme{
		Name:   "custom",
		Border: "rounded",
		Icons:  IconsUnicode,
		Colors: ColorSet{
			Accent:  Color{Light: "#112233", Dark: "#445566"},
			Text:    Color{Light: "#010101", Dark: "#fefefe"},
			Muted:   Color{Light: "#777777", Dark: "#888888"},
			TitleFg: Color{Light: "#ffffff", Dark: "#000000"},
		},
	}

	s := NewStyles(theme)

	if !hasColor(s.Title.GetForeground()) {
		t.Error("title badge has no foreground")
	}
	if !hasColor(s.Title.GetBackground()) {
		t.Error("title badge has no background (accent should paint it)")
	}
	if !s.Title.GetBold() {
		t.Error("title badge is not bold")
	}
	if !hasColor(s.Cursor.GetForeground()) {
		t.Error("cursor marker did not take the accent colour")
	}
	if s.Cursor.GetReverse() {
		t.Error("coloured theme should not fall back to reverse video for the cursor")
	}
	if !hasColor(s.Item.GetForeground()) {
		t.Error("plain item did not take the text colour")
	}
	if !hasColor(s.Muted.GetForeground()) {
		t.Error("muted style did not take the muted colour")
	}
}

// The mono theme has no colour at all: emphasis must come from weight and
// reverse video instead, and nothing may acquire a foreground.
func TestNewStylesMonoDegrades(t *testing.T) {
	mono, ok := BuiltinTheme("mono")
	if !ok {
		t.Fatal("mono preset missing")
	}
	s := NewStyles(mono)

	if hasColor(s.Cursor.GetForeground()) {
		t.Error("mono cursor should have no foreground")
	}
	if !s.Cursor.GetReverse() {
		t.Error("mono cursor should use reverse video to stay visible")
	}
	if hasColor(s.Title.GetForeground()) || hasColor(s.Title.GetBackground()) {
		t.Error("mono title should paint no colour")
	}
	if !s.Title.GetReverse() {
		t.Error("mono title should use reverse video")
	}
	if !s.Muted.GetFaint() {
		t.Error("mono muted text should be faint")
	}
	// mono ships the ASCII glyph set.
	if s.Icons.Cursor != asciiIcons.Cursor {
		t.Errorf("mono cursor glyph = %q, want ascii %q", s.Icons.Cursor, asciiIcons.Cursor)
	}
}

// Resolve falls back to the default preset (and reports the error) when the
// named theme is neither built-in nor an installed file.
func TestResolveUnknownFallsBackToDefault(t *testing.T) {
	got, err := Resolve(Theme{Name: "does-not-exist"})
	if err == nil {
		t.Error("resolving an unknown theme should report an error")
	}
	if got.Name != DefaultThemeName {
		t.Errorf("fallback theme = %q, want %q", got.Name, DefaultThemeName)
	}
}

// A structural override on a built-in preset applies without disturbing the
// palette it inherits.
func TestResolveOverridesStructureKeepsPalette(t *testing.T) {
	got, err := Resolve(Theme{Name: "default", Border: "double", Icons: IconsASCII})
	if err != nil {
		t.Fatalf("resolve default: %v", err)
	}
	if got.Border != "double" {
		t.Errorf("border override lost: %q", got.Border)
	}
	if got.Icons != IconsASCII {
		t.Errorf("icons override lost: %q", got.Icons)
	}
	if got.Colors.Accent.IsZero() {
		t.Error("default palette was dropped by a structural override")
	}
}

// A theme name that tries to escape the themes directory is refused.
func TestLoadThemeFileRejectsPathTraversal(t *testing.T) {
	if _, err := LoadThemeFile("../evil"); err == nil {
		t.Error("path traversal in a theme name was not rejected")
	}
}

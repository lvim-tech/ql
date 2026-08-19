// Package theme describes how ql's built-in TUI looks. It is plain data — this
// file never imports a rendering library — so the palette stays independent of
// the terminal front-end that consumes it (see NewStyles in styles.go).
//
// A theme is a base (a built-in preset, or a YAML file in the themes directory)
// plus optional structural overrides from ql's config. The nine colour roles
// mirror the lvim-tech unified design shared with clipack, so a theme written
// for one reads in the other.
package theme

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Theme is a resolved look: a name, structural knobs, and the palette.
//
// A theme file is YAML, kept next to config.toml under themes/<name>.yaml:
//
//	# ~/.config/ql/themes/mytheme.yaml
//	border: rounded
//	icons: unicode
//	colors:
//	  accent: "#81a1c1"
//	  title_fg: "#1c1c1c"
//
// Anything left out falls back to the default preset, so overriding one colour
// does not mean restating the whole palette.
type Theme struct {
	Name   string   `yaml:"name,omitempty"`
	Border string   `yaml:"border,omitempty"`
	Icons  string   `yaml:"icons,omitempty"`
	Colors ColorSet `yaml:"colors,omitempty"`
}

// Color is one palette entry. It adapts to the terminal background: Light on a
// light background, Dark on a dark one. In YAML it accepts either form:
//
//	accent: "#81a1c1"                            # both backgrounds
//	accent: {light: "#2f5d9e", dark: "#81a1c1"}  # per background
type Color struct {
	Light string `yaml:"light,omitempty"`
	Dark  string `yaml:"dark,omitempty"`
}

// IsZero reports whether the colour was left unset.
func (c Color) IsZero() bool {
	return c.Light == "" && c.Dark == ""
}

// UnmarshalYAML accepts both the scalar and the mapping form.
func (c *Color) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		var value string
		if err := node.Decode(&value); err != nil {
			return err
		}
		c.Light, c.Dark = value, value
		return nil
	}

	// A distinct type avoids recursing back into this method.
	type rawColor Color
	var raw rawColor
	if err := node.Decode(&raw); err != nil {
		return err
	}

	*c = Color(raw)
	// A half-specified colour applies to both backgrounds.
	if c.Light == "" {
		c.Light = c.Dark
	}
	if c.Dark == "" {
		c.Dark = c.Light
	}
	return nil
}

// ColorSet is the palette a theme defines: the nine unified-design roles.
type ColorSet struct {
	// Accent drives the selection cursor, the title badge and the focused border.
	Accent Color `yaml:"accent,omitempty"`
	// AccentAlt marks headings and the active step.
	AccentAlt Color `yaml:"accent_alt,omitempty"`
	// Text is the default foreground.
	Text Color `yaml:"text,omitempty"`
	// Muted is for secondary information: hints and field labels.
	Muted Color `yaml:"muted,omitempty"`
	// Subtle is the colour of dividers and unfocused borders.
	Subtle Color `yaml:"subtle,omitempty"`
	// Success marks completed operations.
	Success Color `yaml:"success,omitempty"`
	// Warning marks non-fatal problems.
	Warning Color `yaml:"warning,omitempty"`
	// Error marks failures.
	Error Color `yaml:"error,omitempty"`
	// TitleFg is the foreground of the title badge, drawn on Accent.
	TitleFg Color `yaml:"title_fg,omitempty"`
}

// Icons is the glyph set used for the cursor marker and the pagination dots.
type Icons struct {
	Cursor       string
	PageActive   string
	PageInactive string
}

// Theme defaults and the accepted values for each knob.
const (
	DefaultThemeName = "default"
	DefaultBorder    = "normal"
	DefaultIcons     = "unicode"

	IconsUnicode = "unicode"
	IconsASCII   = "ascii"

	// ThemeFileExt is the extension of a theme file in the themes directory.
	ThemeFileExt = ".yaml"
)

// unicodeIcons is the default glyph set.
var unicodeIcons = Icons{
	Cursor:       "▌ ",
	PageActive:   "● ",
	PageInactive: "○ ",
}

// asciiIcons keeps the menu readable on terminals or fonts that cannot render
// the box-drawing glyphs.
var asciiIcons = Icons{
	Cursor:       "> ",
	PageActive:   "# ",
	PageInactive: ". ",
}

// builtinThemes holds the presets, keyed by name. Every preset is complete: an
// override always merges on top of a fully populated base.
var builtinThemes = map[string]Theme{
	"default": {
		Name:   "default",
		Border: DefaultBorder,
		Icons:  DefaultIcons,
		Colors: ColorSet{
			// The lvim-tech unified palette shared with clipack: Nord blue and
			// orange, no magenta and no yellow — the two hues that read as
			// foreign on this desktop. Contrast is measured against the
			// backgrounds each is drawn on, above a floor of 3.
			Accent:    Color{Light: "#2f5d9e", Dark: "#81a1c1"},
			AccentAlt: Color{Light: "#a2542b", Dark: "#d08770"},
			Text:      Color{Light: "#1c1c1c", Dark: "#e5e9f0"},
			Muted:     Color{Light: "#6b6b6b", Dark: "#7f8896"},
			Subtle:    Color{Light: "#c8c8c8", Dark: "#3b4252"},
			Success:   Color{Light: "#217a3d", Dark: "#a3be8c"},
			Warning:   Color{Light: "#b3261e", Dark: "#bf616a"},
			Error:     Color{Light: "#b3261e", Dark: "#bf616a"},
			TitleFg:   Color{Light: "#ffffff", Dark: "#1c1c1c"},
		},
	},
	"mono": {
		// No colour at all: everything is drawn with the terminal's own
		// foreground, and emphasis comes from weight and reverse video. Useful
		// over a serial console or on a palette ql cannot know.
		Name:   "mono",
		Border: "normal",
		Icons:  IconsASCII,
		Colors: ColorSet{},
	},
}

// BuiltinTheme returns a preset by name.
func BuiltinTheme(name string) (Theme, bool) {
	t, ok := builtinThemes[name]
	return t, ok
}

// DefaultTheme is the fully populated fallback.
func DefaultTheme() Theme {
	return builtinThemes[DefaultThemeName]
}

// ThemesDir returns the directory holding user theme files, next to
// config.toml: ~/.config/ql/themes.
func ThemesDir() string {
	home := os.Getenv("HOME")
	return filepath.Join(home, ".config", "ql", "themes")
}

// LoadThemeFile reads a theme from the themes directory.
func LoadThemeFile(name string) (Theme, error) {
	// A theme name is a file name, never a path: refuse anything that would
	// escape the themes directory.
	if name != filepath.Base(name) || name == "." || name == ".." {
		return Theme{}, fmt.Errorf("invalid theme name %q", name)
	}

	dir := ThemesDir()
	for _, ext := range []string{ThemeFileExt, ".yml"} {
		data, err := os.ReadFile(filepath.Join(dir, name+ext))
		if err != nil {
			continue
		}

		var t Theme
		if err := yaml.Unmarshal(data, &t); err != nil {
			return Theme{}, fmt.Errorf("parsing theme %s%s: %w", name, ext, err)
		}
		t.Name = name
		return t, nil
	}

	return Theme{}, fmt.Errorf("theme %q not found: no built-in of that name and no %s in %s",
		name, name+ThemeFileExt, dir)
}

// Resolve produces the theme the interface should actually render with.
//
// The base is the built-in preset named by t.Name, or the matching file in the
// themes directory. Whatever t itself sets (border, icons) is then layered on
// top, so a structural knob can be overridden from config without copying the
// palette. A base that is neither built-in nor installed returns the default
// theme and an error; the caller decides whether that is fatal.
func Resolve(t Theme) (Theme, error) {
	name := t.Name
	if name == "" {
		name = DefaultThemeName
	}

	base, ok := builtinThemes[name]
	if !ok {
		loaded, err := LoadThemeFile(name)
		if err != nil {
			return DefaultTheme(), err
		}
		// A file may itself omit the structural knobs or some colours.
		base = builtinThemes[DefaultThemeName].merge(loaded)
	}

	resolved := base.merge(Theme{Border: t.Border, Icons: t.Icons, Colors: t.Colors})
	resolved.Name = name
	return resolved, nil
}

// IconSet returns the glyphs for the theme.
func (t Theme) IconSet() Icons {
	if t.Icons == IconsASCII {
		return asciiIcons
	}
	return unicodeIcons
}

// merge layers the non-empty values of over on top of t.
func (t Theme) merge(over Theme) Theme {
	out := t
	if over.Name != "" {
		out.Name = over.Name
	}
	if over.Border != "" {
		out.Border = over.Border
	}
	if over.Icons != "" {
		out.Icons = over.Icons
	}
	out.Colors = t.Colors.merge(over.Colors)
	return out
}

// merge layers non-empty overrides on top of a base palette.
func (c ColorSet) merge(over ColorSet) ColorSet {
	pick := func(override, base Color) Color {
		if override.IsZero() {
			return base
		}
		return override
	}

	return ColorSet{
		Accent:    pick(over.Accent, c.Accent),
		AccentAlt: pick(over.AccentAlt, c.AccentAlt),
		Text:      pick(over.Text, c.Text),
		Muted:     pick(over.Muted, c.Muted),
		Subtle:    pick(over.Subtle, c.Subtle),
		Success:   pick(over.Success, c.Success),
		Warning:   pick(over.Warning, c.Warning),
		Error:     pick(over.Error, c.Error),
		TitleFg:   pick(over.TitleFg, c.TitleFg),
	}
}

package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// Styles is every lipgloss style ql's TUI draws with, compiled once from a
// resolved Theme. It is built in NewStyles and then only read, so a theme
// change means building a new Styles rather than mutating package state — which
// is also what makes the rendering testable with an arbitrary palette.
//
// It is deliberately small: ql's built-in menu is a single filterable list, not
// clipack's multi-pane browser, so it needs a top-bar chip, a border, a
// selection highlight, key hints and pagination dots — and nothing more.
type Styles struct {
	Theme Theme
	Icons Icons

	// Title is the app-name chip at the left of the top bar: title_fg on
	// accent, bold — the same convention keyforge draws its name with.
	Title lipgloss.Style
	// Prompt is the current menu's name, drawn next to the chip like an
	// active tab button: title_fg on accent_alt, bold.
	Prompt lipgloss.Style
	// Border wraps the whole list. It is the only, focused pane, so it takes
	// the accent colour rather than the subtle one.
	Border lipgloss.Style
	// Cursor is the accent marker in front of the highlighted row.
	Cursor lipgloss.Style
	// Selected is the highlighted row's name.
	Selected lipgloss.Style
	// Item is a normal, unselected row.
	Item lipgloss.Style
	// Muted styles the key-hints line and secondary text.
	Muted lipgloss.Style
	// Key styles the "[k]" part of a footer hint in the accent colour; the
	// label after it is drawn with Muted.
	Key lipgloss.Style
	// Filter styles the filter prompt and cursor, replacing bubbles' neon
	// defaults with the theme's accent.
	Filter lipgloss.Style
	// Placeholder styles the filter's placeholder text.
	Placeholder lipgloss.Style

	// PageActive and PageInactive are the pre-rendered pagination dots. Each
	// carries its own trailing space: the paginator concatenates them with no
	// separator, and adjacent circles are hard to count without one.
	PageActive   string
	PageInactive string
}

// NewStyles compiles a resolved theme into styles.
//
// An unset colour deliberately produces a style with no foreground rather than
// a black one: that is how the "mono" theme renders using the terminal's own
// colours, and how a partial custom theme degrades gracefully.
func NewStyles(theme Theme) Styles {
	c := theme.Colors
	icons := theme.IconSet()

	accent := adaptive(c.Accent)
	text := adaptive(c.Text)
	muted := adaptive(c.Muted)
	subtle := adaptive(c.Subtle)

	border := borderStyle(theme.Border)

	pane := lipgloss.NewStyle().Padding(0, 1)
	if theme.Border != "none" {
		pane = pane.Border(border)
	}
	// The list is the only pane on screen, so it reads as focused: accent
	// border where there is an accent, subtle otherwise.
	paneColor := accent
	if paneColor == nil {
		paneColor = subtle
	}

	// The prompt tab sits on accent_alt — keyforge's active-tab convention —
	// falling back to accent so a palette that skips accent_alt still gets a
	// painted tab rather than an invisible one.
	promptBg := c.AccentAlt
	if promptBg.IsZero() {
		promptBg = c.Accent
	}

	s := Styles{
		Theme: theme,
		Icons: icons,

		// The chip and the prompt tab are the places a background is painted,
		// so each needs its own foreground to stay legible on its colour.
		Title:  bold(fg(bg(lipgloss.NewStyle(), c.Accent), c.TitleFg)).Padding(0, 1),
		Prompt: bold(fg(bg(lipgloss.NewStyle(), promptBg), c.TitleFg)).Padding(0, 1),
		Border: setBorderFg(pane, paneColor),

		Cursor:   setFg(lipgloss.NewStyle().Bold(true), accent),
		Selected: setFg(lipgloss.NewStyle().Bold(true), accent),
		Item:     setFg(lipgloss.NewStyle(), text),
		Muted:    setFg(lipgloss.NewStyle(), muted),
		Key:      setFg(lipgloss.NewStyle().Bold(true), accent),

		Filter:      setFg(lipgloss.NewStyle().Bold(true), accent),
		Placeholder: setFg(lipgloss.NewStyle(), muted),

		PageActive:   icons.PageActive,
		PageInactive: icons.PageInactive,
	}

	// Without an accent colour, emphasis has to come from weight and reverse
	// video so the "mono" theme keeps the selection and the title visible.
	if c.Accent.IsZero() {
		s.Title = s.Title.Reverse(true)
		s.Cursor = s.Cursor.Reverse(true)
	}
	if promptBg.IsZero() {
		s.Prompt = s.Prompt.Reverse(true)
	}
	if c.Muted.IsZero() {
		s.Muted = s.Muted.Faint(true)
		s.Placeholder = s.Placeholder.Faint(true)
	}

	// Pre-render the pagination dots in their colours; the paginator takes
	// finished strings, not styles.
	s.PageActive = s.dot(icons.PageActive, accent)
	s.PageInactive = s.dot(icons.PageInactive, muted, true)

	return s
}

// DefaultStyles is the fallback used before a configuration is available.
func DefaultStyles() Styles {
	return NewStyles(DefaultTheme())
}

// dot renders one pagination glyph. The inactive dot dims itself when there is
// no muted colour to draw it with, matching the mono theme's reliance on weight.
func (s Styles) dot(glyph string, color lipgloss.TerminalColor, faintIfUnset ...bool) string {
	style := lipgloss.NewStyle()
	if color != nil {
		style = style.Foreground(color)
	} else if len(faintIfUnset) > 0 && faintIfUnset[0] {
		style = style.Faint(true)
	}
	return style.Render(glyph)
}

// adaptive turns a theme colour into a lipgloss colour, or nil when unset.
// lipgloss has no "no colour" value, so an unset entry is represented by a nil
// TerminalColor and simply not applied.
func adaptive(c Color) lipgloss.TerminalColor {
	if c.IsZero() {
		return nil
	}
	return lipgloss.AdaptiveColor{Light: c.Light, Dark: c.Dark}
}

// setFg applies a foreground only when the colour is set.
func setFg(s lipgloss.Style, color lipgloss.TerminalColor) lipgloss.Style {
	if color == nil {
		return s
	}
	return s.Foreground(color)
}

// setBorderFg applies a border colour only when it is set.
func setBorderFg(s lipgloss.Style, color lipgloss.TerminalColor) lipgloss.Style {
	if color == nil {
		return s
	}
	return s.BorderForeground(color)
}

func fg(s lipgloss.Style, c Color) lipgloss.Style {
	return setFg(s, adaptive(c))
}

func bg(s lipgloss.Style, c Color) lipgloss.Style {
	if c.IsZero() {
		return s
	}
	return s.Background(lipgloss.AdaptiveColor{Light: c.Light, Dark: c.Dark})
}

func bold(s lipgloss.Style) lipgloss.Style {
	return s.Bold(true)
}

// borderStyle maps a theme border name to a lipgloss border. "none" still needs
// a value here because the caller decides whether to apply it at all.
//
// The unified design draws square corners only, so "rounded" — and any name it
// does not recognise — comes out as the normal border. The variants that stay
// distinct (thick, double) already have square corners of their own.
func borderStyle(name string) lipgloss.Border {
	switch name {
	case "thick":
		return lipgloss.ThickBorder()
	case "double":
		return lipgloss.DoubleBorder()
	case "hidden", "none":
		return lipgloss.HiddenBorder()
	default:
		return lipgloss.NormalBorder()
	}
}

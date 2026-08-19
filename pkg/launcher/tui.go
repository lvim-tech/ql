package launcher

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/theme"
	"github.com/lvim-tech/ql/pkg/utils"
)

// tuiLauncher is the built-in menu: a filterable Bubble Tea list, so ql
// works from a terminal with no external menu program installed. It draws
// on stderr — the same convention as fzf — keeping stdout clean.
//
// Show builds a FRESH tea.Program every call: menus nest (group → module →
// confirm), and a cached program would fight itself over the terminal.
//
// The frame follows the lvim-tech unified layout: a top bar with the app-name
// chip and the current menu as a tab button, the scrolling list in a
// square-cornered pane, and a key-hints footer pinned to the bottom edge.
// All of it is compiled once from the configured theme (see pkg/theme).
type tuiLauncher struct {
	baseLauncher
	styles theme.Styles
}

func newTUI(cfg *config.Config) *tuiLauncher {
	sel := cfg.GetTheme()
	// A missing or malformed theme falls back to the default preset rather than
	// bricking a key-bound launcher; Resolve already returns the default on error.
	resolved, err := theme.Resolve(theme.Theme{Name: sel.Name, Border: sel.Border, Icons: sel.Icons})
	if err != nil {
		utils.Logf("[launcher tui] theme %q: %v", sel.Name, err)
	}
	return &tuiLauncher{
		baseLauncher: baseLauncher{cfg: cfg},
		styles:       theme.NewStyles(resolved),
	}
}

type tuiItem string

func (i tuiItem) FilterValue() string { return string(i) }

// tuiDelegate renders one option per line — the two-line default delegate
// reads as a form, not a menu. It carries the compiled styles so the selection
// marker, the highlighted name and the plain rows all take theme colours.
type tuiDelegate struct {
	styles theme.Styles
}

func (d tuiDelegate) Height() int                             { return 1 }
func (d tuiDelegate) Spacing() int                            { return 0 }
func (d tuiDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d tuiDelegate) Render(w io.Writer, m list.Model, index int, li list.Item) {
	it, ok := li.(tuiItem)
	if !ok {
		return
	}
	name := string(it)
	if index == m.Index() {
		// Selection: an accent cursor marker followed by the bold name. The
		// unselected rows are indented by the marker's width so names line up.
		fmt.Fprint(w, d.styles.Cursor.Render(d.styles.Icons.Cursor)+d.styles.Selected.Render(name))
		return
	}
	indent := strings.Repeat(" ", lipgloss.Width(d.styles.Icons.Cursor))
	fmt.Fprint(w, indent+d.styles.Item.Render(name))
}

type tuiModel struct {
	list   list.Model
	styles theme.Styles
	prompt string
	choice string
	width  int
	height int
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		return m, nil
	case tea.KeyMsg:
		// While the filter input is typing, every key belongs to it.
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "enter":
			if it, ok := m.list.SelectedItem().(tuiItem); ok {
				m.choice = string(it)
			}
			return m, tea.Quit
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	// The footer's hints follow the filter state, and its height decides how
	// many rows the list gets — re-fit after every message so starting or
	// leaving the filter can never push the footer off the bottom edge.
	m.layout()
	return m, cmd
}

// layout hands the list whatever rows remain between the one-line top bar,
// the pane's border and the footer. Sizing the middle — instead of letting
// the list fill the screen — is what pins the footer to the bottom edge.
func (m *tuiModel) layout() {
	if m.width == 0 {
		return
	}
	h := m.styles.Border.GetHorizontalFrameSize()
	v := m.styles.Border.GetVerticalFrameSize()
	head := lipgloss.Height(m.headerView())
	foot := lipgloss.Height(m.footerView())
	m.list.SetSize(m.width-h, max(m.height-head-foot-v, 0))
}

// headerView is the top bar: the app-name chip, then the current menu drawn
// as the active tab button. When even that does not fit, the chip alone
// survives — a truncated tab reads as a different menu, a missing one does not.
func (m tuiModel) headerView() string {
	chip := m.styles.Title.Render("ql")
	if m.prompt == "" {
		return chip
	}
	tab := m.styles.Prompt.Render(m.prompt)
	if m.width > 0 && lipgloss.Width(chip)+1+lipgloss.Width(tab) > m.width {
		return chip
	}
	return chip + " " + tab
}

// footerView is the sticky bottom bar: "[key] label" hints, the key in the
// accent colour. The set follows the filter state, because "q quits" is a
// lie while every letter belongs to the filter input.
func (m tuiModel) footerView() string {
	var parts []string
	if m.list.FilterState() == list.Filtering {
		parts = []string{
			m.hint("enter", "apply"),
			m.hint("esc", "cancel"),
		}
	} else {
		parts = []string{
			m.hint("enter", "select"),
			m.hint("/", "filter"),
			m.hint("esc", "back"),
			m.hint("q", "quit"),
		}
	}
	return wrapHints(parts, m.width)
}

// hint renders one "[k] label" pair for the footer.
func (m tuiModel) hint(k, label string) string {
	return m.styles.Key.Render("["+k+"]") + " " + m.styles.Muted.Render(label)
}

// wrapHints packs the hints into as few lines as fit the width, so a hint
// never falls off the right edge — a key that is not on screen is a key
// nobody discovers. Measured with lipgloss.Width, not len: the hints carry
// colour, and escape codes counted as characters would cut each line short.
func wrapHints(parts []string, width int) string {
	if width < 20 {
		width = 20
	}
	var lines []string
	cur, curW := "", 0
	for _, p := range parts {
		w := lipgloss.Width(p)
		switch {
		case cur == "":
			cur, curW = p, w
		case curW+2+w <= width:
			cur, curW = cur+"  "+p, curW+2+w
		default:
			lines = append(lines, cur)
			cur, curW = p, w
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return strings.Join(lines, "\n")
}

// View stacks top bar, bordered list and footer. The list pads itself to the
// height layout() granted it, so the three parts always sum to the terminal
// height — which is what keeps the footer on the bottom edge while only the
// middle scrolls.
func (m tuiModel) View() string {
	if m.width == 0 {
		return ""
	}
	// The pane is stretched to the terminal's width — left to its own devices
	// lipgloss shrink-wraps the border around the longest row, and a frame
	// that hugs short menu names reads as floating rather than laid out.
	pane := m.styles.Border.Width(m.width - m.styles.Border.GetHorizontalBorderSize())
	return m.headerView() + "\n" +
		pane.Render(m.list.View()) + "\n" +
		m.footerView()
}

// newTUIList builds the menu list; split out so the model is testable
// without a terminal. All colours come from the compiled styles.
func newTUIList(options []string, prompt string, styles theme.Styles) list.Model {
	items := make([]list.Item, len(options))
	for i, option := range options {
		items[i] = tuiItem(option)
	}

	lst := list.New(items, tuiDelegate{styles: styles}, 0, 0)
	// The prompt lives in the top bar the model draws, and the key hints in
	// its sticky footer — the list itself renders neither.
	lst.Title = prompt
	lst.SetShowTitle(false)
	lst.SetShowStatusBar(false)
	lst.SetShowHelp(false)

	// The filter's prompt and cursor take the theme's accent. bubbles reads
	// Styles.Filter* exactly once, inside list.New(), so setting the struct
	// afterwards reaches nothing — the input otherwise keeps the library's
	// defaults: a neon yellow prompt (#ECFD65) and a pink cursor (#EE6FF8),
	// neither of which belongs to any palette here.
	lst.Styles.FilterPrompt = styles.Filter
	lst.Styles.FilterCursor = styles.Filter
	lst.FilterInput.PromptStyle = styles.Filter
	lst.FilterInput.Cursor.Style = styles.Filter
	lst.FilterInput.PlaceholderStyle = styles.Placeholder

	// Pagination dots in the theme's accent / muted colours.
	lst.Paginator.ActiveDot = styles.PageActive
	lst.Paginator.InactiveDot = styles.PageInactive
	return lst
}

func (l *tuiLauncher) Show(options []string, prompt string) (string, error) {
	lst := newTUIList(options, prompt, l.styles)
	program := tea.NewProgram(
		tuiModel{list: lst, styles: l.styles, prompt: prompt},
		tea.WithOutput(os.Stderr),
		tea.WithAltScreen(),
	)
	final, err := program.Run()
	if err != nil {
		utils.Logf("[launcher tui] %v", err)
		return "", fmt.Errorf("tui launcher: %w", err)
	}

	choice := final.(tuiModel).choice
	if choice == "" {
		return "", fmt.Errorf("no selection made")
	}
	return choice, nil
}

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
// The look — title badge, border, selection, hints, pagination — is compiled
// once from the configured theme into styles, mirroring the lvim-tech unified
// design (see pkg/theme).
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
	choice string
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Leave room for the border and padding the View() draws around the
		// list, so the two never overlap the terminal edge.
		h := m.styles.Border.GetHorizontalFrameSize()
		v := m.styles.Border.GetVerticalFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
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
	return m, cmd
}

// View wraps the list in the theme's border — the single, focused pane.
func (m tuiModel) View() string { return m.styles.Border.Render(m.list.View()) }

// newTUIList builds the menu list; split out so the model is testable
// without a terminal. All colours come from the compiled styles.
func newTUIList(options []string, prompt string, styles theme.Styles) list.Model {
	items := make([]list.Item, len(options))
	for i, option := range options {
		items[i] = tuiItem(option)
	}

	lst := list.New(items, tuiDelegate{styles: styles}, 0, 0)
	lst.Title = prompt
	// The title is drawn as a badge (theme title_fg on accent), the same
	// prompt convention as the unified design.
	lst.Styles.Title = styles.Title
	lst.SetShowStatusBar(false)
	lst.SetShowHelp(true)

	// A muted key-hints line at the bottom (enter select, / filter, esc/q back).
	lst.Help.Styles.ShortKey = styles.Muted
	lst.Help.Styles.ShortDesc = styles.Muted
	lst.Help.Styles.ShortSeparator = styles.Muted
	lst.Help.Styles.FullKey = styles.Muted
	lst.Help.Styles.FullDesc = styles.Muted
	lst.Help.Styles.FullSeparator = styles.Muted

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
		tuiModel{list: lst, styles: l.styles},
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

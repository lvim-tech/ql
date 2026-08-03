package launcher

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/utils"
)

// tuiLauncher is the built-in menu: a filterable Bubble Tea list, so ql
// works from a terminal with no external menu program installed. It draws
// on stderr — the same convention as fzf — keeping stdout clean.
//
// Show builds a FRESH tea.Program every call: menus nest (group → module →
// confirm), and a cached program would fight itself over the terminal.
type tuiLauncher struct {
	baseLauncher
}

func newTUI(cfg *config.Config) *tuiLauncher {
	return &tuiLauncher{baseLauncher: baseLauncher{cfg: cfg}}
}

type tuiItem string

func (i tuiItem) FilterValue() string { return string(i) }

var (
	tuiSelected = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	tuiTitle    = lipgloss.NewStyle().Bold(true)
	tuiDim      = lipgloss.NewStyle().Faint(true)
)

// tuiDelegate renders one option per line — the two-line default delegate
// reads as a form, not a menu.
type tuiDelegate struct{}

func (d tuiDelegate) Height() int                             { return 1 }
func (d tuiDelegate) Spacing() int                            { return 0 }
func (d tuiDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d tuiDelegate) Render(w io.Writer, m list.Model, index int, li list.Item) {
	it, ok := li.(tuiItem)
	if !ok {
		return
	}
	if index == m.Index() {
		fmt.Fprint(w, tuiSelected.Render("│ "+string(it)))
		return
	}
	fmt.Fprint(w, "  "+string(it))
}

type tuiModel struct {
	list   list.Model
	choice string
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
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

func (m tuiModel) View() string { return m.list.View() }

// newTUIList builds the menu list; split out so the model is testable
// without a terminal.
func newTUIList(options []string, prompt string) list.Model {
	items := make([]list.Item, len(options))
	for i, option := range options {
		items[i] = tuiItem(option)
	}

	lst := list.New(items, tuiDelegate{}, 0, 0)
	lst.Title = prompt
	lst.Styles.Title = tuiTitle
	lst.SetShowStatusBar(false)
	lst.SetShowHelp(true)
	lst.Paginator.ActiveDot = tuiSelected.Render("● ")
	lst.Paginator.InactiveDot = tuiDim.Render("○ ")
	return lst
}

func (l *tuiLauncher) Show(options []string, prompt string) (string, error) {
	lst := newTUIList(options, prompt)
	program := tea.NewProgram(tuiModel{list: lst}, tea.WithOutput(os.Stderr), tea.WithAltScreen())
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

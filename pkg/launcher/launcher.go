// Package launcher provides context and implementations for various application launchers.
// It supports rofi, dmenu, fzf, bemenu, fuzzel and a built-in Bubble Tea TUI, allowing
// command modules to display interactive menus without direct dependencies on specific
// launcher implementations.
package launcher

import (
	"fmt"
	"os"
	"os/exec"

	"golang.org/x/term"

	"github.com/lvim-tech/ql/pkg/config"
)

// Launcher interface defines launcher behavior
type Launcher interface {
	Show(options []string, prompt string) (string, error)
	Config() *config.Config
	IsDirectLaunch() bool
	SetDirectLaunch(bool)
	Args() []string
	SetArgs([]string)
}

// baseLauncher provides common functionality for all launchers
type baseLauncher struct {
	cfg          *config.Config
	directLaunch bool
	args         []string
}

func (b *baseLauncher) Config() *config.Config {
	return b.cfg
}

func (b *baseLauncher) IsDirectLaunch() bool {
	return b.directLaunch
}

func (b *baseLauncher) SetDirectLaunch(direct bool) {
	b.directLaunch = direct
}

func (b *baseLauncher) Args() []string {
	return b.args
}

func (b *baseLauncher) SetArgs(args []string) {
	b.args = args
}

// New creates a new launcher instance. An unknown name is an error, not a
// silent rofi: default.toml shipped "auto" for months while the old default
// arm quietly ran rofi on machines that did not have it.
func New(name string, cfg *config.Config) (Launcher, error) {
	if name == "" || name == "auto" {
		name = pickAuto()
	}
	if name == "tui" {
		return newTUI(cfg), nil
	}
	if _, ok := externals[name]; ok {
		return newExternal(name, cfg), nil
	}
	return nil, fmt.Errorf("unknown launcher %q (rofi, dmenu, fzf, bemenu, fuzzel, tui, auto)", name)
}

// pickAuto resolves "auto": the built-in TUI when run from a terminal,
// otherwise the first installed graphical menu. rofi leads on both X11 and
// Wayland — it works on either and is what most configurations are themed
// for; the Wayland-native menus are the fallback when it is absent, not a
// silent replacement for it. Override with default_launcher.
func pickAuto() string {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		return "tui"
	}
	order := []string{"rofi", "dmenu", "fzf"}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		order = []string{"rofi", "fuzzel", "bemenu", "dmenu", "fzf"}
	}
	for _, name := range order {
		if _, err := exec.LookPath(name); err == nil {
			return name
		}
	}
	return "rofi"
}

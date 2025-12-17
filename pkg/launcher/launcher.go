package launcher

import (
	"errors"

	"github.com/lvim-tech/ql/pkg/config"
)

var (
	// ErrCancelled се връща когато потребителят отмени операцията
	ErrCancelled = errors.New("cancelled by user")

	// ErrNoLauncher се връща когато няма конфигуриран launcher
	ErrNoLauncher = errors.New("no launcher configured")
)

// Launcher интерфейс за различни menu системи
type Launcher interface {
	Show(options []string, prompt string) (string, error)
	Name() string
	Args() []string
}

// IsCancelled проверява дали грешката е от отказ
func IsCancelled(err error) bool {
	return errors.Is(err, ErrCancelled)
}

// NewContextFromFlags създава launcher context от command line flags
func NewContextFromFlags(flags map[string]bool) *Context {
	cfg := config.Get()

	// Провери flags в priority order
	if flags["r"] { // rofi
		return &Context{Launcher: NewRofi(cfg.Launchers.Rofi.Args)}
	}
	if flags["d"] { // dmenu
		return &Context{Launcher: NewDmenu(cfg.Launchers.Dmenu.Args)}
	}
	if flags["f"] { // fzf
		return &Context{Launcher: NewFzf(cfg.Launchers.Fzf.Args)}
	}
	if flags["b"] { // bemenu
		return &Context{Launcher: NewBemenu(cfg.Launchers.Bemenu.Args)}
	}
	if flags["z"] { // fuzzel
		return &Context{Launcher: NewFuzzel(cfg.Launchers.Fuzzel.Args)}
	}

	// Използвай default launcher от config
	switch cfg.DefaultLauncher {
	case "rofi":
		return &Context{Launcher: NewRofi(cfg.Launchers.Rofi.Args)}
	case "dmenu":
		return &Context{Launcher: NewDmenu(cfg.Launchers.Dmenu.Args)}
	case "fzf":
		return &Context{Launcher: NewFzf(cfg.Launchers.Fzf.Args)}
	case "bemenu":
		return &Context{Launcher: NewBemenu(cfg.Launchers.Bemenu.Args)}
	case "fuzzel":
		return &Context{Launcher: NewFuzzel(cfg.Launchers.Fuzzel.Args)}
	default:
		// Fallback to rofi
		return &Context{Launcher: NewRofi(cfg.Launchers.Rofi.Args)}
	}
}

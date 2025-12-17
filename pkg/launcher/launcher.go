// Package launcher provides context and implementations for various application launchers.
// It supports rofi, dmenu, fzf, bemenu, and fuzzel, allowing command modules to display
// interactive menus without direct dependencies on specific launcher implementations.
package launcher

import (
	"github.com/lvim-tech/ql/pkg/config"
)

// Launcher interface defines launcher behavior
type Launcher interface {
	Show(options []string, prompt string) (string, error)
	Config() *config.Config
}

// New creates a new launcher instance
func New(name string, cfg *config.Config) (Launcher, error) {
	switch name {
	case "rofi":
		return NewRofi(cfg), nil
	case "dmenu":
		return NewDmenu(cfg), nil
	case "fzf":
		return NewFzf(cfg), nil
	case "bemenu":
		return NewBemenu(cfg), nil
	case "fuzzel":
		return NewFuzzel(cfg), nil
	default:
		return NewRofi(cfg), nil
	}
}

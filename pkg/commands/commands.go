package commands

import (
	"github.com/lvim-tech/ql/pkg/config"
)

// Command представлява команда
type Command struct {
	Name        string
	Description string
	Run         func(LauncherContext) error
}

// LauncherContext interface за launcher
type LauncherContext interface {
	Show(options []string, prompt string) (string, error)
	Config() *config.Config
}

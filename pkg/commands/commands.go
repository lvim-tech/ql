// Package commands provides the core command system for ql launcher.
// It defines the Command type, CommandResult for navigation control,
// and LauncherContext interface for command execution.
package commands

import (
	"github.com/lvim-tech/ql/pkg/config"
)

// CommandResult represents the result of command execution
type CommandResult struct {
	Success bool
	Error   error
}

// Command представлява команда
type Command struct {
	Name        string
	Description string
	Run         func(LauncherContext) CommandResult
}

// LauncherContext interface за launcher
type LauncherContext interface {
	Show(options []string, prompt string) (string, error)
	Config() *config.Config
}

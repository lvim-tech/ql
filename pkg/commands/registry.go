// Package commands provides a registry system for ql commands.
// Commands can register themselves on initialization and be discovered dynamically.
package commands

import (
	"github.com/lvim-tech/ql/pkg/launcher"
)

// Command описва една команда
type Command struct {
	Name        string
	Description string
	Run         func(*launcher.Context) error
}

var registry = make(map[string]Command)

// Register регистрира команда
func Register(cmd Command) {
	registry[cmd.Name] = cmd
}

// Find намира команда по име
func Find(name string) *Command {
	if cmd, ok := registry[name]; ok {
		return &cmd
	}
	return nil
}

// List връща всички регистрирани команди
func List() []Command {
	var commands []Command
	for _, cmd := range registry {
		commands = append(commands, cmd)
	}
	return commands
}

// Names връща имената на всички команди
func Names() []string {
	var names []string
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// Package hub provides a central menu for accessing all ql commands.
package hub

import (
	"fmt"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/launcher"
)

func init() {
	commands.Register(commands.Command{
		Name:        "hub",
		Description: "Main command hub",
		Run:         Run,
	})
}

// Run показва главното меню с всички команди
func Run(ctx *launcher.Context) error {
	cfg := config.Get()

	// Вземи всички регистрирани команди
	allCommands := commands.List()

	// Филтрирай само enabled команди (без hub и test)
	var availableCommands []commands.Command
	for _, cmd := range allCommands {
		// Skip hub и test
		if cmd.Name == "hub" || cmd.Name == "test" {
			continue
		}

		// Провери дали е enabled в config
		if !isCommandEnabled(cfg, cmd.Name) {
			continue
		}

		availableCommands = append(availableCommands, cmd)
	}

	if len(availableCommands) == 0 {
		return fmt.Errorf("no commands available (check your config)")
	}

	// Подготви опции за меню
	var options []string
	for _, cmd := range availableCommands {
		options = append(options, cmd.Name)
	}

	// Покажи меню
	choice, err := ctx.Show(options, "Select Command")
	if err != nil {
		return err
	}

	// Намери и изпълни избраната команда
	for _, cmd := range availableCommands {
		if cmd.Name == choice {
			return cmd.Run(ctx)
		}
	}

	return fmt.Errorf("command not found: %s", choice)
}

// isCommandEnabled проверява дали команда е enabled в config
func isCommandEnabled(cfg *config.Config, cmdName string) bool {
	switch cmdName {
	case "power":
		return cfg.Commands.Power.Enabled
	case "screenshot":
		return cfg.Commands.Screenshot.Enabled
	default:
		// По default всички команди са enabled
		return true
	}
}

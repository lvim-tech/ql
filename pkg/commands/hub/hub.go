// Package hub provides a central menu for accessing all ql commands.
package hub

import (
	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/launcher"
)

func init() {
	commands.Register(commands.Command{
		Name:        "hub",
		Description: "Main command menu",
		Run:         Run,
	})
}

func Run(ctx *launcher.Context) error {
	cfg := config.Get()

	// Събери всички enabled команди (без hub)
	var options []string
	for _, cmd := range commands.List() {
		if cmd.Name == "hub" {
			continue
		}

		// Провери дали е enabled
		if !isCommandEnabled(cfg, cmd.Name) {
			continue
		}

		options = append(options, cmd.Name)
	}

	// Покажи меню
	choice, err := ctx.Show(options, "ql")
	if err != nil {
		return err
	}

	// Изпълни избраната команда
	cmd := commands.Find(choice)
	if cmd == nil {
		return launcher.ErrCancelled
	}

	return cmd.Run(ctx)
}

func isCommandEnabled(cfg *config.Config, cmdName string) bool {
	switch cmdName {
	case "power":
		return cfg.Commands.Power.Enabled
	case "screenshot":
		return cfg.Commands.Screenshot.Enabled
	case "radio":
		return cfg.Commands.Radio.Enabled
	default:
		return true
	}
}

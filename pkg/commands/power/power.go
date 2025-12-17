// Package power provides power management functionality for ql.
// It includes options for logout, suspend, hibernate, reboot, and shutdown.
package power

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/launcher"
)

func init() {
	commands.Register(commands.Command{
		Name:        "power",
		Description: "Power management menu",
		Run:         Run,
	})
}

func Run(ctx *launcher.Context) error {
	cfg := config.Get()
	powerCfg := cfg.Commands.Power

	// Провери дали е enabled
	if !powerCfg.Enabled {
		return fmt.Errorf("power module is disabled in config")
	}

	// Създай опции базирани на конфигурацията
	var options []string

	if powerCfg.ShowLogout {
		options = append(options, "Logout")
	}
	if powerCfg.ShowSuspend {
		options = append(options, "Suspend")
	}
	if powerCfg.ShowHibernate {
		options = append(options, "Hibernate")
	}
	if powerCfg.ShowReboot {
		options = append(options, "Reboot")
	}
	if powerCfg.ShowShutdown {
		options = append(options, "Shutdown")
	}

	if len(options) == 0 {
		return fmt.Errorf("no power options enabled in config")
	}

	// Покажи меню
	choice, err := ctx.Show(options, "Power Menu")
	if err != nil {
		return err
	}

	// Обработи избора
	return handleChoice(choice, &powerCfg)
}

func handleChoice(choice string, cfg *config.PowerConfig) error {
	choice = strings.TrimSpace(choice)

	switch choice {
	case "Logout":
		return handleAction(cfg.LogoutCommand, cfg.ConfirmLogout)
	case "Suspend":
		return handleAction(cfg.SuspendCommand, cfg.ConfirmSuspend)
	case "Hibernate":
		return handleAction(cfg.HibernateCommand, cfg.ConfirmHibernate)
	case "Reboot":
		return handleAction(cfg.RebootCommand, cfg.ConfirmReboot)
	case "Shutdown":
		return handleAction(cfg.ShutdownCommand, cfg.ConfirmShutdown)
	default:
		return fmt.Errorf("unknown choice: %s", choice)
	}
}

func handleAction(command string, needsConfirm bool) error {
	// TODO: Implement confirmation dialog when needsConfirm is true
	_ = needsConfirm

	return executeCommand(command)
}

func executeCommand(command string) error {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	return cmd.Run()
}

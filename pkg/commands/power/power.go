// Package power provides power management functionality for ql.
// It includes options for logout, suspend, hibernate, reboot, and shutdown.
package power

import (
	"fmt"
	"os/exec"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/mitchellh/mapstructure"
)

func init() {
	commands.Register(commands.Command{
		Name:        "power",
		Description: "Power management",
		Run:         Run,
	})
}

func Run(ctx commands.LauncherContext) error {
	// Извличаме config директно
	cfgInterface := ctx.Config().GetPowerConfig()
	if cfgInterface == nil {
		return fmt.Errorf("power config not found")
	}

	// Decode с WeaklyTypedInput
	var cfg Config
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           &cfg,
	})
	if err != nil {
		cfg = DefaultConfig()
	} else if err := decoder.Decode(cfgInterface); err != nil {
		cfg = DefaultConfig()
	}

	if !cfg.Enabled {
		return fmt.Errorf("power module is disabled in config")
	}

	// Build menu options
	var options []string
	actionMap := make(map[string]func() error)

	if cfg.ShowLogout {
		options = append(options, "Logout")
		actionMap["Logout"] = func() error {
			return executeWithConfirmation(ctx, "Logout", cfg.LogoutCommand, cfg.ConfirmLogout)
		}
	}

	if cfg.ShowSuspend {
		options = append(options, "Suspend")
		actionMap["Suspend"] = func() error {
			return executeWithConfirmation(ctx, "Suspend", cfg.SuspendCommand, cfg.ConfirmSuspend)
		}
	}

	if cfg.ShowHibernate {
		options = append(options, "Hibernate")
		actionMap["Hibernate"] = func() error {
			return executeWithConfirmation(ctx, "Hibernate", cfg.HibernateCommand, cfg.ConfirmHibernate)
		}
	}

	if cfg.ShowReboot {
		options = append(options, "Reboot")
		actionMap["Reboot"] = func() error {
			return executeWithConfirmation(ctx, "Reboot", cfg.RebootCommand, cfg.ConfirmReboot)
		}
	}

	if cfg.ShowShutdown {
		options = append(options, "Shutdown")
		actionMap["Shutdown"] = func() error {
			return executeWithConfirmation(ctx, "Shutdown", cfg.ShutdownCommand, cfg.ConfirmShutdown)
		}
	}

	if len(options) == 0 {
		return fmt.Errorf("no power options enabled")
	}

	// Show menu
	choice, err := ctx.Show(options, "Power")
	if err != nil {
		return err
	}

	// Execute action
	action, ok := actionMap[choice]
	if !ok {
		return fmt.Errorf("unknown action: %s", choice)
	}

	return action()
}

func executeWithConfirmation(ctx commands.LauncherContext, action, command string, needsConfirm bool) error {
	if needsConfirm {
		confirmOptions := []string{"Yes", "No"}
		choice, err := ctx.Show(confirmOptions, fmt.Sprintf("Confirm %s?", action))
		if err != nil {
			return err
		}
		if choice != "Yes" {
			return fmt.Errorf("action cancelled")
		}
	}

	cmd := exec.Command("sh", "-c", command)
	return cmd.Run()
}

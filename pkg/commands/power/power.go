// Package power provides power management functionality for ql.
// It includes options for logout, suspend, hibernate, reboot, and shutdown.
package power

import (
	"fmt"
	"os"
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

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfgInterface := ctx.Config().GetPowerConfig()

	var cfg Config
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           &cfg,
	})
	if err != nil {
		cfg = DefaultConfig()
	} else {
		if decodeErr := decoder.Decode(cfgInterface); decodeErr != nil {
			cfg = DefaultConfig()
		}
	}

	if !cfg.Enabled {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("power module is disabled in config"),
		}
	}

	for {
		var options []string
		actionMap := make(map[string]func() error)

		options = append(options, "← Back")

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

		if len(options) == 1 {
			return commands.CommandResult{
				Success: false,
				Error:   fmt.Errorf("no power options enabled"),
			}
		}

		choice, err := ctx.Show(options, "Power")
		if err != nil {
			return commands.CommandResult{Success: false}
		}

		if choice == "← Back" {
			return commands.CommandResult{Success: false}
		}

		action, ok := actionMap[choice]
		if !ok {
			showErrorNotification("Power Error", fmt.Sprintf("Unknown action: %s", choice))
			continue
		}

		err = action()
		if err != nil {
			showErrorNotification("Power Error", err.Error())
			continue
		}

		return commands.CommandResult{Success: true}
	}
}

func executeWithConfirmation(ctx commands.LauncherContext, action, command string, needsConfirm bool) error {
	if needsConfirm {
		confirmOptions := []string{"← Back", "Yes", "No"}
		choice, err := ctx.Show(confirmOptions, fmt.Sprintf("Confirm %s?", action))
		if err != nil {
			return fmt.Errorf("cancelled")
		}
		if choice == "← Back" || choice == "No" {
			return fmt.Errorf("cancelled")
		}
		if choice != "Yes" {
			return fmt.Errorf("cancelled")
		}
	}

	cmd := exec.Command("sh", "-c", command)
	return cmd.Run()
}

func showErrorNotification(title, message string) {
	if _, err := exec.LookPath("dunstify"); err == nil {
		cmd := exec.Command("dunstify",
			"-u", "critical",
			"-t", "10000",
			title,
			message)
		cmd.Env = os.Environ()
		cmd.Start()
		return
	}

	if _, err := exec.LookPath("notify-send"); err == nil {
		cmd := exec.Command("notify-send",
			"-u", "critical",
			"-t", "10000",
			title,
			message)
		cmd.Env = os.Environ()
		cmd.Start()
		return
	}
}

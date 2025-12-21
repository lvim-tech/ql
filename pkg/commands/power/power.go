// Package power provides power management functionality for ql.
package power

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/utils"
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

	notifCfg := ctx.Config().GetNotificationConfig()

	args := ctx.Args()
	if len(args) > 0 {
		return executeDirectCommand(args[0], &cfg, &notifCfg)
	}

	for {
		mainChoice, err := showPowerMainMenu(ctx, &cfg)
		if err != nil {
			return commands.CommandResult{Success: false}
		}

		if mainChoice == "← Back" {
			return commands.CommandResult{
				Success: false,
				Error:   commands.ErrBack,
			}
		}

		actionResult := executePowerAction(ctx, &cfg, mainChoice)

		if actionResult.Success {
			return commands.CommandResult{Success: true}
		}

		if actionResult.Error != nil && actionResult.Error != commands.ErrBack {
			return commands.CommandResult{Success: false}
		}

		if actionResult.Error != nil {
			utils.ShowErrorNotificationWithConfig(&notifCfg, "Power Error", actionResult.Error.Error())
		}
		continue
	}
}

func executeDirectCommand(action string, cfg *Config, notifCfg *config.NotificationConfig) commands.CommandResult {
	var err error

	switch strings.ToLower(action) {
	case "logout":
		err = executeLogout(cfg)
	case "suspend":
		err = executeSuspend(cfg)
	case "hibernate":
		err = executeHibernate(cfg)
	case "reboot":
		err = executeReboot(cfg)
	case "shutdown":
		err = executeShutdown(cfg)
	default:
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("unknown power action: %s (available: logout, suspend, hibernate, reboot, shutdown)", action),
		}
	}

	if err != nil {
		utils.ShowErrorNotificationWithConfig(notifCfg, "Power Error", err.Error())
		return commands.CommandResult{Success: false, Error: err}
	}

	return commands.CommandResult{Success: true}
}

func showPowerMainMenu(ctx commands.LauncherContext, cfg *Config) (string, error) {
	var options []string

	if !ctx.IsDirectLaunch() {
		options = append(options, "← Back")
	}

	if cfg.ShowLogout {
		options = append(options, "Logout")
	}
	if cfg.ShowSuspend {
		options = append(options, "Suspend")
	}
	if cfg.ShowHibernate {
		options = append(options, "Hibernate")
	}
	if cfg.ShowReboot {
		options = append(options, "Reboot")
	}
	if cfg.ShowShutdown {
		options = append(options, "Shutdown")
	}

	return ctx.Show(options, "Power")
}

func executePowerAction(ctx commands.LauncherContext, cfg *Config, action string) commands.CommandResult {
	switch action {
	case "Logout":
		if cfg.ConfirmLogout {
			confirmed, confirmErr := confirmAction(ctx, "Logout")
			if confirmErr != nil {
				return commands.CommandResult{Success: false, Error: fmt.Errorf("ESC")}
			}
			if !confirmed {
				return commands.CommandResult{Success: false, Error: commands.ErrBack}
			}
		}
		if err := executeLogout(cfg); err != nil {
			return commands.CommandResult{Success: false, Error: err}
		}
		return commands.CommandResult{Success: true}

	case "Suspend":
		if cfg.ConfirmSuspend {
			confirmed, confirmErr := confirmAction(ctx, "Suspend")
			if confirmErr != nil {
				return commands.CommandResult{Success: false, Error: fmt.Errorf("ESC")}
			}
			if !confirmed {
				return commands.CommandResult{Success: false, Error: commands.ErrBack}
			}
		}
		if err := executeSuspend(cfg); err != nil {
			return commands.CommandResult{Success: false, Error: err}
		}
		return commands.CommandResult{Success: true}

	case "Hibernate":
		if cfg.ConfirmHibernate {
			confirmed, confirmErr := confirmAction(ctx, "Hibernate")
			if confirmErr != nil {
				return commands.CommandResult{Success: false, Error: fmt.Errorf("ESC")}
			}
			if !confirmed {
				return commands.CommandResult{Success: false, Error: commands.ErrBack}
			}
		}
		if err := executeHibernate(cfg); err != nil {
			return commands.CommandResult{Success: false, Error: err}
		}
		return commands.CommandResult{Success: true}

	case "Reboot":
		if cfg.ConfirmReboot {
			confirmed, confirmErr := confirmAction(ctx, "Reboot")
			if confirmErr != nil {
				return commands.CommandResult{Success: false, Error: fmt.Errorf("ESC")}
			}
			if !confirmed {
				return commands.CommandResult{Success: false, Error: commands.ErrBack}
			}
		}
		if err := executeReboot(cfg); err != nil {
			return commands.CommandResult{Success: false, Error: err}
		}
		return commands.CommandResult{Success: true}

	case "Shutdown":
		if cfg.ConfirmShutdown {
			confirmed, confirmErr := confirmAction(ctx, "Shutdown")
			if confirmErr != nil {
				return commands.CommandResult{Success: false, Error: fmt.Errorf("ESC")}
			}
			if !confirmed {
				return commands.CommandResult{Success: false, Error: commands.ErrBack}
			}
		}
		if err := executeShutdown(cfg); err != nil {
			return commands.CommandResult{Success: false, Error: err}
		}
		return commands.CommandResult{Success: true}

	default:
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("unknown action: %s", action),
		}
	}
}

func confirmAction(ctx commands.LauncherContext, action string) (bool, error) {
	options := []string{"No", "Yes"}
	choice, err := ctx.Show(options, fmt.Sprintf("Confirm %s? ", action))
	if err != nil {
		return false, err
	}
	return choice == "Yes", nil
}

func executeLogout(cfg *Config) error {
	cmd := exec.Command("sh", "-c", os.ExpandEnv(cfg.LogoutCommand))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("logout failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func executeSuspend(cfg *Config) error {
	cmd := exec.Command("sh", "-c", os.ExpandEnv(cfg.SuspendCommand))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("suspend failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func executeHibernate(cfg *Config) error {
	cmd := exec.Command("sh", "-c", os.ExpandEnv(cfg.HibernateCommand))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hibernate failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func executeReboot(cfg *Config) error {
	cmd := exec.Command("sh", "-c", os.ExpandEnv(cfg.RebootCommand))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("reboot failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func executeShutdown(cfg *Config) error {
	cmd := exec.Command("sh", "-c", os.ExpandEnv(cfg.ShutdownCommand))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("shutdown failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

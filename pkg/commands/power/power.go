// Package power provides power management functionality for ql.
package power

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/utils"
)

func init() {
	commands.Register(commands.Command{
		Name:        "power",
		Description: "Power management",
		Run:         Run,
	})
}

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfg := commands.DecodeConfig(ctx, "power", DefaultConfig())

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

		if actionResult.Error != nil && !errors.Is(actionResult.Error, commands.ErrBack) {
			return commands.CommandResult{Success: false}
		}

		if actionResult.Error != nil {
			utils.ShowErrorNotificationWithConfig(&notifCfg, "Power Error", actionResult.Error.Error())
		}
		continue
	}
}

func executeDirectCommand(action string, cfg *Config, notifCfg *config.NotificationConfig) commands.CommandResult {
	// The argument is matched against the menu labels case-insensitively
	// rather than title-cased with action[:1], which panicked with a slice
	// bounds error on an empty argument — and `ql power ""` produces exactly
	// that.
	label, spec, ok := lookupPowerCommand(cfg, action)
	if !ok {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("unknown power action: %s (available: logout, suspend, hibernate, reboot, shutdown)", action),
		}
	}

	if err := runPowerCommand(label, spec.Command); err != nil {
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

// powerCommandSpec is what a menu label resolves to: whether the action needs
// confirming, and the configured command that performs it.
type powerCommandSpec struct {
	Confirm bool
	Command string
}

// powerCommands maps a menu label to its confirm flag and configured
// command. One table, one arm below: the five actions differed only in
// these two values, and five copies of the confirm dance were 125 lines of
// the same 25.
func powerCommands(cfg *Config) map[string]powerCommandSpec {
	return map[string]powerCommandSpec{
		"Logout":    {cfg.ConfirmLogout, cfg.LogoutCommand},
		"Suspend":   {cfg.ConfirmSuspend, cfg.SuspendCommand},
		"Hibernate": {cfg.ConfirmHibernate, cfg.HibernateCommand},
		"Reboot":    {cfg.ConfirmReboot, cfg.RebootCommand},
		"Shutdown":  {cfg.ConfirmShutdown, cfg.ShutdownCommand},
	}
}

// lookupPowerCommand finds the menu label matching action, ignoring case, and
// returns its spec. The label is returned too because the rest of the module
// speaks in labels ("Shutdown"), not in whatever the user typed.
func lookupPowerCommand(cfg *Config, action string) (string, powerCommandSpec, bool) {
	table := powerCommands(cfg)
	for label, spec := range table {
		if strings.EqualFold(label, action) {
			return label, spec, true
		}
	}
	return "", powerCommandSpec{}, false
}

func executePowerAction(ctx commands.LauncherContext, cfg *Config, action string) commands.CommandResult {
	spec, ok := powerCommands(cfg)[action]
	if !ok {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("unknown action: %s", action),
		}
	}

	if spec.Confirm {
		choice, err := confirmAction(ctx, action)
		if err != nil {
			return commands.CommandResult{Success: false, Error: commands.ErrCancelled}
		}
		switch choice {
		case "← Back":
			return commands.CommandResult{Success: false, Error: commands.ErrBack}
		case "No":
			return commands.CommandResult{Success: true}
		}
	}

	if err := runPowerCommand(action, spec.Command); err != nil {
		return commands.CommandResult{Success: false, Error: err}
	}
	return commands.CommandResult{Success: true}
}

func confirmAction(ctx commands.LauncherContext, action string) (string, error) {
	options := []string{"← Back", "No", "Yes"}
	choice, err := ctx.Show(options, fmt.Sprintf("Confirm %s?", action))
	if err != nil {
		return "", err
	}
	return choice, nil
}

// runPowerCommand runs the configured command through sh — that is the
// point: users configure arbitrary shell for these actions.
func runPowerCommand(action, command string) error {
	cmd := exec.Command("sh", "-c", os.ExpandEnv(command))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %s", strings.ToLower(action), strings.TrimSpace(string(output)))
	}
	return nil
}

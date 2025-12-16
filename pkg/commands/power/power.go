// Package power provides power management functionality for ql.
// It includes options for logout, suspend, hibernate, reboot, and shutdown.
package power

import (
	"fmt"
	"os"
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

// Action представлява power action
type Action struct {
	Name          string
	Label         string
	ConfigShow    bool
	ConfigConfirm bool
	Command       string
}

// Run показва power management menu
func Run(ctx *launcher.Context) error {
	cfg := config.Get()

	// Дефинирай всички възможни actions
	allActions := []Action{
		{
			Name:          "logout",
			Label:         "Logout",
			ConfigShow:    cfg.Commands.Power.ShowLogout,
			ConfigConfirm: cfg.Commands.Power.ConfirmLogout,
			Command:       cfg.Commands.Power.LogoutCommand,
		},
		{
			Name:          "suspend",
			Label:         "Suspend",
			ConfigShow:    cfg.Commands.Power.ShowSuspend,
			ConfigConfirm: cfg.Commands.Power.ConfirmSuspend,
			Command:       cfg.Commands.Power.SuspendCommand,
		},
		{
			Name:          "hibernate",
			Label:         "Hibernate",
			ConfigShow:    cfg.Commands.Power.ShowHibernate,
			ConfigConfirm: cfg.Commands.Power.ConfirmHibernate,
			Command:       cfg.Commands.Power.HibernateCommand,
		},
		{
			Name:          "reboot",
			Label:         "Reboot",
			ConfigShow:    cfg.Commands.Power.ShowReboot,
			ConfigConfirm: cfg.Commands.Power.ConfirmReboot,
			Command:       cfg.Commands.Power.RebootCommand,
		},
		{
			Name:          "shutdown",
			Label:         "Shutdown",
			ConfigShow:    cfg.Commands.Power.ShowShutdown,
			ConfigConfirm: cfg.Commands.Power.ConfirmShutdown,
			Command:       cfg.Commands.Power.ShutdownCommand,
		},
	}

	// Филтрирай само показваните actions
	var availableActions []Action
	var options []string

	for _, action := range allActions {
		if action.ConfigShow {
			availableActions = append(availableActions, action)
			options = append(options, action.Label)
		}
	}

	if len(options) == 0 {
		return fmt.Errorf("no power options enabled in config")
	}

	// Покажи menu
	choice, err := ctx.Show(options, "Power Menu")
	if err != nil {
		return err
	}

	// Намери избраното action
	var selectedAction *Action
	for i := range availableActions {
		if availableActions[i].Label == choice {
			selectedAction = &availableActions[i]
			break
		}
	}

	if selectedAction == nil {
		return fmt.Errorf("action not found: %s", choice)
	}

	// Провери дали се изисква потвърждение
	if selectedAction.ConfigConfirm {
		confirmed, err := confirm(ctx, fmt.Sprintf("%s now?", selectedAction.Label))
		if err != nil {
			return err
		}
		if !confirmed {
			return nil
		}
	}

	// Изпълни командата
	return executeCommand(selectedAction.Command)
}

// confirm пита потребителя за потвърждение
func confirm(ctx *launcher.Context, prompt string) (bool, error) {
	options := []string{"Yes", "No"}
	choice, err := ctx.Show(options, prompt)
	if err != nil {
		return false, err
	}
	return choice == "Yes", nil
}

// executeCommand изпълнява shell команда
func executeCommand(cmdStr string) error {
	if cmdStr == "" {
		return fmt.Errorf("no command specified")
	}

	// Expand ~ и environment variables
	cmdStr = expandPath(cmdStr)

	cmd := exec.Command("sh", "-c", cmdStr)
	return cmd.Run()
}

// expandPath разширява ~ и $HOME в команда
func expandPath(cmdStr string) string {
	home := os.Getenv("HOME")
	if home == "" {
		home = "/home/" + os.Getenv("USER")
	}

	// Replace $HOME with actual home path
	cmdStr = strings.ReplaceAll(cmdStr, "$HOME", home)

	// Replace ~ at the beginning
	if strings.HasPrefix(cmdStr, "~/") {
		cmdStr = strings.Replace(cmdStr, "~", home, 1)
	}

	// Replace ~ after space (for commands with multiple paths)
	cmdStr = strings.ReplaceAll(cmdStr, " ~/", " "+home+"/")

	// Replace ~ after && or ||
	cmdStr = strings.ReplaceAll(cmdStr, "&& ~/", "&& "+home+"/")
	cmdStr = strings.ReplaceAll(cmdStr, "|| ~/", "|| "+home+"/")

	return cmdStr
}

// Package mail fronts the existing mail machinery: the user's own status
// script for the unread count, mbsync for fetching, neomutt for reading.
// ql adds the menu, not another mail implementation.
package mail

import (
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
		Name:        "mail",
		Description: "Mail (status, sync, read)",
		Run:         Run,
	})
}

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfg := commands.DecodeConfig(ctx, "mail", DefaultConfig())

	if !cfg.Enabled {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("mail module is disabled in config"),
		}
	}

	notifCfg := ctx.Config().GetNotificationConfig()

	if args := ctx.Args(); len(args) > 0 {
		if err := action(strings.ToLower(args[0]), &cfg, &notifCfg); err != nil {
			utils.ShowErrorNotificationWithConfig(&notifCfg, "Mail Error", err.Error())
			return commands.CommandResult{Success: false, Error: err}
		}
		return commands.CommandResult{Success: true}
	}

	options := []string{}
	if !ctx.IsDirectLaunch() {
		options = append(options, "← Back")
	}
	options = append(options, "Status", "Sync now", "Open client")

	choice, err := ctx.Show(options, "Mail")
	if err != nil {
		return commands.CommandResult{Success: false}
	}
	if choice == "← Back" {
		return commands.CommandResult{Success: false, Error: commands.ErrBack}
	}

	verb := map[string]string{"Status": "status", "Sync now": "sync", "Open client": "open"}[choice]
	if err := action(verb, &cfg, &notifCfg); err != nil {
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Mail Error", err.Error())
		return commands.CommandResult{Success: false, Error: err}
	}
	return commands.CommandResult{Success: true}
}

func action(verb string, cfg *Config, notifCfg *config.NotificationConfig) error {
	switch verb {
	case "status":
		out, err := exec.Command("sh", "-c", cfg.StatusCmd).CombinedOutput()
		text := strings.TrimSpace(string(out))
		if err != nil {
			if text == "" {
				text = err.Error()
			}
			return fmt.Errorf("status: %s", text)
		}
		if text == "" {
			text = "no output"
		}
		utils.NotifyWithConfig(notifCfg, "Mail", text)
		return nil

	case "sync":
		// In a terminal, visibly: a full mbsync run takes long enough that a
		// silent background job reads as "nothing happened".
		return inTerminal(cfg.SyncCmd + `; echo; echo "Done — Enter closes."; read x`)

	case "open":
		return inTerminal(cfg.ClientCmd)

	default:
		return fmt.Errorf("unknown mail action: %s (use: status, sync, open)", verb)
	}
}

func inTerminal(command string) error {
	terminal := utils.DetectTerminal()
	if terminal == "" {
		return fmt.Errorf("no terminal emulator found")
	}
	args := append(utils.TerminalArgs(terminal), "-e", "sh", "-c", command)
	cmd := exec.Command(terminal, args...)
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open terminal: %w", err)
	}
	go cmd.Wait()
	return nil
}

// Package updates shows and installs pending system updates. The commands
// are configuration — zypper here, but nothing in the module knows that.
package updates

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
		Name:        "updates",
		Description: "System updates",
		Run:         Run,
	})
}

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfg := commands.DecodeConfig(ctx, "updates", DefaultConfig())

	if !cfg.Enabled {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("updates module is disabled in config"),
		}
	}

	notifCfg := ctx.Config().GetNotificationConfig()

	if args := ctx.Args(); len(args) > 0 {
		if err := action(strings.ToLower(args[0]), &cfg, &notifCfg); err != nil {
			utils.ShowErrorNotificationWithConfig(&notifCfg, "Updates Error", err.Error())
			return commands.CommandResult{Success: false, Error: err}
		}
		return commands.CommandResult{Success: true}
	}

	options := []string{}
	if !ctx.IsDirectLaunch() {
		options = append(options, "← Back")
	}
	options = append(options, "Show pending", "Install updates")

	choice, err := ctx.Show(options, "Updates")
	if err != nil {
		return commands.CommandResult{Success: false}
	}
	if choice == "← Back" {
		return commands.CommandResult{Success: false, Error: commands.ErrBack}
	}

	verb := map[string]string{"Show pending": "show", "Install updates": "install"}[choice]
	if err := action(verb, &cfg, &notifCfg); err != nil {
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Updates Error", err.Error())
		return commands.CommandResult{Success: false, Error: err}
	}
	return commands.CommandResult{Success: true}
}

func action(verb string, cfg *Config, notifCfg *config.NotificationConfig) error {
	switch verb {
	case "show":
		out, err := exec.Command("sh", "-c", cfg.ListCmd).CombinedOutput()
		text := strings.TrimSpace(string(out))
		if err != nil {
			if text == "" {
				text = err.Error()
			}
			return fmt.Errorf("list: %s", text)
		}
		if text == "" {
			utils.NotifyWithConfig(notifCfg, "Updates", "Nothing pending")
			return nil
		}
		return utils.DisplayTextGUI("Pending updates", text)

	case "install":
		// Interactive and visible: the update command asks for a password
		// and for confirmation, so it gets a real terminal.
		terminal := utils.DetectTerminal()
		if terminal == "" {
			return fmt.Errorf("no terminal emulator found")
		}
		cmd := exec.Command(terminal, "-e", "sh", "-c",
			cfg.UpdateCmd+`; echo; echo "Done — Enter closes."; read x`)
		cmd.Env = os.Environ()
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to open terminal: %w", err)
		}
		go cmd.Wait()
		return nil

	default:
		return fmt.Errorf("unknown updates action: %s (use: show, install)", verb)
	}
}

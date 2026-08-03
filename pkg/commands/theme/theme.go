// Package theme is the bridge to themer: pick a theme from its list, let it
// switch the whole desktop. ql adds nothing on top on purpose — themer owns
// the appliers, the state file and the reloads.
package theme

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/utils"
)

func init() {
	commands.Register(commands.Command{
		Name:        "theme",
		Description: "Switch desktop theme",
		Run:         Run,
	})
}

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfg := commands.DecodeConfig(ctx, "theme", DefaultConfig())

	if !cfg.Enabled {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("theme module is disabled in config"),
		}
	}

	notifCfg := ctx.Config().GetNotificationConfig()

	if !utils.CommandExists(cfg.Bin) {
		err := fmt.Errorf("%s is not installed", cfg.Bin)
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Theme Error", err.Error())
		return commands.CommandResult{Success: false, Error: err}
	}

	if args := ctx.Args(); len(args) > 0 {
		return apply(args[0], &cfg, &notifCfg)
	}

	names, current, err := listThemes(&cfg)
	if err != nil {
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Theme Error", err.Error())
		return commands.CommandResult{Success: false, Error: err}
	}

	var options []string
	if !ctx.IsDirectLaunch() {
		options = append(options, "← Back")
	}
	for _, name := range names {
		marker := "  "
		if name == current {
			marker = "● "
		}
		options = append(options, marker+name)
	}

	choice, err := ctx.Show(options, "Theme")
	if err != nil {
		return commands.CommandResult{Success: false}
	}
	if choice == "← Back" {
		return commands.CommandResult{Success: false, Error: commands.ErrBack}
	}

	return apply(strings.TrimSpace(strings.TrimPrefix(choice, "●")), &cfg, &notifCfg)
}

func apply(name string, cfg *Config, notifCfg *config.NotificationConfig) commands.CommandResult {
	out, err := exec.Command(cfg.Bin, name).CombinedOutput()
	if err != nil {
		msg := lastLine(string(out))
		if msg == "" {
			msg = err.Error()
		}
		utils.ShowErrorNotificationWithConfig(notifCfg, "Theme Error", msg)
		return commands.CommandResult{Success: false, Error: fmt.Errorf("themer: %s", msg)}
	}
	utils.NotifyWithConfig(notifCfg, "Theme", name)
	return commands.CommandResult{Success: true}
}

// listThemes parses `themer --list`: two-space indent for a theme, "● " in
// front of the active one.
func listThemes(cfg *Config) ([]string, string, error) {
	out, err := exec.Command(cfg.Bin, "--list").Output()
	if err != nil {
		return nil, "", fmt.Errorf("%s --list: %w", cfg.Bin, err)
	}
	var names []string
	current := ""
	for _, line := range strings.Split(string(out), "\n") {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		if marked, ok := strings.CutPrefix(name, "● "); ok {
			name = marked
			current = marked
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return nil, "", fmt.Errorf("themer reported no themes")
	}
	return names, current, nil
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

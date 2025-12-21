// Package man provides manpage search and viewing functionality for ql.
// It allows searching and viewing manpages.
package man

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
		Name:        "man",
		Description: "Manual pages",
		Run:         Run,
	})
}

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfgInterface := ctx.Config().GetManConfig()

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
			Error:   fmt.Errorf("man module is disabled in config"),
		}
	}

	if !utils.CommandExists("man") {
		notifCfg := ctx.Config().GetNotificationConfig()
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Man Error",
			"man command not found")
		return commands.CommandResult{Success: false}
	}

	notifCfg := ctx.Config().GetNotificationConfig()

	// Check for direct command (man page name)
	args := ctx.Args()
	if len(args) > 0 {
		if err := openManpage(args[0], &cfg, ctx.Config()); err != nil {
			return commands.CommandResult{Success: false, Error: err}
		}
		return commands.CommandResult{Success: true}
	}

	manpages, err := getAllManpages(&cfg)
	if err != nil {
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Man Error", err.Error())
		return commands.CommandResult{Success: false}
	}

	if len(manpages) == 0 {
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Man Error", "No manpages found")
		return commands.CommandResult{Success: false}
	}

	var options []string

	if !ctx.IsDirectLaunch() {
		options = append(options, "← Back")
	}

	options = append(options, manpages...)

	selected, err := ctx.Show(options, "Manual Pages")
	if err != nil {
		// ESC pressed - exit completely
		return commands.CommandResult{Success: false}
	}

	if selected == "← Back" {
		return commands.CommandResult{
			Success: false,
			Error:   commands.ErrBack,
		}
	}

	if selected == "" {
		return commands.CommandResult{Success: false}
	}

	if err := openManpage(selected, &cfg, ctx.Config()); err != nil {
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Man Error", err.Error())
		return commands.CommandResult{Success: false}
	}

	return commands.CommandResult{Success: true}
}

func getAllManpages(cfg *Config) ([]string, error) {
	cmd := exec.Command("man", "-k", ".")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get manpages:     %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var manpages []string

	for _, line := range lines {
		if line == "" {
			continue
		}

		formatted := formatManpage(line)
		if formatted != "" {
			manpages = append(manpages, formatted)
		}

		if cfg.MaxResults > 0 && len(manpages) >= cfg.MaxResults {
			break
		}
	}

	return manpages, nil
}

func formatManpage(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}

	parts := strings.Fields(line)
	if len(parts) < 2 {
		return line
	}

	name := parts[0]
	section := parts[1]
	section = strings.Trim(section, "()")

	if len(parts) > 2 {
		description := strings.Join(parts[3:], " ")
		return fmt.Sprintf("%s (%s) - %s", name, section, description)
	}

	return fmt.Sprintf("%s (%s)", name, section)
}

func openManpage(entry string, cfg *Config, globalCfg *config.Config) error {
	parts := strings.Fields(entry)
	if len(parts) == 0 {
		return fmt.Errorf("invalid manpage entry")
	}

	manName := parts[0]

	// Get man viewer from global config
	pager := globalCfg.GetManViewer()

	// Check if pager exists, fallback to less
	if !utils.CommandExists(pager) {
		pager = "less"
	}

	// Get terminal
	terminal := cfg.Terminal
	if terminal == "" {
		terminal = utils.DetectTerminal()
	}
	if terminal == "" {
		terminal = "xterm"
	}

	if !utils.CommandExists(terminal) {
		return fmt.Errorf("terminal not found:  %s", terminal)
	}

	// Build pager command with -p flag for nvimpager (force pager mode)
	pagerCmd := pager
	if strings.Contains(pager, "nvimpager") {
		pagerCmd = pager + " -p"
	}

	// Use man piped to pager (no read at the end)
	script := fmt.Sprintf("man %s | %s", manName, pagerCmd)
	cmd := exec.Command(terminal, "-e", "sh", "-c", script)
	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open manpage: %w", err)
	}

	go cmd.Wait()

	return nil
}

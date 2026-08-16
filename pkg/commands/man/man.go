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
)

func init() {
	commands.Register(commands.Command{
		Name:        "man",
		Description: "Manual pages",
		Run:         Run,
	})
}

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfg := commands.DecodeConfig(ctx, "man", DefaultConfig())

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
		return nil, fmt.Errorf("failed to get manpages: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var manpages []string

	for _, line := range lines {
		if line == "" {
			continue
		}

		formatted := formatManpage(line, cfg.ShowDescriptions)
		if formatted != "" {
			manpages = append(manpages, formatted)
		}

		if cfg.MaxResults > 0 && len(manpages) >= cfg.MaxResults {
			break
		}
	}

	return manpages, nil
}

// formatManpage renders one `man -k` line. withDescription reflects the
// show_descriptions setting, which used to be decoded and then ignored — the
// description was always appended regardless of what the user configured.
func formatManpage(line string, withDescription bool) string {
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

	if withDescription && len(parts) > 2 {
		description := strings.Join(parts[3:], " ")
		return fmt.Sprintf("%s (%s) - %s", name, section, description)
	}

	return fmt.Sprintf("%s (%s)", name, section)
}

// manScript builds the `sh -c` line that pipes a page into the pager.
//
// The page name comes from `man -k` output — third-party packages get a say in
// it — so it is shell-quoted, not interpolated raw. It has to be SHELL quoting:
// %q produces a double-quoted Go string, and the shell still expands $ and
// backticks inside double quotes, so a page whose NAME field carried $(...)
// would have been executed. The pager stays unquoted on purpose: it is the
// user's own config and may legitimately carry flags ("nvimpager -p").
func manScript(manName, pager string) string {
	// nvimpager needs -p to be forced into pager mode.
	pagerCmd := pager
	if strings.Contains(pager, "nvimpager") {
		pagerCmd = pager + " -p"
	}

	return fmt.Sprintf("man %s | %s", utils.ShellQuote(manName), pagerCmd)
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
	// A configured terminal is resolved the same way as a detected one: by name it would be
	// looked up in PATH alone, which is exactly what a compositor keybinding does not have.
	terminal := cfg.Terminal
	if terminal != "" {
		terminal = utils.Look(terminal)
	}
	if terminal == "" {
		terminal = utils.DetectTerminal()
	}
	if terminal == "" {
		return fmt.Errorf("no terminal emulator found")
	}

	script := manScript(manName, pager)
	cmd := exec.Command(terminal, append(utils.TerminalArgs(terminal), "-e", "sh", "-c", script)...)
	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open manpage: %w", err)
	}

	go cmd.Wait()

	return nil
}

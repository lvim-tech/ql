// Package clipboard provides clipboard history management for ql.
// It supports cliphist, clipman, and clipmenu backends with X11/Wayland detection.
package clipboard

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/utils"
	"github.com/mitchellh/mapstructure"
)

func init() {
	commands.Register(commands.Command{
		Name:        "clipboard",
		Description: "Clipboard manager",
		Run:         Run,
	})
}

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfgInterface := ctx.Config().GetClipboardConfig()

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
			Error:   fmt.Errorf("clipboard module is disabled in config"),
		}
	}

	backend := detectBackend()
	if backend == "" {
		notifCfg := ctx.Config().GetNotificationConfig()
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Clipboard Error",
			"No clipboard backend found. Install cliphist, clipman, or clipmenu")
		return commands.CommandResult{Success: false}
	}

	notifCfg := ctx.Config().GetNotificationConfig()

	// Check for direct command
	args := ctx.Args()
	if len(args) > 0 {
		return executeDirectCommand(ctx, args[0], backend, &cfg, &notifCfg)
	}

	for {
		var options []string

		if !ctx.IsDirectLaunch() {
			options = append(options, "← Back")
		}

		options = append(options,
			"Show History",
			"Clear History",
		)

		choice, err := ctx.Show(options, "Clipboard Manager")
		if err != nil {
			// ESC pressed at main menu - exit completely
			return commands.CommandResult{Success: false}
		}

		if choice == "← Back" {
			return commands.CommandResult{
				Success: false,
				Error:   commands.ErrBack,
			}
		}

		switch choice {
		case "Show History":
			result := showHistory(ctx, backend, &cfg)
			if result.Success {
				return result
			}
			// If error is NOT ErrBack - it's ESC, exit completely
			if result.Error != nil && result.Error != commands.ErrBack {
				return commands.CommandResult{Success: false}
			}
			// If ErrBack - continue loop
			if result.Error == commands.ErrBack {
				continue
			}
			// If error is nil - also exit
			return commands.CommandResult{Success: false}

		case "Clear History":
			result := clearHistory(ctx, backend, &notifCfg)
			// If error is NOT ErrBack - it's ESC, exit completely
			if result.Error != nil && result.Error != commands.ErrBack {
				return commands.CommandResult{Success: false}
			}
			// If ErrBack - continue loop
			if result.Error == commands.ErrBack {
				continue
			}
			// If nil - exit
			return commands.CommandResult{Success: false}
		}
	}
}

func executeDirectCommand(ctx commands.LauncherContext, action string, backend string, cfg *Config, notifCfg *config.NotificationConfig) commands.CommandResult {
	switch strings.ToLower(action) {
	case "show", "history":
		return showHistory(ctx, backend, cfg)
	case "clear":
		return clearHistoryDirect(backend, notifCfg)
	default:
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("unknown clipboard action: %s (use 'show' or 'clear')", action),
		}
	}
}

func clearHistoryDirect(backend string, notifCfg *config.NotificationConfig) commands.CommandResult {
	var cmd *exec.Cmd
	switch backend {
	case "cliphist":
		cmd = exec.Command("cliphist", "wipe")
	case "clipman":
		cmd = exec.Command("clipman", "clear", "--all")
	case "clipmenu":
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("clear not supported for clipmenu"),
		}
	default:
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("unsupported backend:  %s", backend),
		}
	}

	if err := cmd.Run(); err != nil {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("failed to clear clipboard:  %w", err),
		}
	}

	utils.NotifyWithConfig(notifCfg, "Clipboard", "History cleared")
	return commands.CommandResult{Success: true}
}

func detectBackend() string {
	if utils.CommandExists("cliphist") {
		return "cliphist"
	}
	if utils.CommandExists("clipman") {
		return "clipman"
	}
	if utils.CommandExists("clipmenu") {
		return "clipmenu"
	}
	return ""
}

func showHistory(ctx commands.LauncherContext, backend string, cfg *Config) commands.CommandResult {
	historyLines, err := getHistory(backend, cfg.MaxItems)
	if err != nil {
		return commands.CommandResult{Success: false, Error: err}
	}

	var options []string

	if !ctx.IsDirectLaunch() {
		options = append(options, "← Back")
	}

	if len(historyLines) == 0 {
		options = append(options, "Clipboard history is empty")
	} else {
		options = append(options, historyLines...)
	}

	selected, err := ctx.Show(options, "Clipboard History")
	if err != nil {
		// ESC pressed - return error that's NOT ErrBack
		return commands.CommandResult{Success: false, Error: fmt.Errorf("ESC")}
	}

	if selected == "← Back" {
		return commands.CommandResult{
			Success: false,
			Error:   commands.ErrBack,
		}
	}

	if selected == "Clipboard history is empty" || selected == "" {
		return commands.CommandResult{Success: false, Error: commands.ErrBack}
	}

	if err := copyToClipboard(selected); err != nil {
		return commands.CommandResult{Success: false, Error: err}
	}

	notifCfg := ctx.Config().GetNotificationConfig()
	utils.NotifyWithConfig(&notifCfg, "Clipboard", "Copied to clipboard")

	return commands.CommandResult{Success: true}
}

func getHistory(backend string, maxItems int) ([]string, error) {
	var cmd *exec.Cmd

	switch backend {
	case "cliphist":
		cmd = exec.Command("cliphist", "list")
	case "clipman":
		cmd = exec.Command("clipman", "pick", "--print-query")
	case "clipmenu":
		return getClipmenuHistory()
	default:
		return nil, fmt.Errorf("unsupported backend: %s", backend)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get clipboard history: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	var filtered []string
	for _, line := range lines {
		if line == "" {
			continue
		}

		displayLine := line
		if backend == "cliphist" {
			if _, content, found := strings.Cut(line, "\t"); found {
				displayLine = content
			}
		}

		if len(displayLine) > 100 {
			displayLine = displayLine[:97] + "..."
		}

		filtered = append(filtered, displayLine)
	}

	if maxItems > 0 && len(filtered) > maxItems {
		filtered = filtered[:maxItems]
	}

	return filtered, nil
}

func getClipmenuHistory() ([]string, error) {
	return []string{"clipmenu:   Use 'clipmenu' directly"}, nil
}

func clearHistory(ctx commands.LauncherContext, backend string, notifCfg *config.NotificationConfig) commands.CommandResult {
	options := []string{"← Back", "Yes", "No"}
	choice, err := ctx.Show(options, "Clear clipboard history? ")
	if err != nil {
		// ESC pressed - return error that's NOT ErrBack
		return commands.CommandResult{Success: false, Error: fmt.Errorf("ESC")}
	}

	if choice == "← Back" || choice == "No" {
		return commands.CommandResult{Success: false, Error: commands.ErrBack}
	}

	if choice != "Yes" {
		return commands.CommandResult{Success: false, Error: commands.ErrBack}
	}

	var cmd *exec.Cmd
	switch backend {
	case "cliphist":
		cmd = exec.Command("cliphist", "wipe")
	case "clipman":
		cmd = exec.Command("clipman", "clear", "--all")
	case "clipmenu":
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("clear not supported for clipmenu"),
		}
	default:
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("unsupported backend: %s", backend),
		}
	}

	if err := cmd.Run(); err != nil {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("failed to clear clipboard:  %w", err),
		}
	}

	utils.NotifyWithConfig(notifCfg, "Clipboard", "History cleared")
	return commands.CommandResult{Success: false, Error: commands.ErrBack}
}

func copyToClipboard(content string) error {
	server := utils.DetectDisplayServer()

	var cmd *exec.Cmd
	if server.IsWayland() {
		if !utils.CommandExists("wl-copy") {
			return fmt.Errorf("wl-copy not found (install wl-clipboard)")
		}
		cmd = exec.Command("wl-copy")
	} else {
		if utils.CommandExists("xclip") {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if utils.CommandExists("xsel") {
			cmd = exec.Command("xsel", "-b")
		} else {
			return fmt.Errorf("no clipboard tool found (install xclip or xsel)")
		}
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	_, err = stdin.Write([]byte(content))
	if err != nil {
		return err
	}
	stdin.Close()

	return cmd.Wait()
}

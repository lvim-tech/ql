// Package screenshot provides screenshot functionality for ql.
// It supports both X11 (scrot/maim) and Wayland (grim/slurp) with multiple capture modes.
package screenshot

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/utils"
	"github.com/mitchellh/mapstructure"
)

func init() {
	commands.Register(commands.Command{
		Name:        "screenshot",
		Description: "Take screenshot",
		Run:         Run,
	})
}

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfgInterface := ctx.Config().GetScreenshotConfig()

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
			Error:   fmt.Errorf("screenshot module is disabled in config"),
		}
	}

	saveDir := utils.ExpandHomeDir(cfg.SaveDir)
	if err := utils.EnsureDir(saveDir); err != nil {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("failed to create save directory: %w", err),
		}
	}

	notifCfg := ctx.Config().GetNotificationConfig()

	// Check for direct command
	args := ctx.Args()
	if len(args) > 0 {
		return executeDirectCommand(args, &cfg, &notifCfg)
	}

	for {
		var options []string

		if !ctx.IsDirectLaunch() {
			options = append(options, "← Back")
		}

		options = append(options,
			"Fullscreen",
			"Active Window",
			"Select Region",
		)

		choice, err := ctx.Show(options, "Screenshot")
		if err != nil {
			// ESC pressed - exit completely
			return commands.CommandResult{Success: false}
		}

		if choice == "← Back" {
			return commands.CommandResult{
				Success: false,
				Error:   commands.ErrBack,
			}
		}

		timestamp := utils.GetTimestamp()
		filename := fmt.Sprintf("%s_%s.png", cfg.FilePrefix, timestamp)
		outputPath := filepath.Join(saveDir, filename)

		server := utils.DetectDisplayServer()

		var cmd *exec.Cmd
		if server.IsWayland() {
			cmd, err = buildWaylandCommand(choice, outputPath)
		} else {
			cmd, err = buildX11Command(choice, outputPath)
		}

		if err != nil {
			// Error building command - show notification and loop back
			utils.ShowErrorNotificationWithConfig(&notifCfg, "Screenshot Error", err.Error())
			continue
		}

		if err := cmd.Run(); err != nil {
			// Screenshot failed - show notification and loop back
			utils.ShowErrorNotificationWithConfig(&notifCfg, "Screenshot Error", fmt.Sprintf("Screenshot failed: %v", err))
			continue
		}

		// Screenshot succeeded - show notification and exit
		utils.NotifyWithConfig(&notifCfg, "Screenshot saved", filename)

		return commands.CommandResult{Success: true}
	}
}

func executeDirectCommand(args []string, cfg *Config, notifCfg *config.NotificationConfig) commands.CommandResult {
	mode := strings.ToLower(args[0])

	var screenshotMode string

	switch mode {
	case "full", "fullscreen":
		screenshotMode = "Fullscreen"

	case "window", "active":
		screenshotMode = "Active Window"

	case "region", "area", "select":
		screenshotMode = "Select Region"

	default:
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("unknown screenshot mode: %s (use:  full, window, region)", mode),
		}
	}

	saveDir := utils.ExpandHomeDir(cfg.SaveDir)
	if err := utils.EnsureDir(saveDir); err != nil {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("failed to create save directory:  %w", err),
		}
	}

	timestamp := utils.GetTimestamp()
	filename := fmt.Sprintf("%s_%s.png", cfg.FilePrefix, timestamp)
	outputPath := filepath.Join(saveDir, filename)

	server := utils.DetectDisplayServer()

	var cmd *exec.Cmd
	var err error

	if server.IsWayland() {
		cmd, err = buildWaylandCommand(screenshotMode, outputPath)
	} else {
		cmd, err = buildX11Command(screenshotMode, outputPath)
	}

	if err != nil {
		return commands.CommandResult{Success: false, Error: err}
	}

	if err := cmd.Run(); err != nil {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("screenshot failed: %w", err),
		}
	}

	utils.NotifyWithConfig(notifCfg, "Screenshot saved", filename)

	return commands.CommandResult{Success: true}
}

func buildWaylandCommand(mode, outputPath string) (*exec.Cmd, error) {
	compositor := detectCompositor()

	if compositor == "gnome" {
		return buildGNOMECommand(mode, outputPath)
	}
	if compositor == "kde" {
		return buildKDECommand(mode, outputPath)
	}

	if !utils.CommandExists("grim") {
		return nil, fmt.Errorf("grim is not installed (required for Wayland)")
	}

	switch mode {
	case "Fullscreen":
		return exec.Command("grim", outputPath), nil

	case "Active Window":
		return exec.Command("sh", "-c",
			fmt.Sprintf("grim -g \"$(swaymsg -t get_tree | jq -r '..  | select(.focused?) | .rect | \"\\(.x),\\(.y) \\(.width)x\\(.height)\"')\" %s", outputPath)), nil

	case "Select Region":
		if !utils.CommandExists("slurp") {
			return nil, fmt.Errorf("slurp is not installed (required for region selection)")
		}
		return exec.Command("sh", "-c",
			fmt.Sprintf("grim -g \"$(slurp)\" %s", outputPath)), nil

	default:
		return nil, fmt.Errorf("unknown mode: %s", mode)
	}
}

func buildX11Command(mode, outputPath string) (*exec.Cmd, error) {
	if utils.CommandExists("maim") {
		switch mode {
		case "Fullscreen":
			return exec.Command("maim", outputPath), nil
		case "Active Window":
			return exec.Command("maim", "-i", "$(xdotool getactivewindow)", outputPath), nil
		case "Select Region":
			return exec.Command("maim", "-s", outputPath), nil
		default:
			return nil, fmt.Errorf("unknown mode: %s", mode)
		}
	}

	if utils.CommandExists("scrot") {
		switch mode {
		case "Fullscreen":
			return exec.Command("scrot", outputPath), nil
		case "Active Window":
			return exec.Command("scrot", "-u", outputPath), nil
		case "Select Region":
			return exec.Command("scrot", "-s", outputPath), nil
		default:
			return nil, fmt.Errorf("unknown mode: %s", mode)
		}
	}

	return nil, fmt.Errorf("no screenshot tool found (install maim or scrot)")
}

func buildGNOMECommand(mode, outputPath string) (*exec.Cmd, error) {
	if !utils.CommandExists("gnome-screenshot") {
		return nil, fmt.Errorf("gnome-screenshot is not installed")
	}

	switch mode {
	case "Fullscreen":
		return exec.Command("gnome-screenshot", "-f", outputPath), nil
	case "Active Window":
		return exec.Command("gnome-screenshot", "-w", "-f", outputPath), nil
	case "Select Region":
		return exec.Command("gnome-screenshot", "-a", "-f", outputPath), nil
	default:
		return nil, fmt.Errorf("unknown mode: %s", mode)
	}
}

func buildKDECommand(mode, outputPath string) (*exec.Cmd, error) {
	if !utils.CommandExists("spectacle") {
		return nil, fmt.Errorf("spectacle is not installed")
	}

	switch mode {
	case "Fullscreen":
		return exec.Command("spectacle", "-b", "-n", "-o", outputPath), nil
	case "Active Window":
		return exec.Command("spectacle", "-a", "-b", "-n", "-o", outputPath), nil
	case "Select Region":
		return exec.Command("spectacle", "-r", "-b", "-n", "-o", outputPath), nil
	default:
		return nil, fmt.Errorf("unknown mode: %s", mode)
	}
}

func detectCompositor() string {
	desktop := utils.GetCurrentDesktop()
	if desktop == "GNOME" {
		return "gnome"
	}
	if desktop == "KDE" {
		return "kde"
	}
	return ""
}

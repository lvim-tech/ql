// Package screenshot provides screenshot functionality for ql.
// It supports both X11 (scrot/maim) and Wayland (grim/slurp) with multiple capture modes.
package screenshot

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/lvim-tech/ql/pkg/commands"
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

	saveDir := cfg.SaveDir
	if len(saveDir) >= 2 && saveDir[:2] == "~/" {
		saveDir = filepath.Join(os.Getenv("HOME"), saveDir[2:])
	}

	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("failed to create save directory: %w", err),
		}
	}

	for {
		options := []string{
			"← Back",
			"Fullscreen",
			"Active Window",
			"Select Region",
		}

		choice, err := ctx.Show(options, "Screenshot")
		if err != nil {
			return commands.CommandResult{Success: false}
		}

		if choice == "← Back" {
			return commands.CommandResult{Success: false}
		}

		timestamp := time.Now().Format("2006-01-02_15-04-05")
		filename := fmt.Sprintf("%s_%s.png", cfg.FilePrefix, timestamp)
		outputPath := filepath.Join(saveDir, filename)

		isWayland := os.Getenv("WAYLAND_DISPLAY") != ""

		var cmd *exec.Cmd
		if isWayland {
			cmd, err = buildWaylandCommand(choice, outputPath)
		} else {
			cmd, err = buildX11Command(choice, outputPath)
		}

		if err != nil {
			showErrorNotification("Screenshot Error", err.Error())
			continue
		}

		if err := cmd.Run(); err != nil {
			showErrorNotification("Screenshot Error", fmt.Sprintf("Screenshot failed: %v", err))
			continue
		}

		notify("Screenshot saved", filename)

		return commands.CommandResult{Success: true}
	}
}

func buildWaylandCommand(mode, outputPath string) (*exec.Cmd, error) {
	compositor := detectCompositor()

	if compositor == "gnome" {
		return buildGNOMECommand(mode, outputPath)
	}
	if compositor == "kde" {
		return buildKDECommand(mode, outputPath)
	}

	if _, err := exec.LookPath("grim"); err != nil {
		return nil, fmt.Errorf("grim is not installed (required for Wayland)")
	}

	switch mode {
	case "Fullscreen":
		return exec.Command("grim", outputPath), nil

	case "Active Window":
		switch compositor {
		case "niri":
			return exec.Command("niri", "msg", "action", "screenshot-window", "--path", outputPath), nil

		case "sway":
			cmd := exec.Command("sh", "-c",
				fmt.Sprintf("swaymsg -t get_tree | jq -r '..  | select(.focused?  == true) | .rect | \"\\(.x),\\(.y) \\(.width)x\\(.height)\"' | grim -g - '%s'", outputPath))
			return cmd, nil

		case "hyprland":
			cmd := exec.Command("sh", "-c",
				fmt.Sprintf("hyprctl -j activewindow | jq -r '. at,. size | @tsv' | awk '{print $1\",\"$2\" \"$3\"x\"$4}' | grim -g - '%s'", outputPath))
			return cmd, nil

		default:
			if _, err := exec.LookPath("slurp"); err != nil {
				return nil, fmt.Errorf("slurp is not installed and compositor not detected")
			}
			cmd := exec.Command("sh", "-c", fmt.Sprintf("slurp | grim -g - '%s'", outputPath))
			return cmd, nil
		}

	case "Select Region":
		if _, err := exec.LookPath("slurp"); err != nil {
			return nil, fmt.Errorf("slurp is not installed (required for region selection)")
		}
		cmd := exec.Command("sh", "-c", fmt.Sprintf("slurp | grim -g - '%s'", outputPath))
		return cmd, nil

	default:
		return nil, fmt.Errorf("unknown mode: %s", mode)
	}
}

func buildX11Command(mode, outputPath string) (*exec.Cmd, error) {
	de := detectDE()

	if de == "gnome" {
		return buildGNOMECommand(mode, outputPath)
	}
	if de == "kde" {
		return buildKDECommand(mode, outputPath)
	}

	switch mode {
	case "Fullscreen":
		if _, err := exec.LookPath("maim"); err == nil {
			return exec.Command("maim", outputPath), nil
		}
		if _, err := exec.LookPath("scrot"); err == nil {
			return exec.Command("scrot", outputPath), nil
		}
		return nil, fmt.Errorf("neither maim nor scrot is installed (required for X11)")

	case "Active Window":
		wm := detectX11WM()

		switch wm {
		case "i3":
			if _, err := exec.LookPath("maim"); err == nil {
				cmd := exec.Command("sh", "-c",
					fmt.Sprintf("i3-msg -t get_tree | jq -r '..  | select(.focused? == true and .window?) | .rect | \"\\(.x),\\(.y) \\(.width)x\\(.height)\"' | maim -g - '%s'", outputPath))
				return cmd, nil
			}
			if _, err := exec.LookPath("scrot"); err == nil {
				return exec.Command("scrot", "-u", outputPath), nil
			}

		case "xmonad", "qtile":
			if _, err := exec.LookPath("maim"); err == nil {
				cmd := exec.Command("sh", "-c",
					fmt.Sprintf("maim -i $(xdotool getactivewindow) '%s'", outputPath))
				return cmd, nil
			}
			if _, err := exec.LookPath("scrot"); err == nil {
				return exec.Command("scrot", "-u", outputPath), nil
			}

		default:
			if _, err := exec.LookPath("maim"); err == nil {
				if _, err := exec.LookPath("xdotool"); err == nil {
					cmd := exec.Command("sh", "-c",
						fmt.Sprintf("maim -i $(xdotool getactivewindow) '%s'", outputPath))
					return cmd, nil
				}
			}
			if _, err := exec.LookPath("scrot"); err == nil {
				return exec.Command("scrot", "-u", outputPath), nil
			}
		}

		return nil, fmt.Errorf("no suitable screenshot tool found for active window")

	case "Select Region":
		if _, err := exec.LookPath("maim"); err == nil {
			return exec.Command("maim", "-s", outputPath), nil
		}
		if _, err := exec.LookPath("scrot"); err == nil {
			return exec.Command("scrot", "-s", outputPath), nil
		}
		return nil, fmt.Errorf("neither maim nor scrot is installed (required for X11)")

	default:
		return nil, fmt.Errorf("unknown mode: %s", mode)
	}
}

func buildGNOMECommand(mode, outputPath string) (*exec.Cmd, error) {
	if _, err := exec.LookPath("gnome-screenshot"); err != nil {
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
	if _, err := exec.LookPath("spectacle"); err != nil {
		return nil, fmt.Errorf("spectacle is not installed")
	}

	switch mode {
	case "Fullscreen":
		return exec.Command("spectacle", "-fb", "-o", outputPath), nil

	case "Active Window":
		return exec.Command("spectacle", "-ab", "-o", outputPath), nil

	case "Select Region":
		return exec.Command("spectacle", "-rb", "-o", outputPath), nil

	default:
		return nil, fmt.Errorf("unknown mode: %s", mode)
	}
}

func detectCompositor() string {
	sessionType := os.Getenv("XDG_SESSION_TYPE")

	if sessionType != "wayland" {
		return "unknown"
	}

	if _, err := exec.LookPath("niri"); err == nil {
		if out, err := exec.Command("niri", "msg", "version").Output(); err == nil && len(out) > 0 {
			return "niri"
		}
	}

	if os.Getenv("SWAYSOCK") != "" {
		return "sway"
	}

	if os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") != "" {
		return "hyprland"
	}

	desktop := os.Getenv("XDG_CURRENT_DESKTOP")
	if desktop == "GNOME" || os.Getenv("DESKTOP_SESSION") == "gnome" {
		return "gnome"
	}

	if desktop == "KDE" || desktop == "plasma" {
		return "kde"
	}

	return "unknown"
}

func detectDE() string {
	sessionType := os.Getenv("XDG_SESSION_TYPE")

	if sessionType != "x11" {
		return "unknown"
	}

	desktop := os.Getenv("XDG_CURRENT_DESKTOP")
	session := os.Getenv("DESKTOP_SESSION")

	if desktop == "GNOME" || session == "gnome" {
		return "gnome"
	}

	if desktop == "KDE" || desktop == "plasma" || session == "plasma" {
		return "kde"
	}

	return "unknown"
}

func detectX11WM() string {
	sessionType := os.Getenv("XDG_SESSION_TYPE")

	if sessionType != "x11" {
		return "unknown"
	}

	if _, err := exec.LookPath("i3-msg"); err == nil {
		if out, err := exec.Command("i3-msg", "-t", "get_version").Output(); err == nil && len(out) > 0 {
			return "i3"
		}
	}

	if out, err := exec.Command("pgrep", "-x", "xmonad").Output(); err == nil && len(out) > 0 {
		return "xmonad"
	}

	if out, err := exec.Command("pgrep", "-x", "qtile").Output(); err == nil && len(out) > 0 {
		return "qtile"
	}

	return "unknown"
}

func notify(title, message string) {
	if _, err := exec.LookPath("notify-send"); err == nil {
		exec.Command("notify-send", title, message).Run()
	}
}

func showErrorNotification(title, message string) {
	if _, err := exec.LookPath("dunstify"); err == nil {
		cmd := exec.Command("dunstify",
			"-u", "critical",
			"-t", "10000",
			title,
			message)
		cmd.Env = os.Environ()
		cmd.Start()
		return
	}

	if _, err := exec.LookPath("notify-send"); err == nil {
		cmd := exec.Command("notify-send",
			"-u", "critical",
			"-t", "10000",
			title,
			message)
		cmd.Env = os.Environ()
		cmd.Start()
		return
	}
}

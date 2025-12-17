// Package screenshot provides screenshot functionality for ql.
// It supports both X11 (maim) and Wayland (grim/slurp) with multiple capture modes.
package screenshot

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/launcher"
)

func init() {
	commands.Register(commands.Command{
		Name:        "screenshot",
		Description: "Take screenshots",
		Run:         Run,
	})
}

func Run(ctx *launcher.Context) error {
	cfg := config.Get()
	screenshotCfg := cfg.Commands.Screenshot

	// Провери дали е enabled
	if !screenshotCfg.Enabled {
		return fmt.Errorf("screenshot module is disabled in config")
	}

	// Detect session type
	sessionType := detectSessionType()

	// Създай опции
	options := []string{
		"Fullscreen",
		"Active Window",
		"Selected Region",
	}

	if sessionType == "wayland" {
		options = append(options, "Current Output")
	}

	// Покажи меню
	mode, err := ctx.Show(options, "Screenshot Mode")
	if err != nil {
		return err
	}

	// Избери destination
	destOptions := []string{
		"Save to File",
		"Copy to Clipboard",
		"Both",
	}

	destination, err := ctx.Show(destOptions, "Destination")
	if err != nil {
		return err
	}

	// Вземи screenshot
	return takeScreenshot(mode, destination, sessionType, &screenshotCfg)
}

func detectSessionType() string {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return "wayland"
	}
	return "x11"
}

func takeScreenshot(mode, destination, sessionType string, cfg *config.ScreenshotConfig) error {
	// Expand save dir
	saveDir := os.ExpandEnv(cfg.SaveDir)
	if strings.HasPrefix(saveDir, "~/") {
		home, _ := os.UserHomeDir()
		saveDir = filepath.Join(home, saveDir[2:])
	}

	// Създай директорията ако не съществува
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return fmt.Errorf("failed to create save directory: %w", err)
	}

	// Генерирай filename
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(saveDir, fmt.Sprintf("%s_%s.png", cfg.FilePrefix, timestamp))

	// Вземи screenshot
	var err error
	if sessionType == "wayland" {
		err = takeWaylandScreenshot(mode, destination, filename)
	} else {
		err = takeX11Screenshot(mode, destination, filename)
	}

	if err != nil {
		return err
	}

	// Notification
	notify(fmt.Sprintf("Screenshot saved: %s", filepath.Base(filename)))
	return nil
}

func takeWaylandScreenshot(mode, destination, filename string) error {
	var grimArgs []string

	switch mode {
	case "Fullscreen":
		grimArgs = []string{}
	case "Active Window":
		// Get active window geometry
		geometry, err := getWaylandActiveWindowGeometry()
		if err != nil {
			return err
		}
		grimArgs = []string{"-g", geometry}
	case "Selected Region":
		// Use slurp for region selection
		slurpCmd := exec.Command("slurp")
		output, err := slurpCmd.Output()
		if err != nil {
			return fmt.Errorf("slurp failed: %w", err)
		}
		geometry := strings.TrimSpace(string(output))
		grimArgs = []string{"-g", geometry}
	case "Current Output":
		// Get current output
		output, err := getCurrentWaylandOutput()
		if err != nil {
			return err
		}
		grimArgs = []string{"-o", output}
	}

	// Execute based on destination
	if strings.Contains(destination, "Save") || destination == "Both" {
		args := append(grimArgs, filename)
		cmd := exec.Command("grim", args...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("grim failed: %w", err)
		}
	}

	if strings.Contains(destination, "Clipboard") || destination == "Both" {
		args := append(grimArgs, "-")
		grimCmd := exec.Command("grim", args...)
		wlCopyCmd := exec.Command("wl-copy")

		pipe, err := grimCmd.StdoutPipe()
		if err != nil {
			return err
		}
		wlCopyCmd.Stdin = pipe

		if err := grimCmd.Start(); err != nil {
			return err
		}
		if err := wlCopyCmd.Start(); err != nil {
			return err
		}
		if err := grimCmd.Wait(); err != nil {
			return err
		}
		if err := wlCopyCmd.Wait(); err != nil {
			return err
		}
	}

	return nil
}

func takeX11Screenshot(mode, destination, filename string) error {
	var maimArgs []string

	switch mode {
	case "Fullscreen":
		maimArgs = []string{}
	case "Active Window":
		activeWindow, err := exec.Command("xdotool", "getactivewindow").Output()
		if err != nil {
			return fmt.Errorf("failed to get active window: %w", err)
		}
		maimArgs = []string{"-i", strings.TrimSpace(string(activeWindow))}
	case "Selected Region":
		maimArgs = []string{"-s"}
	}

	// Execute based on destination
	if strings.Contains(destination, "Save") || destination == "Both" {
		args := append(maimArgs, filename)
		cmd := exec.Command("maim", args...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("maim failed: %w", err)
		}
	}

	if strings.Contains(destination, "Clipboard") || destination == "Both" {
		maimCmd := exec.Command("maim", maimArgs...)
		xclipCmd := exec.Command("xclip", "-selection", "clipboard", "-t", "image/png")

		pipe, err := maimCmd.StdoutPipe()
		if err != nil {
			return err
		}
		xclipCmd.Stdin = pipe

		if err := maimCmd.Start(); err != nil {
			return err
		}
		if err := xclipCmd.Start(); err != nil {
			return err
		}
		if err := maimCmd.Wait(); err != nil {
			return err
		}
		if err := xclipCmd.Wait(); err != nil {
			return err
		}
	}

	return nil
}

func getWaylandActiveWindowGeometry() (string, error) {
	// Sway example
	cmd := exec.Command("swaymsg", "-t", "get_tree")
	_, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("swaymsg failed: %w", err)
	}

	// Parse JSON to find focused window
	// TODO: Implement proper JSON parsing with jq or encoding/json
	return "", fmt.Errorf("active window detection not yet implemented for wayland")
}

func getCurrentWaylandOutput() (string, error) {
	// Sway example
	cmd := exec.Command("swaymsg", "-t", "get_outputs")
	_, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("swaymsg failed: %w", err)
	}

	// Parse JSON to find focused output
	// TODO: Implement proper JSON parsing
	return "", fmt.Errorf("output detection not yet implemented")
}

func notify(message string) {
	if _, err := exec.LookPath("notify-send"); err == nil {
		exec.Command("notify-send", "ql screenshot", message).Run()
	}
}

// Package wifi provides WiFi network management functionality for ql.
// It uses nmcli (NetworkManager) to scan and connect to wireless networks,
// with optional password input and connection testing.
package wifi

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/mitchellh/mapstructure"
)

func init() {
	commands.Register(commands.Command{
		Name:        "wifi",
		Description: "WiFi manager",
		Run:         Run,
	})
}

func Run(ctx commands.LauncherContext) error {
	// Извличаме config директно
	cfgInterface := ctx.Config().GetWifiConfig()
	if cfgInterface == nil {
		return fmt.Errorf("wifi config not found")
	}

	// Decode с WeaklyTypedInput
	var cfg Config
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           &cfg,
	})
	if err != nil {
		cfg = DefaultConfig()
	} else if err := decoder.Decode(cfgInterface); err != nil {
		cfg = DefaultConfig()
	}

	if !cfg.Enabled {
		return fmt.Errorf("wifi module is disabled in config")
	}

	// Check if nmcli is installed
	if _, err := exec.LookPath("nmcli"); err != nil {
		return fmt.Errorf("nmcli is not installed (required for wifi management)")
	}

	// Menu options
	options := []string{
		"Connect to Network",
		"Disconnect",
		"Show Current Connection",
		"Toggle WiFi",
	}

	choice, err := ctx.Show(options, "WiFi")
	if err != nil {
		return err
	}

	switch choice {
	case "Connect to Network":
		return connectToNetwork(ctx, &cfg)
	case "Disconnect":
		return disconnect(&cfg)
	case "Show Current Connection":
		return showCurrentConnection(&cfg)
	case "Toggle WiFi":
		return toggleWifi(&cfg)
	default:
		return fmt.Errorf("unknown choice: %s", choice)
	}
}

func connectToNetwork(ctx commands.LauncherContext, cfg *Config) error {
	// Scan for networks
	cmd := exec.Command("nmcli", "-t", "-f", "SSID", "dev", "wifi", "list")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to scan networks: %w", err)
	}

	// Parse networks
	lines := strings.Split(string(output), "\n")
	var networks []string
	seen := make(map[string]bool)

	for _, line := range lines {
		ssid := strings.TrimSpace(line)
		if ssid != "" && !seen[ssid] {
			networks = append(networks, ssid)
			seen[ssid] = true
		}
	}

	if len(networks) == 0 {
		return fmt.Errorf("no networks found")
	}

	// Show network menu
	choice, err := ctx.Show(networks, "Select Network")
	if err != nil {
		return err
	}

	// Connect (will prompt for password if needed)
	cmd = exec.Command("nmcli", "dev", "wifi", "connect", choice)
	if err := cmd.Run(); err != nil {
		if cfg.ShowNotify {
			notify("Connection failed", choice)
		}
		return fmt.Errorf("failed to connect: %w", err)
	}

	if cfg.ShowNotify {
		notify("Connected to", choice)
	}

	return nil
}

func disconnect(cfg *Config) error {
	// Get current connection
	cmd := exec.Command("nmcli", "-t", "-f", "NAME", "con", "show", "--active")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get active connection: %w", err)
	}

	connection := strings.TrimSpace(string(output))
	if connection == "" {
		return fmt.Errorf("no active connection")
	}

	// Disconnect
	cmd = exec.Command("nmcli", "con", "down", connection)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disconnect: %w", err)
	}

	if cfg.ShowNotify {
		notify("Disconnected from", connection)
	}

	return nil
}

func showCurrentConnection(cfg *Config) error {
	cmd := exec.Command("nmcli", "-t", "-f", "NAME,TYPE,DEVICE", "con", "show", "--active")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get connection info: %w", err)
	}

	info := strings.TrimSpace(string(output))
	if info == "" {
		info = "No active connection"
	}

	if cfg.ShowNotify {
		notify("Current Connection", info)
	}

	return nil
}

func toggleWifi(cfg *Config) error {
	// Check current state
	cmd := exec.Command("nmcli", "radio", "wifi")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get wifi state: %w", err)
	}

	state := strings.TrimSpace(string(output))

	// Toggle
	var newState string
	if state == "enabled" {
		cmd = exec.Command("nmcli", "radio", "wifi", "off")
		newState = "disabled"
	} else {
		cmd = exec.Command("nmcli", "radio", "wifi", "on")
		newState = "enabled"
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to toggle wifi: %w", err)
	}

	if cfg.ShowNotify {
		notify("WiFi", fmt.Sprintf("WiFi %s", newState))
	}

	return nil
}

func notify(title, message string) {
	if _, err := exec.LookPath("notify-send"); err == nil {
		exec.Command("notify-send", "ql wifi", fmt.Sprintf("%s\n%s", title, message)).Run()
	}
}

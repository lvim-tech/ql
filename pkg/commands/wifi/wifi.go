// Package wifi provides WiFi network management functionality for ql.
// It uses nmcli (NetworkManager) to scan and connect to wireless networks,
// with optional password input and connection testing.
package wifi

import (
	"fmt"
	"os"
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

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfgInterface := ctx.Config().GetWifiConfig()

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
			Error:   fmt.Errorf("wifi module is disabled in config"),
		}
	}

	if _, err := exec.LookPath("nmcli"); err != nil {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("nmcli is not installed (required for wifi management)"),
		}
	}

	for {
		options := []string{
			"← Back",
			"Connect to Network",
			"Disconnect",
			"Show Current Connection",
			"Toggle WiFi",
		}

		choice, err := ctx.Show(options, "WiFi")
		if err != nil {
			return commands.CommandResult{Success: false}
		}

		if choice == "← Back" {
			return commands.CommandResult{Success: false}
		}

		var actionErr error
		switch choice {
		case "Connect to Network":
			actionErr = connectToNetwork(ctx, &cfg)
		case "Disconnect":
			actionErr = disconnect(&cfg)
		case "Show Current Connection":
			actionErr = showCurrentConnection(&cfg)
		case "Toggle WiFi":
			actionErr = toggleWifi(&cfg)
		default:
			showErrorNotification("WiFi Error", fmt.Sprintf("Unknown choice: %s", choice))
			continue
		}

		if actionErr != nil {
			showErrorNotification("WiFi Error", actionErr.Error())
			continue
		}

		return commands.CommandResult{Success: true}
	}
}

func connectToNetwork(ctx commands.LauncherContext, cfg *Config) error {
	cmd := exec.Command("nmcli", "-t", "-f", "SSID", "dev", "wifi", "list")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to scan networks:  %w", err)
	}

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

	networks = append([]string{"← Back"}, networks...)

	choice, err := ctx.Show(networks, "Select Network")
	if err != nil {
		return fmt.Errorf("cancelled")
	}

	if choice == "← Back" {
		return fmt.Errorf("cancelled")
	}

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
	cmd := exec.Command("nmcli", "-t", "-f", "NAME", "con", "show", "--active")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get active connection: %w", err)
	}

	connection := strings.TrimSpace(string(output))
	if connection == "" {
		return fmt.Errorf("no active connection")
	}

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
	cmd := exec.Command("nmcli", "radio", "wifi")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get wifi state: %w", err)
	}

	state := strings.TrimSpace(string(output))

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

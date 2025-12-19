// Package wifi provides WiFi network management functionality for ql.
// It uses nmcli (NetworkManager) to scan and connect to wireless networks,
// with optional password input and connection testing.
package wifi

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

	if !utils.CommandExists("nmcli") {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("nmcli is not installed (required for wifi management)"),
		}
	}

	notifCfg := ctx.Config().GetNotificationConfig()

	// Main loop - keeps showing menu until Back, ESC, or successful action
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
			// ESC pressed - exit completely
			return commands.CommandResult{Success: false}
		}

		if choice == "← Back" {
			// Back pressed - return to module menu
			return commands.CommandResult{
				Success: false,
				Error:   commands.ErrBack,
			}
		}

		var actionErr error
		switch choice {
		case "Connect to Network":
			actionErr = connectToNetwork(ctx, &cfg, &notifCfg)
		case "Disconnect":
			actionErr = disconnect(&cfg, &notifCfg)
		case "Show Current Connection":
			actionErr = showCurrentConnection(&cfg, &notifCfg)
		case "Toggle WiFi":
			actionErr = toggleWifi(&cfg, &notifCfg)
		default:
			utils.ShowErrorNotificationWithConfig(&notifCfg, "WiFi Error", fmt.Sprintf("Unknown choice: %s", choice))
			continue
		}

		if actionErr != nil {
			// Check if user cancelled in submenu
			if actionErr.Error() == "cancelled" {
				// Loop back to main menu
				continue
			}
			// Other error - show notification and loop back
			utils.ShowErrorNotificationWithConfig(&notifCfg, "WiFi Error", actionErr.Error())
			continue
		}

		// Action succeeded - exit
		return commands.CommandResult{Success: true}
	}
}

func connectToNetwork(ctx commands.LauncherContext, cfg *Config, notifCfg *config.NotificationConfig) error {
	cmd := exec.Command("nmcli", "-t", "-f", "SSID", "dev", "wifi", "list")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to scan networks: %w", err)
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
		// ESC pressed in network selection
		return fmt.Errorf("cancelled")
	}

	if choice == "← Back" {
		// Back pressed in network selection
		return fmt.Errorf("cancelled")
	}

	cmd = exec.Command("nmcli", "dev", "wifi", "connect", choice)
	output, err = cmd.CombinedOutput()

	if err != nil {
		if strings.Contains(string(output), "Secrets were required") ||
			strings.Contains(string(output), "password") {

			password, passErr := utils.PromptPassword("WiFi Password")
			if passErr != nil || password == "" {
				return fmt.Errorf("password required but not provided")
			}

			cmd = exec.Command("nmcli", "dev", "wifi", "connect", choice, "password", password)
			output, err = cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to connect: %s", strings.TrimSpace(string(output)))
			}
		} else {
			return fmt.Errorf("failed to connect: %s", strings.TrimSpace(string(output)))
		}
	}

	if cfg.ShowNotify {
		utils.NotifyWithConfig(notifCfg, "WiFi Connected", choice)
	}

	if cfg.TestHost != "" {
		if testErr := testConnection(cfg); testErr != nil {
			if cfg.ShowNotify {
				utils.ShowErrorNotificationWithConfig(notifCfg, "WiFi Warning", fmt.Sprintf("Connected but no internet: %v", testErr))
			}
		}
	}

	return nil
}

func disconnect(cfg *Config, notifCfg *config.NotificationConfig) error {
	cmd := exec.Command("nmcli", "-t", "-f", "NAME,TYPE", "con", "show", "--active")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get active connections: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var wifiConnection string

	for _, line := range lines {
		if strings.Contains(line, "802-11-wireless") || strings.Contains(line, "wireless") {
			parts := strings.Split(line, ":")
			if len(parts) > 0 {
				wifiConnection = parts[0]
				break
			}
		}
	}

	if wifiConnection == "" {
		return fmt.Errorf("no active WiFi connection")
	}

	cmd = exec.Command("nmcli", "con", "down", wifiConnection)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to disconnect: %s", strings.TrimSpace(string(output)))
	}

	if cfg.ShowNotify {
		utils.NotifyWithConfig(notifCfg, "WiFi Disconnected", wifiConnection)
	}

	return nil
}

func showCurrentConnection(cfg *Config, notifCfg *config.NotificationConfig) error {
	cmd := exec.Command("nmcli", "-t", "-f", "NAME,TYPE,DEVICE", "con", "show", "--active")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get connection info: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var wifiInfo string

	for _, line := range lines {
		if strings.Contains(line, "802-11-wireless") || strings.Contains(line, "wireless") {
			parts := strings.Split(line, ":")
			if len(parts) >= 3 {
				wifiInfo = fmt.Sprintf("Network: %s\nDevice: %s", parts[0], parts[2])
				break
			}
		}
	}

	if wifiInfo == "" {
		wifiInfo = "Not connected to WiFi"
	}

	if cfg.ShowNotify {
		utils.NotifyWithConfig(notifCfg, "WiFi Status", wifiInfo)
	}

	return nil
}

func toggleWifi(cfg *Config, notifCfg *config.NotificationConfig) error {
	cmd := exec.Command("nmcli", "radio", "wifi")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get WiFi state: %w", err)
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

	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to toggle WiFi: %s", strings.TrimSpace(string(output)))
	}

	if cfg.ShowNotify {
		utils.NotifyWithConfig(notifCfg, "WiFi", fmt.Sprintf("WiFi %s", newState))
	}

	return nil
}

func testConnection(cfg *Config) error {
	if !utils.CommandExists("ping") {
		return fmt.Errorf("ping command not found")
	}

	cmd := exec.Command("ping",
		"-c", fmt.Sprintf("%d", cfg.TestCount),
		"-W", fmt.Sprintf("%d", cfg.TestWait),
		cfg.TestHost)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("connection test failed: %s", strings.TrimSpace(string(output)))
	}

	return nil
}

// Package wifi provides WiFi network management functionality for ql.
// It uses nmcli (NetworkManager) to scan and connect to wireless networks,
// with optional password input and connection testing.
package wifi

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/launcher"
)

func init() {
	commands.Register(commands.Command{
		Name:        "wifi",
		Description: "Connect to WiFi networks",
		Run:         Run,
	})
}

func Run(ctx *launcher.Context) error {
	cfg := config.Get()
	wifiCfg := cfg.Commands.Wifi

	// Провери дали е enabled
	if !wifiCfg.Enabled {
		return fmt.Errorf("wifi module is disabled in config")
	}

	// Провери дали има nmcli
	if _, err := exec.LookPath("nmcli"); err != nil {
		return fmt.Errorf("nmcli is not installed")
	}

	// Провери дали NetworkManager работи
	if err := checkNetworkManager(); err != nil {
		return err
	}

	// Включи WiFi ако е изключен
	enableWifi()

	// Rescan за нови мрежи
	rescanWifi()

	// Вземи списък с WiFi мрежи
	networks, err := getWifiNetworks()
	if err != nil {
		return fmt.Errorf("failed to get wifi networks: %w", err)
	}

	if len(networks) == 0 {
		return fmt.Errorf("no wifi networks found")
	}

	// Покажи меню
	choice, err := ctx.Show(networks, "Select WiFi")
	if err != nil {
		return err
	}

	// Извлечи SSID
	ssid := extractSSID(choice)

	// Попитай за парола
	password, err := ctx.ShowInput("Enter Password", "")
	if err != nil {
		return err
	}

	// Свържи се
	if err := connectToWifi(ssid, password); err != nil {
		if wifiCfg.ShowNotify {
			notify("Failed to connect to WiFi")
		}
		return fmt.Errorf("failed to connect:  %w", err)
	}

	// Изчакай малко
	time.Sleep(3 * time.Second)

	// Тествай connection
	if testConnection(wifiCfg.TestHost, wifiCfg.TestCount, wifiCfg.TestWait) {
		if wifiCfg.ShowNotify {
			notify("WiFi connected!  Internet is working")
		}
	} else {
		if wifiCfg.ShowNotify {
			notify("WiFi connected but internet is not working")
		}
	}

	return nil
}

// checkNetworkManager проверява дали NetworkManager работи
func checkNetworkManager() error {
	cmd := exec.Command("nmcli", "general", "status")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("NetworkManager is not running.  Please start it with: sudo systemctl start NetworkManager")
	}
	return nil
}

// enableWifi включва WiFi ако е изключен
func enableWifi() {
	exec.Command("nmcli", "radio", "wifi", "on").Run()
}

// rescanWifi прави rescan за нови мрежи
func rescanWifi() {
	exec.Command("nmcli", "device", "wifi", "rescan").Run()
	time.Sleep(2 * time.Second)
}

// getWifiNetworks връща списък с WiFi мрежи
func getWifiNetworks() ([]string, error) {
	cmd := exec.Command("nmcli", "--fields", "SSID,SIGNAL,SECURITY", "device", "wifi", "list")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("nmcli error: %s (stderr: %s)", err, stderr.String())
	}

	output := stdout.String()
	lines := strings.Split(output, "\n")

	if len(lines) <= 1 {
		return nil, fmt.Errorf("no networks found")
	}

	// Skip header line и празни редове
	var networks []string
	for i, line := range lines {
		if i == 0 { // Skip header
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Добави мрежата
		networks = append(networks, line)
	}

	if len(networks) == 0 {
		return nil, fmt.Errorf("no networks found after parsing")
	}

	return networks, nil
}

// extractSSID извлича SSID от избрания ред
func extractSSID(choice string) string {
	// Формат: "SSID  SIGNAL  SECURITY"
	// Вземи първото поле (SSID)
	fields := strings.Fields(choice)
	if len(fields) > 0 {
		return fields[0]
	}
	return choice
}

// connectToWifi се свързва към WiFi мрежа
func connectToWifi(ssid, password string) error {
	var cmd *exec.Cmd

	if password != "" {
		cmd = exec.Command("nmcli", "device", "wifi", "connect", ssid, "password", password)
	} else {
		cmd = exec.Command("nmcli", "device", "wifi", "connect", ssid)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, stderr.String())
	}

	return nil
}

// testConnection тества дали интернет връзката работи
func testConnection(host string, count, wait int) bool {
	cmd := exec.Command("ping",
		"-q",
		"-c", fmt.Sprintf("%d", count),
		"-W", fmt.Sprintf("%d", wait),
		host)

	return cmd.Run() == nil
}

// notify изпраща notification
func notify(message string) {
	if _, err := exec.LookPath("notify-send"); err == nil {
		exec.Command("notify-send", "ql wifi", message).Run()
	}
}

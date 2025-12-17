// Package weather provides weather information functionality for ql.
// It fetches weather data from wttr.in and displays it.
package weather

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/mitchellh/mapstructure"
)

func init() {
	commands.Register(commands.Command{
		Name:        "weather",
		Description: "Check weather information",
		Run:         Run,
	})
}

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfgInterface := ctx.Config().GetWeatherConfig()

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
			Error:   fmt.Errorf("weather module is disabled in config"),
		}
	}

	if len(cfg.Locations) == 0 {
		cfg.Locations = []string{"Sofia", "London", "New York"}
	}

	for {
		var items []string
		items = append(items, "← Back")
		items = append(items, cfg.Locations...)

		choice, err := ctx.Show(items, "Weather Location")
		if err != nil {
			return commands.CommandResult{Success: false}
		}

		if choice == "← Back" {
			return commands.CommandResult{Success: false}
		}

		notifyID := showPersistentNotification("Weather", fmt.Sprintf("Fetching weather for %s.. .", choice))

		weatherData, err := fetchWeather(choice, cfg.Options, cfg.Timeout)

		closePersistentNotification(notifyID)

		if err != nil {
			showErrorNotification("Weather Error", fmt.Sprintf("Failed to fetch weather for %s:\n%v", choice, err))
			continue
		}

		if isatty() {
			displayWeatherTerminal(weatherData)
		} else {
			displayWeatherGUI(weatherData)
		}

		return commands.CommandResult{Success: true}
	}
}

func isatty() bool {
	stdinInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	stdinIsTTY := (stdinInfo.Mode() & os.ModeCharDevice) != 0

	if !stdinIsTTY {
		return false
	}

	tty, err := os.Open("/dev/tty")
	if err != nil {
		return false
	}
	tty.Close()

	return true
}

func showPersistentNotification(title, message string) int {
	notifyID := int(time.Now().UnixNano() % 1000000)

	if _, err := exec.LookPath("dunstify"); err == nil {
		cmd := exec.Command("dunstify",
			"-u", "normal",
			"-t", "0",
			"-r", strconv.Itoa(notifyID),
			title,
			message)
		cmd.Env = os.Environ()
		cmd.Start()
		return notifyID
	}

	if _, err := exec.LookPath("notify-send"); err == nil {
		cmd := exec.Command("notify-send",
			"-u", "normal",
			"-t", "0",
			title,
			message)
		cmd.Env = os.Environ()
		cmd.Start()
		return notifyID
	}

	return notifyID
}

func closePersistentNotification(notifyID int) {
	if _, err := exec.LookPath("dunstify"); err == nil {
		cmd := exec.Command("dunstify", "-C", strconv.Itoa(notifyID))
		cmd.Env = os.Environ()
		cmd.Run()
		return
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

func fetchWeather(location string, options string, timeout int) (string, error) {
	location = strings.ReplaceAll(location, " ", "%20")

	url := fmt.Sprintf("https://wttr.in/%s?T", location)
	if options != "" {
		url += "&" + options
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "curl/7.88.0")

	if timeout == 0 {
		timeout = 30
	}

	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(body), nil
}

func displayWeatherTerminal(data string) error {
	fmt.Println(data)
	return nil
}

func displayWeatherGUI(data string) error {
	if _, err := exec.LookPath("yad"); err == nil {
		tmpFile := "/tmp/ql-weather-data.txt"
		if err := os.WriteFile(tmpFile, []byte(data), 0644); err == nil {
			defer os.Remove(tmpFile)

			cmd := exec.Command("yad",
				"--text-info",
				"--title=Weather",
				"--width=800",
				"--height=600",
				"--fontname=Monospace 10",
				"--filename="+tmpFile)
			cmd.Env = os.Environ()
			return cmd.Run()
		}
	}

	if _, err := exec.LookPath("zenity"); err == nil {
		tmpFile := "/tmp/ql-weather-data.txt"
		if err := os.WriteFile(tmpFile, []byte(data), 0644); err == nil {
			defer os.Remove(tmpFile)

			cmd := exec.Command("zenity",
				"--text-info",
				"--title=Weather",
				"--width=800",
				"--height=600",
				"--filename="+tmpFile)
			cmd.Env = os.Environ()
			return cmd.Run()
		}
	}

	terminal := detectTerminal()
	if terminal != "" {
		tmpScript := "/tmp/ql-weather. sh"
		script := fmt.Sprintf("#!/bin/sh\ncat << 'EOF'\n%s\nEOF\necho ''\necho 'Press Enter to close...'\nread\n", data)

		if err := os.WriteFile(tmpScript, []byte(script), 0755); err == nil {
			defer os.Remove(tmpScript)
			return exec.Command(terminal, "-e", tmpScript).Run()
		}
	}

	return displayWeatherTerminal(data)
}

func detectTerminal() string {
	terminals := []string{
		"kitty",
		"alacritty",
		"foot",
		"wezterm",
		"gnome-terminal",
		"konsole",
		"xterm",
	}

	for _, term := range terminals {
		if _, err := exec.LookPath(term); err == nil {
			return term
		}
	}

	return ""
}

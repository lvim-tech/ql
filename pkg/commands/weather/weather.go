// Package weather provides weather information functionality for ql.
// It fetches weather data from wttr.in and displays it.
package weather

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/utils"
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

	notifCfg := ctx.Config().GetNotificationConfig()

	// Check for direct command
	args := ctx.Args()
	if len(args) > 0 {
		return executeDirectCommand(args, &cfg, &notifCfg)
	}

	for {
		var items []string

		if !ctx.IsDirectLaunch() {
			items = append(items, "← Back")
		}

		items = append(items, cfg.Locations...)

		choice, err := ctx.Show(items, "Weather Location")
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

		notifyID := utils.ShowPersistentNotificationWithConfig(&notifCfg, "Weather", fmt.Sprintf("Fetching weather for %s...", choice))

		weatherData, err := fetchWeather(choice, cfg.Options, cfg.Timeout)

		utils.ClosePersistentNotificationWithConfig(&notifCfg, notifyID)

		if err != nil {
			// Error fetching weather - show notification and loop back
			utils.ShowErrorNotificationWithConfig(&notifCfg, "Weather Error", fmt.Sprintf("Failed to fetch weather for %s:\n%v", choice, err))
			continue
		}

		// Display weather data
		if utils.IsTerminal() {
			displayWeatherTerminal(weatherData)
		} else {
			displayWeatherGUI(weatherData)
		}

		// Weather displayed successfully - exit
		return commands.CommandResult{Success: true}
	}
}

func executeDirectCommand(args []string, cfg *Config, notifCfg *config.NotificationConfig) commands.CommandResult {
	// Join all args as location name (supports "New York" etc.)
	location := strings.Join(args, " ")

	// Check if location is in configured locations (case-insensitive partial match)
	var matchedLocation string
	locationLower := strings.ToLower(location)

	for _, configLoc := range cfg.Locations {
		configLocLower := strings.ToLower(configLoc)
		if configLocLower == locationLower || strings.Contains(configLocLower, locationLower) {
			matchedLocation = configLoc
			break
		}
	}

	// If not found in config, use the provided location directly
	if matchedLocation == "" {
		matchedLocation = location
	}

	notifyID := utils.ShowPersistentNotificationWithConfig(notifCfg, "Weather", fmt.Sprintf("Fetching weather for %s...", matchedLocation))

	weatherData, err := fetchWeather(matchedLocation, cfg.Options, cfg.Timeout)

	utils.ClosePersistentNotificationWithConfig(notifCfg, notifyID)

	if err != nil {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("failed to fetch weather for %s: %w", matchedLocation, err),
		}
	}

	// Display weather data
	if utils.IsTerminal() {
		displayWeatherTerminal(weatherData)
	} else {
		displayWeatherGUI(weatherData)
	}

	return commands.CommandResult{Success: true}
}

func fetchWeather(location string, options string, timeout int) (string, error) {
	location = strings.ReplaceAll(location, " ", "%20")

	url := fmt.Sprintf("https://wttr.in/%s?T", location)
	if options != "" {
		url += "&" + options
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request:     %w", err)
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
	if utils.CommandExists("yad") {
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

	if utils.CommandExists("zenity") {
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

	terminal := utils.DetectTerminal()
	if terminal != "" {
		tmpScript := "/tmp/ql-weather. sh"
		script := fmt.Sprintf("#!/bin/sh\ncat << 'EOF'\n%s\nEOF\necho ''\necho 'Press Enter to close... '\nread\n", data)

		if err := os.WriteFile(tmpScript, []byte(script), 0755); err == nil {
			defer os.Remove(tmpScript)
			return exec.Command(terminal, "-e", tmpScript).Run()
		}
	}

	return displayWeatherTerminal(data)
}

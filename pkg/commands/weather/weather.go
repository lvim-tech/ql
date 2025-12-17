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

// Run executes the weather command
func Run(ctx commands.LauncherContext) error {
	// Get config
	cfgInterface := ctx.Config().GetWeatherConfig()

	// Decode config
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
		return fmt.Errorf("weather module is disabled in config")
	}

	// If no locations configured, use default
	if len(cfg.Locations) == 0 {
		cfg.Locations = []string{"Sofia", "London", "New York"}
	}

	// Show location menu
	choice, err := ctx.Show(cfg.Locations, "Weather Location")
	if err != nil {
		return err
	}

	// Show persistent notification while fetching
	notifyID := showPersistentNotification("Weather", fmt.Sprintf("Fetching weather for %s.. .", choice))

	// Fetch weather data
	weatherData, err := fetchWeather(choice, cfg.Options, cfg.Timeout)

	// Close the persistent notification
	closePersistentNotification(notifyID)

	if err != nil {
		// Show error notification (stays for 10 seconds)
		showErrorNotification("Weather Error", fmt.Sprintf("Failed to fetch weather for %s:\n%v", choice, err))
		return fmt.Errorf("failed to fetch weather: %w", err)
	}

	// Auto-detect display method based on TTY
	if isatty() {
		// We have a terminal - show there
		return displayWeatherTerminal(weatherData)
	}

	// No terminal (launched from hotkey) - use GUI
	return displayWeatherGUI(weatherData)
}

// isatty checks if we're running in an interactive terminal
func isatty() bool {
	// Check if stdin is a terminal
	stdinInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	stdinIsTTY := (stdinInfo.Mode() & os.ModeCharDevice) != 0

	if !stdinIsTTY {
		return false
	}

	// Try to open /dev/tty - if this fails, we're not in an interactive session
	tty, err := os.Open("/dev/tty")
	if err != nil {
		// No controlling terminal = launched from GUI/hotkey
		return false
	}
	tty.Close()

	return true
}

// showPersistentNotification shows a notification that stays until closed
// Returns a notification ID that can be used to close it
func showPersistentNotification(title, message string) int {
	// Generate unique ID
	notifyID := int(time.Now().UnixNano() % 1000000)

	// Try dunstify (supports replace-id for persistent notifications)
	if _, err := exec.LookPath("dunstify"); err == nil {
		cmd := exec.Command("dunstify",
			"-u", "normal",
			"-t", "0", // 0 = never expire
			"-r", strconv.Itoa(notifyID), // Replace ID as string
			title,
			message)
		cmd.Env = os.Environ()
		cmd.Start()
		return notifyID
	}

	// Fallback to notify-send (no replace support, but works)
	if _, err := exec.LookPath("notify-send"); err == nil {
		cmd := exec.Command("notify-send",
			"-u", "normal",
			"-t", "0", // 0 = never expire (may not work on all notification daemons)
			title,
			message)
		cmd.Env = os.Environ()
		cmd.Start()
		return notifyID
	}

	return notifyID
}

// closePersistentNotification closes a persistent notification
func closePersistentNotification(notifyID int) {
	// Try dunstify close
	if _, err := exec.LookPath("dunstify"); err == nil {
		// Close notification with ID
		cmd := exec.Command("dunstify", "-C", strconv.Itoa(notifyID))
		cmd.Env = os.Environ()
		cmd.Run()
		return
	}

	// notify-send doesn't support closing, notification will timeout naturally
}

// showErrorNotification shows an error notification (stays for 10 seconds)
func showErrorNotification(title, message string) {
	// Try dunstify first
	if _, err := exec.LookPath("dunstify"); err == nil {
		cmd := exec.Command("dunstify",
			"-u", "critical",
			"-t", "10000", // 10 seconds
			title,
			message)
		cmd.Env = os.Environ()
		cmd.Start()
		return
	}

	// Fallback to notify-send
	if _, err := exec.LookPath("notify-send"); err == nil {
		cmd := exec.Command("notify-send",
			"-u", "critical",
			"-t", "10000", // 10 seconds
			title,
			message)
		cmd.Env = os.Environ()
		cmd.Start()
		return
	}
}

func fetchWeather(location string, options string, timeout int) (string, error) {
	// Replace spaces with %20 for URL
	location = strings.ReplaceAll(location, " ", "%20")

	// Build URL with options
	url := fmt.Sprintf("https://wttr.in/%s?T", location)
	if options != "" {
		url += "&" + options
	}

	// Create HTTP request with curl User-Agent
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent to curl (forces plain text response)
	req.Header.Set("User-Agent", "curl/7.88.0")

	// Default timeout if not specified
	if timeout == 0 {
		timeout = 30
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	// Fetch weather data
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(body), nil
}

func displayWeatherTerminal(data string) error {
	// Simply print to stdout
	fmt.Println(data)
	return nil
}

func displayWeatherGUI(data string) error {
	// 1. Try yad (best for large text)
	if _, err := exec.LookPath("yad"); err == nil {
		// Write data to temp file (more reliable than stdin pipe)
		tmpFile := "/tmp/ql-weather-data. txt"
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

	// 2. Try zenity
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

	// 3. Try opening in auto-detected terminal emulator
	terminal := detectTerminal()
	if terminal != "" {
		tmpScript := "/tmp/ql-weather. sh"
		script := fmt.Sprintf("#!/bin/sh\ncat << 'EOF'\n%s\nEOF\necho ''\necho 'Press Enter to close...'\nread\n", data)

		if err := os.WriteFile(tmpScript, []byte(script), 0755); err == nil {
			defer os.Remove(tmpScript)
			return exec.Command(terminal, "-e", tmpScript).Run()
		}
	}

	// 4. Fallback:  print to stdout
	return displayWeatherTerminal(data)
}

func detectTerminal() string {
	// Check common terminal emulators
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

// Package radio provides online radio streaming functionality for ql.
// It supports multiple radio stations configured via TOML and uses mpv for playback.
package radio

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/utils"
)

func init() {
	commands.Register(commands.Command{
		Name:        "radio",
		Description: "Internet radio player",
		Run:         Run,
	})
}

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfg := commands.DecodeConfig(ctx, "radio", DefaultConfig())

	if !cfg.Enabled {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("radio module is disabled in config"),
		}
	}

	if !utils.CommandExists("mpv") {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("mpv is not installed"),
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

		options = append(options, "Play Station", "Stop Radio")

		choice, err := ctx.Show(options, "Radio")
		if err != nil {
			// ESC pressed at main menu - exit completely
			return commands.CommandResult{Success: false}
		}

		if choice == "← Back" {
			return commands.CommandResult{
				Success: false,
				Error:   commands.ErrBack,
			}
		}

		var actionErr error
		switch choice {
		case "Play Station":
			actionErr = playStation(ctx, &cfg, &notifCfg)
		case "Stop Radio":
			actionErr = stopRadio(&notifCfg)
		default:
			utils.ShowErrorNotificationWithConfig(&notifCfg, "Radio Error", fmt.Sprintf("Unknown choice: %s", choice))
			continue
		}

		if actionErr != nil {
			// ESC from the submenu exits completely. A typed sentinel, not
			// a string compare — commands.ErrCancelled matched by text was
			// one config change away from silently breaking.
			if errors.Is(actionErr, commands.ErrCancelled) {
				return commands.CommandResult{Success: false}
			}
			// Other error - show and loop back
			utils.ShowErrorNotificationWithConfig(&notifCfg, "Radio Error", actionErr.Error())
			continue
		}

		// Action succeeded - exit
		return commands.CommandResult{Success: true}
	}
}

func executeDirectCommand(args []string, cfg *Config, notifCfg *config.NotificationConfig) commands.CommandResult {
	action := strings.ToLower(args[0])

	var err error

	switch action {
	case "stop":
		err = stopRadio(notifCfg)

	case "play":
		// If station name is provided, play it directly
		if len(args) > 1 {
			stationName := strings.Join(args[1:], " ")
			err = playStationDirect(stationName, cfg, notifCfg)
		} else {
			return commands.CommandResult{
				Success: false,
				Error:   fmt.Errorf("usage: ql radio play <station name>"),
			}
		}

	default:
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("unknown radio action: %s (use:  play, stop)", action),
		}
	}

	if err != nil {
		return commands.CommandResult{Success: false, Error: err}
	}

	return commands.CommandResult{Success: true}
}

func playStationDirect(stationName string, cfg *Config, notifCfg *config.NotificationConfig) error {
	// Find station by name (case-insensitive partial match)
	var matchedStation string
	var matchedURL string

	stationNameLower := strings.ToLower(stationName)

	for name, url := range cfg.RadioStations {
		nameLower := strings.ToLower(name)
		if nameLower == stationNameLower || strings.Contains(nameLower, stationNameLower) {
			matchedStation = name
			matchedURL = url
			break
		}
	}

	if matchedURL == "" {
		return fmt.Errorf("station not found: %s", stationName)
	}

	// Stop any playing radio first
	stopRadio(notifCfg)

	args := []string{
		"--no-video",
		fmt.Sprintf("--volume=%d", cfg.Volume),
		matchedURL,
	}

	if err := utils.StartDetachedProcess("mpv", args...); err != nil {
		return fmt.Errorf("failed to start radio: %w", err)
	}

	utils.NotifyWithConfig(notifCfg, "Radio", fmt.Sprintf("Playing: %s", matchedStation))

	return nil
}

func playStation(ctx commands.LauncherContext, cfg *Config, notifCfg *config.NotificationConfig) error {
	var stations []string
	stationMap := make(map[string]string)

	for name, url := range cfg.RadioStations {
		stations = append(stations, name)
		stationMap[name] = url
	}

	if len(stations) == 0 {
		return fmt.Errorf("no radio stations configured")
	}

	stations = append([]string{"← Back"}, stations...)

	choice, err := ctx.Show(stations, "Select Station")
	if err != nil {
		return commands.ErrCancelled
	}

	if choice == "← Back" {
		return commands.ErrCancelled
	}

	url, ok := stationMap[choice]
	if !ok {
		return fmt.Errorf("station not found: %s", choice)
	}

	stopRadio(notifCfg)

	args := []string{
		"--no-video",
		fmt.Sprintf("--volume=%d", cfg.Volume),
		url,
	}

	if err := utils.StartDetachedProcess("mpv", args...); err != nil {
		return fmt.Errorf("failed to start radio: %w", err)
	}

	utils.NotifyWithConfig(notifCfg, "Radio", fmt.Sprintf("Playing: %s", choice))

	return nil
}

func stopRadio(notifCfg *config.NotificationConfig) error {
	if err := utils.KillProcessByName("mpv"); err != nil {
		return err
	}

	utils.NotifyWithConfig(notifCfg, "Radio", "Stopped")
	return nil
}

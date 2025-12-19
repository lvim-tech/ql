// Package radio provides online radio streaming functionality for ql.
// It supports multiple radio stations configured via TOML and uses mpv for playback.
package radio

import (
	"fmt"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/utils"
	"github.com/mitchellh/mapstructure"
)

func init() {
	commands.Register(commands.Command{
		Name:        "radio",
		Description: "Internet radio player",
		Run:         Run,
	})
}

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfgInterface := ctx.Config().GetRadioConfig()

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

	// Main loop - keeps showing menu until Back, ESC, or successful action
	for {
		options := []string{"← Back", "Play Station", "Stop Radio"}

		choice, err := ctx.Show(options, "Radio")
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
		case "Play Station":
			actionErr = playStation(ctx, &cfg, &notifCfg)
		case "Stop Radio":
			actionErr = stopRadio(&notifCfg)
		default:
			utils.ShowErrorNotificationWithConfig(&notifCfg, "Radio Error", fmt.Sprintf("Unknown choice: %s", choice))
			continue
		}

		if actionErr != nil {
			// Check if user cancelled in submenu
			if actionErr.Error() == "cancelled" {
				// Loop back to main menu
				continue
			}
			// Other error - show notification and loop back
			utils.ShowErrorNotificationWithConfig(&notifCfg, "Radio Error", actionErr.Error())
			continue
		}

		// Action succeeded - exit
		return commands.CommandResult{Success: true}
	}
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
		// ESC pressed in station selection
		return fmt.Errorf("cancelled")
	}

	if choice == "← Back" {
		// Back pressed in station selection
		return fmt.Errorf("cancelled")
	}

	url, ok := stationMap[choice]
	if !ok {
		return fmt.Errorf("station not found:  %s", choice)
	}

	stopRadio(notifCfg)

	args := []string{
		"--no-video",
		fmt.Sprintf("--volume=%d", cfg.Volume),
		url,
	}

	if err := utils.StartDetachedProcess("mpv", args...); err != nil {
		return fmt.Errorf("failed to start radio:  %w", err)
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

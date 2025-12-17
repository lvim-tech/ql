// Package radio provides online radio streaming functionality for ql.
// It supports multiple radio stations configured via TOML and uses mpv for playback.
package radio

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/lvim-tech/ql/pkg/commands"
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

	if _, err := exec.LookPath("mpv"); err != nil {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("mpv is not installed"),
		}
	}

	for {
		options := []string{"← Back", "Play Station", "Stop Radio"}

		choice, err := ctx.Show(options, "Radio")
		if err != nil {
			return commands.CommandResult{Success: false}
		}

		if choice == "← Back" {
			return commands.CommandResult{Success: false}
		}

		var actionErr error
		switch choice {
		case "Play Station":
			actionErr = playStation(ctx, &cfg)
		case "Stop Radio":
			actionErr = stopRadio()
		default:
			showErrorNotification("Radio Error", fmt.Sprintf("Unknown choice: %s", choice))
			continue
		}

		if actionErr != nil {
			showErrorNotification("Radio Error", actionErr.Error())
			continue
		}

		return commands.CommandResult{Success: true}
	}
}

func playStation(ctx commands.LauncherContext, cfg *Config) error {
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
		return fmt.Errorf("cancelled")
	}

	if choice == "← Back" {
		return fmt.Errorf("cancelled")
	}

	url, ok := stationMap[choice]
	if !ok {
		return fmt.Errorf("station not found: %s", choice)
	}

	stopRadio()

	cmd := exec.Command("mpv",
		"--no-video",
		fmt.Sprintf("--volume=%d", cfg.Volume),
		url,
	)

	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start radio:  %w", err)
	}

	pidFile := "/tmp/ql_radio.pid"
	pidData := fmt.Sprintf("%d", cmd.Process.Pid)
	if err := os.WriteFile(pidFile, []byte(pidData), 0644); err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	cmd.Process.Release()

	notify("Radio started", choice)

	return nil
}

func stopRadio() error {
	pidFile := "/tmp/ql_radio.pid"

	data, err := os.ReadFile(pidFile)
	if err != nil {
		return nil
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		os.Remove(pidFile)
		return nil
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(pidFile)
		return nil
	}

	process.Signal(syscall.SIGTERM)
	os.Remove(pidFile)

	notify("Radio stopped", "")

	return nil
}

func notify(title, message string) {
	if _, err := exec.LookPath("notify-send"); err == nil {
		exec.Command("notify-send", "ql radio", fmt.Sprintf("%s\n%s", title, message)).Run()
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

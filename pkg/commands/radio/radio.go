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

func Run(ctx commands.LauncherContext) error {
	// Извличаме config директно
	cfgInterface := ctx.Config().GetRadioConfig()
	if cfgInterface == nil {
		return fmt.Errorf("radio config not found")
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
		return fmt.Errorf("radio module is disabled in config")
	}

	// Check if mpv is installed
	if _, err := exec.LookPath("mpv"); err != nil {
		return fmt.Errorf("mpv is not installed")
	}

	// Menu options
	options := []string{"Play Station", "Stop Radio"}

	choice, err := ctx.Show(options, "Radio")
	if err != nil {
		return err
	}

	switch choice {
	case "Play Station":
		return playStation(ctx, &cfg)
	case "Stop Radio":
		return stopRadio()
	default:
		return fmt.Errorf("unknown choice: %s", choice)
	}
}

func playStation(ctx commands.LauncherContext, cfg *Config) error {
	// Build station list
	var stations []string
	stationMap := make(map[string]string)

	for name, url := range cfg.RadioStations {
		stations = append(stations, name)
		stationMap[name] = url
	}

	if len(stations) == 0 {
		return fmt.Errorf("no radio stations configured")
	}

	// Show station menu
	choice, err := ctx.Show(stations, "Select Station")
	if err != nil {
		return err
	}

	url, ok := stationMap[choice]
	if !ok {
		return fmt.Errorf("station not found: %s", choice)
	}

	// Stop any existing radio
	stopRadio()

	// Start mpv in background
	cmd := exec.Command("mpv",
		"--no-video",
		fmt.Sprintf("--volume=%d", cfg.Volume),
		url,
	)

	// Detach process
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

	// Save PID
	pidFile := "/tmp/ql_radio. pid"
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
		return nil // No radio running
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

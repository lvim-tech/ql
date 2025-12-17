// Package radio provides online radio streaming functionality for ql.
// It supports multiple radio stations configured via TOML and uses mpv for playback.
package radio

import (
	"fmt"
	"os/exec"
	"sort"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/launcher"
)

func init() {
	commands.Register(commands.Command{
		Name:        "radio",
		Description: "Online radio player",
		Run:         Run,
	})
}

func Run(ctx *launcher.Context) error {
	cfg := config.Get()
	radioCfg := cfg.Commands.Radio

	// Провери дали е enabled
	if !radioCfg.Enabled {
		return fmt.Errorf("radio module is disabled in config")
	}

	// Провери дали има mpv
	if _, err := exec.LookPath("mpv"); err != nil {
		return fmt.Errorf("mpv is not installed")
	}

	// Създай опции
	stations := radioCfg.RadioStations
	if len(stations) == 0 {
		return fmt.Errorf("no radio stations configured")
	}

	// Сортирай станциите по име
	var stationNames []string
	for name := range stations {
		stationNames = append(stationNames, name)
	}
	sort.Strings(stationNames)

	// Добави menu опции
	options := []string{"Play Station", "Stop Playing", "Quit"}

	// Покажи меню
	choice, err := ctx.Show(options, "Radio")
	if err != nil {
		return err
	}

	switch choice {
	case "Play Station":
		return playStation(ctx, stationNames, stations, radioCfg.Volume)
	case "Stop Playing":
		return stopRadio()
	case "Quit":
		return nil
	default:
		return nil
	}
}

func playStation(ctx *launcher.Context, stationNames []string, stations map[string]string, volume int) error {
	// Избери станция
	station, err := ctx.Show(stationNames, "Select Station")
	if err != nil {
		return err
	}

	url, ok := stations[station]
	if !ok {
		return fmt.Errorf("station not found:  %s", station)
	}

	// Спри текущо пускане
	stopRadio()

	// Notify
	notify(fmt.Sprintf("Starting radio: %s", station))

	// Пусни с mpv (background)
	cmd := exec.Command("mpv",
		fmt.Sprintf("--volume=%d", volume),
		"--no-video",
		"--no-terminal",
		url)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start mpv: %w", err)
	}

	return nil
}

func stopRadio() error {
	// Kill всички mpv процеси с http/https в аргументите
	exec.Command("pkill", "-f", "mpv.*http").Run()
	notify("Radio stopped")
	return nil
}

func notify(message string) {
	if _, err := exec.LookPath("notify-send"); err == nil {
		exec.Command("notify-send", "ql radio", message).Run()
	}
}

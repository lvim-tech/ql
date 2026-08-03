// Package audioout switches the default audio output (PipeWire/Pulse sink)
// and moves the currently playing streams over to it — the part people
// forget, which leaves music on the old speakers.
package audioout

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"os/exec"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/utils"
)

func init() {
	commands.Register(commands.Command{
		Name:        "audioout",
		Description: "Audio output device",
		Run:         Run,
	})
}

type sink struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfg := commands.DecodeConfig(ctx, "audioout", DefaultConfig())

	if !cfg.Enabled {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("audioout module is disabled in config"),
		}
	}

	notifCfg := ctx.Config().GetNotificationConfig()

	if !utils.CommandExists("pactl") {
		err := fmt.Errorf("pactl is not installed (pipewire-pulse or pulseaudio)")
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Audio Error", err.Error())
		return commands.CommandResult{Success: false, Error: err}
	}

	sinks, defaultSink, err := listSinks()
	if err != nil {
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Audio Error", err.Error())
		return commands.CommandResult{Success: false, Error: err}
	}
	if len(sinks) == 0 {
		err := fmt.Errorf("no audio sinks found")
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Audio Error", err.Error())
		return commands.CommandResult{Success: false, Error: err}
	}

	var options []string
	byOption := map[string]sink{}

	if !ctx.IsDirectLaunch() {
		options = append(options, "← Back")
	}
	for _, s := range sinks {
		marker := "  "
		if s.Name == defaultSink {
			marker = "● "
		}
		entry := marker + s.Description
		options = append(options, entry)
		byOption[entry] = s
	}

	choice, err := ctx.Show(options, "Audio Output")
	if err != nil {
		return commands.CommandResult{Success: false}
	}
	if choice == "← Back" {
		return commands.CommandResult{Success: false, Error: commands.ErrBack}
	}

	s := byOption[choice]
	if err := setDefaultSink(s); err != nil {
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Audio Error", err.Error())
		return commands.CommandResult{Success: false, Error: err}
	}

	utils.NotifyWithConfig(&notifCfg, "Audio Output", s.Description)
	return commands.CommandResult{Success: true}
}

func listSinks() ([]sink, string, error) {
	out, err := exec.Command("pactl", "-f", "json", "list", "sinks").Output()
	if err != nil {
		return nil, "", fmt.Errorf("pactl list sinks: %w", err)
	}
	var sinks []sink
	if err := json.Unmarshal(out, &sinks); err != nil {
		return nil, "", fmt.Errorf("pactl json: %w", err)
	}

	current := ""
	if out, err := exec.Command("pactl", "get-default-sink").Output(); err == nil {
		current = strings.TrimSpace(string(out))
	}
	return sinks, current, nil
}

// setDefaultSink switches the default and drags every playing stream along.
// New streams follow the default on their own; existing ones do not.
func setDefaultSink(s sink) error {
	if out, err := exec.Command("pactl", "set-default-sink", s.Name).CombinedOutput(); err != nil {
		return fmt.Errorf("set-default-sink: %s", strings.TrimSpace(string(out)))
	}

	out, err := exec.Command("pactl", "-f", "json", "list", "sink-inputs").Output()
	if err != nil {
		return nil // default is switched; stream moving is best-effort
	}
	var inputs []struct {
		Index int `json:"index"`
	}
	if json.Unmarshal(out, &inputs) != nil {
		return nil
	}
	for _, in := range inputs {
		// Best-effort per stream: some refuse to move (e.g. bound to a device).
		exec.Command("pactl", "move-sink-input", strconv.Itoa(in.Index), s.Name).Run()
	}
	return nil
}

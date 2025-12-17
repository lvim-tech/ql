// Package mpc provides MPD/MPC music player control functionality for ql.
// It supports playing music, managing playlists, and controlling playback via mpc commands.
package mpc

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/mitchellh/mapstructure"
)

func init() {
	commands.Register(commands.Command{
		Name:        "mpc",
		Description: "MPD client",
		Run:         Run,
	})
}

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfgInterface := ctx.Config().GetMpcConfig()

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
			Error:   fmt.Errorf("mpc module is disabled in config"),
		}
	}

	if _, err := exec.LookPath("mpc"); err != nil {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("mpc is not installed"),
		}
	}

	for {
		options := []string{
			"← Back",
			"Play/Pause",
			"Next",
			"Previous",
			"Stop",
			"Select Playlist",
			"Select Song",
			"Show Current",
		}

		choice, err := ctx.Show(options, "MPC")
		if err != nil {
			return commands.CommandResult{Success: false}
		}

		if choice == "← Back" {
			return commands.CommandResult{Success: false}
		}

		var actionErr error
		switch choice {
		case "Play/Pause":
			actionErr = togglePlayPause()
		case "Next":
			actionErr = next()
		case "Previous":
			actionErr = previous()
		case "Stop":
			actionErr = stop()
		case "Select Playlist":
			actionErr = selectPlaylist(ctx, &cfg)
		case "Select Song":
			actionErr = selectSong(ctx)
		case "Show Current":
			actionErr = showCurrent()
		default:
			showErrorNotification("MPC Error", fmt.Sprintf("Unknown choice: %s", choice))
			continue
		}

		if actionErr != nil {
			showErrorNotification("MPC Error", actionErr.Error())
			continue
		}

		return commands.CommandResult{Success: true}
	}
}

func togglePlayPause() error {
	cmd := exec.Command("mpc", "toggle")
	return cmd.Run()
}

func next() error {
	cmd := exec.Command("mpc", "next")
	return cmd.Run()
}

func previous() error {
	cmd := exec.Command("mpc", "prev")
	return cmd.Run()
}

func stop() error {
	cmd := exec.Command("mpc", "stop")
	return cmd.Run()
}

func selectPlaylist(ctx commands.LauncherContext, cfg *Config) error {
	cmd := exec.Command("mpc", "lsplaylists")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get playlists:  %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var playlists []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			playlists = append(playlists, line)
		}
	}

	if len(playlists) == 0 {
		return fmt.Errorf("no playlists found")
	}

	playlists = append([]string{"← Back"}, playlists...)

	choice, err := ctx.Show(playlists, "Select Playlist")
	if err != nil {
		return fmt.Errorf("cancelled")
	}

	if choice == "← Back" {
		return fmt.Errorf("cancelled")
	}

	cmd = exec.Command("mpc", "clear")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clear playlist: %w", err)
	}

	cmd = exec.Command("mpc", "load", choice)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to load playlist: %w", err)
	}

	cmd = exec.Command("mpc", "play")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to play: %w", err)
	}

	cachePlaylist(cfg, choice)

	return nil
}

func selectSong(ctx commands.LauncherContext) error {
	cmd := exec.Command("mpc", "playlist", "-f", "%position% - %artist% - %title%")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get playlist: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var songs []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			songs = append(songs, line)
		}
	}

	if len(songs) == 0 {
		return fmt.Errorf("playlist is empty")
	}

	songs = append([]string{"← Back"}, songs...)

	choice, err := ctx.Show(songs, "Select Song")
	if err != nil {
		return fmt.Errorf("cancelled")
	}

	if choice == "← Back" {
		return fmt.Errorf("cancelled")
	}

	var position int
	fmt.Sscanf(choice, "%d", &position)

	cmd = exec.Command("mpc", "play", fmt.Sprintf("%d", position))
	return cmd.Run()
}

func showCurrent() error {
	cmd := exec.Command("mpc", "current", "-f", "%artist% - %title%")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get current song: %w", err)
	}

	current := strings.TrimSpace(string(output))
	if current == "" {
		current = "Nothing playing"
	}

	notify("Now Playing", current)

	return nil
}

func cachePlaylist(cfg *Config, playlist string) {
	cachePath := cfg.CurrentPlaylistCache
	if len(cachePath) >= 2 && cachePath[:2] == "~/" {
		cachePath = filepath.Join(os.Getenv("HOME"), cachePath[2:])
	}

	cacheDir := filepath.Dir(cachePath)
	os.MkdirAll(cacheDir, 0755)

	os.WriteFile(cachePath, []byte(playlist), 0644)
}

func notify(title, message string) {
	if _, err := exec.LookPath("notify-send"); err == nil {
		exec.Command("notify-send", "ql mpc", fmt.Sprintf("%s\n%s", title, message)).Run()
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

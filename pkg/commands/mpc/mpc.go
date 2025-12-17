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

func Run(ctx commands.LauncherContext) error {
	// Извличаме config директно
	cfgInterface := ctx.Config().GetMpcConfig()
	if cfgInterface == nil {
		return fmt.Errorf("mpc config not found")
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
		return fmt.Errorf("mpc module is disabled in config")
	}

	// Check if mpc is installed
	if _, err := exec.LookPath("mpc"); err != nil {
		return fmt.Errorf("mpc is not installed")
	}

	// Menu options
	options := []string{
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
		return err
	}

	switch choice {
	case "Play/Pause":
		return togglePlayPause()
	case "Next":
		return next()
	case "Previous":
		return previous()
	case "Stop":
		return stop()
	case "Select Playlist":
		return selectPlaylist(ctx, &cfg)
	case "Select Song":
		return selectSong(ctx, &cfg)
	case "Show Current":
		return showCurrent()
	default:
		return fmt.Errorf("unknown choice: %s", choice)
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
	// Get playlists
	cmd := exec.Command("mpc", "lsplaylists")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get playlists: %w", err)
	}

	// Parse playlists
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

	// Show playlist menu
	choice, err := ctx.Show(playlists, "Select Playlist")
	if err != nil {
		return err
	}

	// Load and play playlist
	cmd = exec.Command("mpc", "clear")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clear playlist:  %w", err)
	}

	cmd = exec.Command("mpc", "load", choice)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to load playlist: %w", err)
	}

	cmd = exec.Command("mpc", "play")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to play:  %w", err)
	}

	// Cache current playlist
	cachePlaylist(cfg, choice)

	return nil
}

func selectSong(ctx commands.LauncherContext, cfg *Config) error {
	// Get current playlist
	cmd := exec.Command("mpc", "playlist", "-f", "%position% - %artist% - %title%")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get playlist: %w", err)
	}

	// Parse songs
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

	// Show song menu
	choice, err := ctx.Show(songs, "Select Song")
	if err != nil {
		return err
	}

	// Extract position (first number)
	var position int
	fmt.Sscanf(choice, "%d", &position)

	// Play song
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

	// Create cache directory
	cacheDir := filepath.Dir(cachePath)
	os.MkdirAll(cacheDir, 0755)

	// Write playlist name
	os.WriteFile(cachePath, []byte(playlist), 0644)
}

func notify(title, message string) {
	if _, err := exec.LookPath("notify-send"); err == nil {
		exec.Command("notify-send", "ql mpc", fmt.Sprintf("%s\n%s", title, message)).Run()
	}
}

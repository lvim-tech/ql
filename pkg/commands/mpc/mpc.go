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

var mpcPath string

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

	// Check if mpc exists in PATH
	mpcPath, err = exec.LookPath("mpc")
	if err != nil {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("mpc is not installed or not in PATH"),
		}
	}

	// Setup MPD connection from config
	if err := setupMpdConnection(&cfg); err != nil {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("MPD setup failed: %w", err),
		}
	}

	// Test connection
	testCmd := exec.Command(mpcPath, "status")
	output, err := testCmd.CombinedOutput()
	if err != nil {
		errMsg := strings.TrimSpace(string(output))
		if errMsg == "" {
			errMsg = err.Error()
		}
		return commands.CommandResult{
			Success: false,
			Error: fmt.Errorf("MPD connection failed: %s\n\nConnection:  %s\nMPD_HOST: %s",
				errMsg,
				cfg.ConnectionType,
				os.Getenv("MPD_HOST")),
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

func setupMpdConnection(cfg *Config) error {
	// Set XDG_RUNTIME_DIR if not set
	if os.Getenv("XDG_RUNTIME_DIR") == "" {
		uid := os.Getuid()
		os.Setenv("XDG_RUNTIME_DIR", fmt.Sprintf("/run/user/%d", uid))
	}

	// Setup based on connection type
	switch strings.ToLower(cfg.ConnectionType) {
	case "socket":
		socketPath := cfg.Socket

		// Expand tilde
		if strings.HasPrefix(socketPath, "~/") {
			home := os.Getenv("HOME")
			socketPath = filepath.Join(home, socketPath[2:])
		}

		// Check if socket exists
		if _, err := os.Stat(socketPath); err != nil {
			return fmt.Errorf("socket not found: %s", socketPath)
		}

		os.Setenv("MPD_HOST", socketPath)

	case "tcp":
		if cfg.Host == "" {
			return fmt.Errorf("host not specified in config")
		}

		mpdHost := cfg.Host

		// Add password if provided
		if cfg.Password != "" {
			mpdHost = cfg.Password + "@" + mpdHost
		}

		os.Setenv("MPD_HOST", mpdHost)

		if cfg.Port != "" {
			os.Setenv("MPD_PORT", cfg.Port)
		} else {
			os.Setenv("MPD_PORT", "6600")
		}

	default:
		return fmt.Errorf("invalid connection_type: %s (must be 'tcp' or 'socket')", cfg.ConnectionType)
	}

	return nil
}

func togglePlayPause() error {
	cmd := exec.Command(mpcPath, "toggle")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("toggle failed: %s", strings.TrimSpace(string(output)))
	}

	// Get current status and show notification
	statusCmd := exec.Command(mpcPath, "status")
	statusOutput, _ := statusCmd.Output()
	statusLines := strings.Split(string(statusOutput), "\n")

	if len(statusLines) > 1 {
		// Check if playing or paused
		if strings.Contains(statusLines[1], "[playing]") {
			notify("MPC", "Playing")
		} else if strings.Contains(statusLines[1], "[paused]") {
			notify("MPC", "Paused")
		}
	}

	return nil
}

func next() error {
	cmd := exec.Command(mpcPath, "next")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("next failed: %s", strings.TrimSpace(string(output)))
	}

	// Show current song after next
	currentCmd := exec.Command(mpcPath, "current", "-f", "%artist% - %title%")
	currentOutput, _ := currentCmd.Output()
	current := strings.TrimSpace(string(currentOutput))

	if current != "" {
		notify("MPC - Next", current)
	}

	return nil
}

func previous() error {
	cmd := exec.Command(mpcPath, "prev")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("prev failed: %s", strings.TrimSpace(string(output)))
	}

	// Show current song after previous
	currentCmd := exec.Command(mpcPath, "current", "-f", "%artist% - %title%")
	currentOutput, _ := currentCmd.Output()
	current := strings.TrimSpace(string(currentOutput))

	if current != "" {
		notify("MPC - Previous", current)
	}

	return nil
}

func stop() error {
	cmd := exec.Command(mpcPath, "stop")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("stop failed: %s", strings.TrimSpace(string(output)))
	}

	notify("MPC", "Stopped")

	return nil
}

func selectPlaylist(ctx commands.LauncherContext, cfg *Config) error {
	cmd := exec.Command(mpcPath, "lsplaylists")
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

	cmd = exec.Command(mpcPath, "clear")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clear playlist:  %w", err)
	}

	cmd = exec.Command(mpcPath, "load", choice)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to load playlist: %w", err)
	}

	cmd = exec.Command(mpcPath, "play")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to play:  %w", err)
	}

	cachePlaylist(cfg, choice)

	// Show notification
	notify("MPC - Playlist Loaded", choice)

	return nil
}

func selectSong(ctx commands.LauncherContext) error {
	cmd := exec.Command(mpcPath, "playlist", "-f", "%position% - %artist% - %title%")
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

	cmd = exec.Command(mpcPath, "play", fmt.Sprintf("%d", position))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to play song: %w", err)
	}

	// Show notification with song info
	currentCmd := exec.Command(mpcPath, "current", "-f", "%artist% - %title%")
	currentOutput, _ := currentCmd.Output()
	current := strings.TrimSpace(string(currentOutput))

	if current != "" {
		notify("Now Playing", current)
	}

	return nil
}

func showCurrent() error {
	cmd := exec.Command(mpcPath, "current", "-f", "%artist% - %title%")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get current song:  %w", err)
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
	exec.Command("notify-send", "ql mpc", fmt.Sprintf("%s\n%s", title, message)).Run()
}

func showErrorNotification(title, message string) {
	cmd := exec.Command("notify-send",
		"-u", "critical",
		"-t", "10000",
		title,
		message)
	cmd.Env = os.Environ()
	cmd.Start()
}

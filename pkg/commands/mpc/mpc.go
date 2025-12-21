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
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/utils"
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

func runMpcCommand(args ...string) *exec.Cmd {
	cmd := exec.Command(mpcPath, args...)
	cmd.Env = os.Environ()
	return cmd
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

	if !utils.CommandExists("mpc") {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("mpc is not installed or not in PATH"),
		}
	}

	mpcPath, _ = exec.LookPath("mpc")

	notifCfg := ctx.Config().GetNotificationConfig()

	if err := setupMpdConnection(&cfg); err != nil {
		utils.ShowErrorNotificationWithConfig(&notifCfg, "MPC Setup Error",
			fmt.Sprintf("MPD setup failed: %v", err))
		return commands.CommandResult{
			Success: false,
			Error:   commands.ErrBack,
		}
	}

	testCmd := runMpcCommand("status")
	output, err := testCmd.CombinedOutput()
	if err != nil {
		errMsg := strings.TrimSpace(string(output))
		if errMsg == "" {
			errMsg = err.Error()
		}
		utils.ShowErrorNotificationWithConfig(&notifCfg, "MPC Connection Error",
			fmt.Sprintf("MPD connection failed: %s\n\nConnection:     %s\nMPD_HOST: %s",
				errMsg,
				cfg.ConnectionType,
				os.Getenv("MPD_HOST")))
		return commands.CommandResult{
			Success: false,
			Error:   commands.ErrBack,
		}
	}

	// Check for direct command
	args := ctx.Args()
	if len(args) > 0 {
		return executeDirectCommand(ctx, args, &cfg, &notifCfg)
	}

	for {
		var options []string

		if !ctx.IsDirectLaunch() {
			options = append(options, "← Back")
		}

		options = append(options,
			"Play/Pause",
			"Next",
			"Previous",
			"Stop",
			"Select Playlist",
			"Select Song",
			"Show Current",
		)

		choice, err := ctx.Show(options, "MPC")
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
		case "Play/Pause":
			actionErr = togglePlayPause(&notifCfg)
		case "Next":
			actionErr = next(&notifCfg)
		case "Previous":
			actionErr = previous(&notifCfg)
		case "Stop":
			actionErr = stop(&notifCfg)
		case "Select Playlist":
			actionErr = selectPlaylist(ctx, &cfg, &notifCfg)
		case "Select Song":
			actionErr = selectSong(ctx, &notifCfg)
		case "Show Current":
			actionErr = showCurrent(&notifCfg)
		default:
			utils.ShowErrorNotificationWithConfig(&notifCfg, "MPC Error", fmt.Sprintf("Unknown choice: %s", choice))
			continue
		}

		if actionErr != nil {
			// If error is "cancelled" - it's ESC from submenu, exit completely
			if actionErr.Error() == "cancelled" {
				return commands.CommandResult{Success: false}
			}
			// Other error - show and loop back
			utils.ShowErrorNotificationWithConfig(&notifCfg, "MPC Error", actionErr.Error())
			continue
		}

		// Action succeeded - exit
		return commands.CommandResult{Success: true}
	}
}

func executeDirectCommand(ctx commands.LauncherContext, args []string, cfg *Config, notifCfg *config.NotificationConfig) commands.CommandResult {
	action := strings.ToLower(args[0])

	var err error

	switch action {
	case "toggle", "play", "pause":
		err = togglePlayPause(notifCfg)

	case "next":
		err = next(notifCfg)

	case "prev", "previous":
		err = previous(notifCfg)

	case "stop":
		err = stop(notifCfg)

	case "current", "status":
		err = showCurrent(notifCfg)

	case "playlist":
		// If playlist name is provided, load it directly
		if len(args) > 1 {
			playlistName := strings.Join(args[1:], " ")
			err = loadPlaylistDirect(playlistName, cfg, notifCfg)
		} else {
			// Otherwise show playlist selection menu
			err = selectPlaylist(ctx, cfg, notifCfg)
		}

	case "song":
		err = selectSong(ctx, notifCfg)

	default:
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("unknown mpc action: %s (use:  toggle, next, prev, stop, current, playlist, song)", action),
		}
	}

	if err != nil {
		return commands.CommandResult{Success: false, Error: err}
	}

	return commands.CommandResult{Success: true}
}

func loadPlaylistDirect(playlistName string, cfg *Config, notifCfg *config.NotificationConfig) error {
	// Clear current playlist
	cmd := runMpcCommand("clear")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clear playlist: %w", err)
	}

	// Load the playlist
	cmd = runMpcCommand("load", playlistName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to load playlist '%s': %w", playlistName, err)
	}

	// Start playing
	cmd = runMpcCommand("play")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to play: %w", err)
	}

	cachePlaylist(cfg, playlistName)
	utils.NotifyWithConfig(notifCfg, "MPC - Playlist Loaded", playlistName)

	return nil
}

func setupMpdConnection(cfg *Config) error {
	if os.Getenv("XDG_RUNTIME_DIR") == "" {
		uid := os.Getuid()
		os.Setenv("XDG_RUNTIME_DIR", fmt.Sprintf("/run/user/%d", uid))
	}

	switch strings.ToLower(cfg.ConnectionType) {
	case "socket":
		socketPath := utils.ExpandHomeDir(cfg.Socket)

		if !utils.FileExists(socketPath) {
			return fmt.Errorf("socket not found: %s", socketPath)
		}

		os.Setenv("MPD_HOST", socketPath)

	case "tcp":
		if cfg.Host == "" {
			return fmt.Errorf("host not specified in config")
		}

		mpdHost := cfg.Host

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

func togglePlayPause(notifCfg *config.NotificationConfig) error {
	cmd := runMpcCommand("toggle")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("toggle failed: %s", strings.TrimSpace(string(output)))
	}

	statusCmd := runMpcCommand("status")
	statusOutput, _ := statusCmd.Output()
	statusLines := strings.Split(string(statusOutput), "\n")

	if len(statusLines) > 1 {
		if strings.Contains(statusLines[1], "[playing]") {
			utils.NotifyWithConfig(notifCfg, "MPC", "Playing")
		} else if strings.Contains(statusLines[1], "[paused]") {
			utils.NotifyWithConfig(notifCfg, "MPC", "Paused")
		}
	}

	return nil
}

func next(notifCfg *config.NotificationConfig) error {
	cmd := runMpcCommand("next")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("next failed: %s", strings.TrimSpace(string(output)))
	}

	currentCmd := runMpcCommand("current", "-f", "%artist% - %title%")
	currentOutput, _ := currentCmd.Output()
	current := strings.TrimSpace(string(currentOutput))

	if current != "" {
		utils.NotifyWithConfig(notifCfg, "MPC - Next", current)
	}

	return nil
}

func previous(notifCfg *config.NotificationConfig) error {
	cmd := runMpcCommand("prev")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("prev failed: %s", strings.TrimSpace(string(output)))
	}

	currentCmd := runMpcCommand("current", "-f", "%artist% - %title%")
	currentOutput, _ := currentCmd.Output()
	current := strings.TrimSpace(string(currentOutput))

	if current != "" {
		utils.NotifyWithConfig(notifCfg, "MPC - Previous", current)
	}

	return nil
}

func stop(notifCfg *config.NotificationConfig) error {
	cmd := runMpcCommand("stop")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("stop failed: %s", strings.TrimSpace(string(output)))
	}

	utils.NotifyWithConfig(notifCfg, "MPC", "Stopped")

	return nil
}

func selectPlaylist(ctx commands.LauncherContext, cfg *Config, notifCfg *config.NotificationConfig) error {
	cmd := runMpcCommand("lsplaylists")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get playlists: %w", err)
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
		return fmt.Errorf("no saved playlists found.    Use 'mpc save <name>' to create one")
	}

	playlists = append([]string{"← Back"}, playlists...)

	choice, err := ctx.Show(playlists, "Select Playlist")
	if err != nil {
		// ESC pressed - return "cancelled" to exit completely
		return fmt.Errorf("cancelled")
	}

	if choice == "← Back" {
		// Back pressed - return "cancelled" to loop back
		return fmt.Errorf("cancelled")
	}

	return loadPlaylistDirect(choice, cfg, notifCfg)
}

func selectSong(ctx commands.LauncherContext, notifCfg *config.NotificationConfig) error {
	cmd := runMpcCommand("playlist", "-f", "%position% - %artist% - %title%")
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
		// ESC pressed - return "cancelled" to exit completely
		return fmt.Errorf("cancelled")
	}

	if choice == "← Back" {
		// Back pressed - return "cancelled" to loop back
		return fmt.Errorf("cancelled")
	}

	var position int
	fmt.Sscanf(choice, "%d", &position)

	cmd = runMpcCommand("play", fmt.Sprintf("%d", position))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to play song: %w", err)
	}

	currentCmd := runMpcCommand("current", "-f", "%artist% - %title%")
	currentOutput, _ := currentCmd.Output()
	current := strings.TrimSpace(string(currentOutput))

	if current != "" {
		utils.NotifyWithConfig(notifCfg, "Now Playing", current)
	}

	return nil
}

func showCurrent(notifCfg *config.NotificationConfig) error {
	cmd := runMpcCommand("current", "-f", "%artist% - %title%")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get current song:    %w", err)
	}

	current := strings.TrimSpace(string(output))
	if current == "" {
		current = "Nothing playing"
	}

	utils.NotifyWithConfig(notifCfg, "Now Playing", current)

	return nil
}

func cachePlaylist(cfg *Config, playlist string) {
	cachePath := utils.ExpandHomeDir(cfg.CurrentPlaylistCache)
	cacheDir := filepath.Dir(cachePath)

	utils.EnsureDir(cacheDir)
	os.WriteFile(cachePath, []byte(playlist), 0644)
}

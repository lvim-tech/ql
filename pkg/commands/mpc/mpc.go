// Package mpc provides MPD/MPC music player control functionality for ql.
// It supports playing music, managing playlists, and controlling playback via mpc commands.
package mpc

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/launcher"
)

func init() {
	commands.Register(commands.Command{
		Name:        "mpc",
		Description: "MPD music player control",
		Run:         Run,
	})
}

func Run(ctx *launcher.Context) error {
	cfg := config.Get()
	mpcCfg := cfg.Commands.Mpc

	// Провери дали е enabled
	if !mpcCfg.Enabled {
		return fmt.Errorf("mpc module is disabled in config")
	}

	// Провери дали има mpc
	if _, err := exec.LookPath("mpc"); err != nil {
		return fmt.Errorf("mpc is not installed")
	}

	// Меню опции
	options := []string{
		"Play",
		"Toggle",
		"Next",
		"Previous",
		"Queue Playlist",
		"Current Playlist",
		"Stop",
	}

	choice, err := ctx.Show(options, "MPC Control")
	if err != nil {
		return err
	}

	// Обработи избора
	switch choice {
	case "Play":
		return playSong(ctx)
	case "Toggle":
		return mpcCommand("toggle")
	case "Next":
		return mpcCommand("next")
	case "Previous":
		return mpcCommand("prev")
	case "Queue Playlist":
		return queuePlaylist(ctx, &mpcCfg)
	case "Current Playlist":
		return playFromCurrentPlaylist(ctx, &mpcCfg)
	case "Stop":
		return mpcCommand("stop")
	default:
		return fmt.Errorf("unknown choice: %s", choice)
	}
}

// playSong показва всички песни и пуска избраната
func playSong(ctx *launcher.Context) error {
	// Вземи всички песни
	cmd := exec.Command("mpc", "listall")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list music: %w", err)
	}

	songs := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(songs) == 0 {
		return fmt.Errorf("no music found")
	}

	// Покажи меню
	song, err := ctx.Show(songs, "Music to play")
	if err != nil {
		return err
	}

	// Clear queue и добави песента
	_ = mpcCommand("crop")

	if err := mpcCommandWithArgs("add", song); err != nil {
		return fmt.Errorf("failed to add song: %w", err)
	}

	// Изтрий предишната песен (position 0) ако има
	mpcCommandWithArgs("del", "0")

	// Play и repeat on
	if err := mpcCommand("play"); err != nil {
		return err
	}

	return mpcCommand("repeat", "on")
}

// queuePlaylist зарежда playlist
func queuePlaylist(ctx *launcher.Context, cfg *config.MpcConfig) error {
	// Вземи списък с playlists
	cmd := exec.Command("mpc", "lsplaylists")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list playlists: %w", err)
	}

	playlists := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(playlists) == 0 {
		return fmt.Errorf("no playlists found")
	}

	// Покажи меню
	playlist, err := ctx.Show(playlists, "Your playlists")
	if err != nil {
		return err
	}

	// Clear queue
	if err := mpcCommand("clear"); err != nil {
		return err
	}

	// Load playlist
	if err := mpcCommandWithArgs("load", playlist); err != nil {
		return fmt.Errorf("failed to load playlist: %w", err)
	}

	// Play first song
	if err := mpcCommandWithArgs("play", "1"); err != nil {
		return err
	}

	// Запази текущ playlist в cache
	cacheFile := cfg.CurrentPlaylistCache
	if cacheFile == "" {
		cacheFile = filepath.Join(os.Getenv("HOME"), ".cache", "ql_current_playlist")
	}

	// Expand ~ to home directory
	if strings.HasPrefix(cacheFile, "~/") {
		cacheFile = filepath.Join(os.Getenv("HOME"), cacheFile[2:])
	}

	// Създай директорията ако не съществува
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err == nil {
		os.WriteFile(cacheFile, []byte(playlist), 0644)
	}

	return nil
}

// playFromCurrentPlaylist пуска песен от текущ playlist
func playFromCurrentPlaylist(ctx *launcher.Context, cfg *config.MpcConfig) error {
	// Прочети текущ playlist от cache
	cacheFile := cfg.CurrentPlaylistCache
	if cacheFile == "" {
		cacheFile = filepath.Join(os.Getenv("HOME"), ".cache", "ql_current_playlist")
	}

	// Expand ~ to home directory
	if strings.HasPrefix(cacheFile, "~/") {
		cacheFile = filepath.Join(os.Getenv("HOME"), cacheFile[2:])
	}

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return fmt.Errorf("no current playlist cached, queue a playlist first")
	}

	playlist := strings.TrimSpace(string(data))

	// Вземи песните от playlist
	cmd := exec.Command("mpc", "playlist", playlist)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get playlist: %w", err)
	}

	songs := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(songs) == 0 {
		return fmt.Errorf("playlist is empty")
	}

	// Покажи меню
	song, err := ctx.Show(songs, "Music to play")
	if err != nil {
		return err
	}

	// Намери номера на песента
	songNumber := findSongNumber(songs, song)
	if songNumber == -1 {
		return fmt.Errorf("song not found in playlist")
	}

	// Clear queue и зареди playlist
	if err := mpcCommand("clear"); err != nil {
		return err
	}

	if err := mpcCommandWithArgs("load", playlist); err != nil {
		return err
	}

	// Play избраната песен
	return mpcCommandWithArgs("play", strconv.Itoa(songNumber))
}

// mpcCommand изпълнява mpc команда без аргументи
func mpcCommand(args ...string) error {
	cmd := exec.Command("mpc", args...)
	cmd.Stdout = nil // Suppress output
	cmd.Stderr = nil
	return cmd.Run()
}

// mpcCommandWithArgs изпълнява mpc команда с аргументи
func mpcCommandWithArgs(args ...string) error {
	cmd := exec.Command("mpc", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// findSongNumber намира номера на песен в списък
func findSongNumber(songs []string, target string) int {
	for i, song := range songs {
		if song == target {
			return i + 1 // mpc номерира от 1
		}
	}
	return -1
}

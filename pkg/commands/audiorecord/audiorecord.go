// Package audiorecord provides audio recording functionality for ql.
// It uses ffmpeg for recording and supports PulseAudio/PipeWire.
package audiorecord

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/launcher"
)

func init() {
	commands.Register(commands.Command{
		Name:        "audiorecord",
		Description: "Record audio",
		Run:         Run,
	})
}

func Run(ctx *launcher.Context) error {
	cfg := config.Get()
	audioCfg := cfg.Commands.AudioRecord

	// Провери дали е enabled
	if !audioCfg.Enabled {
		return fmt.Errorf("audiorecord module is disabled in config")
	}

	// Провери дали има ffmpeg
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg is not installed")
	}

	// Меню опции
	options := []string{
		"Start Recording",
		"Stop Recording",
	}

	choice, err := ctx.Show(options, "Audio Record")
	if err != nil {
		return err
	}

	switch choice {
	case "Start Recording":
		return startRecording(&audioCfg)
	case "Stop Recording":
		return stopRecording()
	default:
		return fmt.Errorf("unknown choice: %s", choice)
	}
}

// startRecording започва audio запис
func startRecording(cfg *config.AudioRecordConfig) error {
	// Expand save dir
	saveDir := cfg.SaveDir
	if saveDir[:2] == "~/" {
		saveDir = filepath.Join(os.Getenv("HOME"), saveDir[2:])
	}

	// Създай директорията ако не съществува
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return fmt.Errorf("failed to create save directory: %w", err)
	}

	// Генерирай име на файл
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s_%s.%s", cfg.FilePrefix, timestamp, cfg.Format)
	filepath := filepath.Join(saveDir, filename)

	// Опитай PulseAudio/PipeWire (работи и на X11 и на Wayland)
	audioDevice := detectAudioDevice()
	if audioDevice == "" {
		return fmt.Errorf("no audio device found (PulseAudio/PipeWire required)")
	}

	// Запиши PID в temp файл за спиране
	pidFile := "/tmp/ql_audiorecord. pid"

	// Стартирай ffmpeg в background
	cmd := exec.Command("ffmpeg",
		"-f", audioDevice,
		"-i", "default",
		"-c:a", getAudioCodec(cfg.Format),
		"-q:a", cfg.Quality,
		filepath,
	)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start recording: %w", err)
	}

	// Запиши PID
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0644); err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	notify("Audio recording started", filename)

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		stopRecording()
	}()

	return nil
}

// stopRecording спира audio запис
func stopRecording() error {
	pidFile := "/tmp/ql_audiorecord.pid"

	data, err := os.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("no recording in progress")
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return fmt.Errorf("invalid PID file")
	}

	// Изпрати SIGINT за graceful shutdown
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("recording process not found")
	}

	if err := process.Signal(syscall.SIGINT); err != nil {
		return fmt.Errorf("failed to stop recording: %w", err)
	}

	// Изтрий PID file
	os.Remove(pidFile)

	notify("Audio recording stopped", "")

	return nil
}

// detectAudioDevice открива audio device (pulse or pipewire)
func detectAudioDevice() string {
	// Провери за PipeWire
	if _, err := exec.LookPath("pw-cli"); err == nil {
		return "pulse" // PipeWire има pulse compatibility
	}

	// Провери за PulseAudio
	if _, err := exec.LookPath("pactl"); err == nil {
		return "pulse"
	}

	return ""
}

// getAudioCodec връща codec за format
func getAudioCodec(format string) string {
	switch format {
	case "mp3":
		return "libmp3lame"
	case "ogg":
		return "libvorbis"
	case "flac":
		return "flac"
	case "wav":
		return "pcm_s16le"
	default:
		return "libmp3lame"
	}
}

// notify изпраща notification
func notify(title, message string) {
	if _, err := exec.LookPath("notify-send"); err == nil {
		exec.Command("notify-send", "ql audiorecord", fmt.Sprintf("%s\n%s", title, message)).Run()
	}
}

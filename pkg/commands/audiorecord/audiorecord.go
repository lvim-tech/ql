// Package audiorecord provides audio recording functionality for ql.
// It uses ffmpeg for recording and supports PulseAudio/PipeWire.
package audiorecord

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/mitchellh/mapstructure"
)

func init() {
	commands.Register(commands.Command{
		Name:        "audiorecord",
		Description: "Record audio from microphone",
		Run:         Run,
	})
}

func Run(ctx commands.LauncherContext) error {
	// Извличаме config директно
	cfgInterface := ctx.Config().GetAudioRecordConfig()
	if cfgInterface == nil {
		return fmt.Errorf("audiorecord config not found")
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
		return fmt.Errorf("audiorecord module is disabled in config")
	}

	// Menu options
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
		return startRecording(&cfg)
	case "Stop Recording":
		return stopRecording()
	default:
		return fmt.Errorf("unknown choice: %s", choice)
	}
}

func startRecording(cfg *Config) error {
	// Check if already recording
	if isRecording() {
		return fmt.Errorf("recording already in progress")
	}

	// Check if ffmpeg is installed
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg is not installed")
	}

	// Expand save dir
	saveDir := cfg.SaveDir
	if len(saveDir) >= 2 && saveDir[:2] == "~/" {
		saveDir = filepath.Join(os.Getenv("HOME"), saveDir[2:])
	}

	// Create directory if not exists
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return fmt.Errorf("failed to create save directory: %w", err)
	}

	// Generate unique filename
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s_%s.%s", cfg.FilePrefix, timestamp, cfg.Format)
	outputPath := filepath.Join(saveDir, filename)

	// If file exists, add milliseconds to make it unique
	if _, err := os.Stat(outputPath); err == nil {
		timestamp = time.Now().Format("2006-01-02_15-04-05.000")
		filename = fmt.Sprintf("%s_%s.%s", cfg.FilePrefix, timestamp, cfg.Format)
		outputPath = filepath.Join(saveDir, filename)
	}

	// Build ffmpeg command
	args := []string{
		"-f", "pulse",
		"-i", "default",
		"-q:a", cfg.Quality,
		"-n",
		outputPath,
	}

	cmd := exec.Command("ffmpeg", args...)

	// Redirect stderr/stdout to /dev/null
	devNull, err := os.Open(os.DevNull)
	if err == nil {
		cmd.Stderr = devNull
		cmd.Stdout = devNull
		defer devNull.Close()
	}

	// Start recording
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start recording: %w", err)
	}

	// Save PID immediately after start
	pidFile := getPIDFile()
	pidBytes := []byte(strconv.Itoa(cmd.Process.Pid))
	if err := os.WriteFile(pidFile, pidBytes, 0644); err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("failed to save PID: %w", err)
	}

	// Save output path
	pathFile := getOutputPathFile()
	if err := os.WriteFile(pathFile, []byte(outputPath), 0644); err != nil {
		cmd.Process.Kill()
		os.Remove(pidFile)
		return fmt.Errorf("failed to save output path: %w", err)
	}

	// Monitor the process in a goroutine
	go func() {
		cmd.Wait()
		// Clean up PID files when process exits
		os.Remove(pidFile)
		os.Remove(pathFile)
	}()

	// Give the process a moment to actually start
	time.Sleep(500 * time.Millisecond)

	// Verify it's still running
	if !isRecording() {
		return fmt.Errorf("recording process failed to start")
	}

	// Notify
	notify("Recording Started", filename)

	return nil
}

func stopRecording() error {
	if !isRecording() {
		return fmt.Errorf("no recording in progress")
	}

	pidFile := getPIDFile()
	pathFile := getOutputPathFile()

	// Read PID
	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		return fmt.Errorf("invalid PID:  %w", err)
	}

	// Read output path
	outputPath, err := os.ReadFile(pathFile)
	if err != nil {
		return fmt.Errorf("failed to read output path: %w", err)
	}

	// Send SIGINT to gracefully stop ffmpeg
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := process.Signal(syscall.SIGINT); err != nil {
		return fmt.Errorf("failed to stop recording: %w", err)
	}

	// Wait a bit for process to finish
	time.Sleep(1 * time.Second)

	// Clean up
	os.Remove(pidFile)
	os.Remove(pathFile)

	filename := filepath.Base(string(outputPath))

	// Notify
	notify("Recording Stopped", filename)

	return nil
}

// Helper functions

func isRecording() bool {
	pidFile := getPIDFile()
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		return false
	}

	// Check if process is actually running
	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		// Process doesn't exist - clean up stale PID file
		os.Remove(pidFile)
		os.Remove(getOutputPathFile())
		return false
	}

	return true
}

func getPIDFile() string {
	return filepath.Join(os.TempDir(), "ql_audiorecord. pid")
}

func getOutputPathFile() string {
	return filepath.Join(os.TempDir(), "ql_audiorecord_output.txt")
}

func notify(title, message string) {
	cmd := exec.Command("notify-send", title, message)
	cmd.Run()
}

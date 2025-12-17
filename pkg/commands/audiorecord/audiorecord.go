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

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfgInterface := ctx.Config().GetAudioRecordConfig()

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
			Error:   fmt.Errorf("audiorecord module is disabled in config"),
		}
	}

	for {
		options := []string{
			"← Back",
			"Start Recording",
			"Stop Recording",
		}

		choice, err := ctx.Show(options, "Audio Record")
		if err != nil {
			return commands.CommandResult{Success: false}
		}

		if choice == "← Back" {
			return commands.CommandResult{Success: false}
		}

		var actionErr error
		switch choice {
		case "Start Recording":
			actionErr = startRecording(&cfg)
		case "Stop Recording":
			actionErr = stopRecording()
		default:
			showErrorNotification("Audio Record Error", fmt.Sprintf("Unknown choice: %s", choice))
			continue
		}

		if actionErr != nil {
			showErrorNotification("Audio Record Error", actionErr.Error())
			continue
		}

		return commands.CommandResult{Success: true}
	}
}

func startRecording(cfg *Config) error {
	if isRecording() {
		return fmt.Errorf("recording already in progress")
	}

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg is not installed")
	}

	saveDir := cfg.SaveDir
	if len(saveDir) >= 2 && saveDir[:2] == "~/" {
		saveDir = filepath.Join(os.Getenv("HOME"), saveDir[2:])
	}

	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return fmt.Errorf("failed to create save directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s_%s.%s", cfg.FilePrefix, timestamp, cfg.Format)
	outputPath := filepath.Join(saveDir, filename)

	if _, err := os.Stat(outputPath); err == nil {
		timestamp = time.Now().Format("2006-01-02_15-04-05.000")
		filename = fmt.Sprintf("%s_%s.%s", cfg.FilePrefix, timestamp, cfg.Format)
		outputPath = filepath.Join(saveDir, filename)
	}

	args := []string{
		"-f", "pulse",
		"-i", "default",
		"-q: a", cfg.Quality,
		"-n",
		outputPath,
	}

	cmd := exec.Command("ffmpeg", args...)

	devNull, err := os.Open(os.DevNull)
	if err == nil {
		cmd.Stderr = devNull
		cmd.Stdout = devNull
		defer devNull.Close()
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start recording: %w", err)
	}

	pidFile := getPIDFile()
	pidBytes := []byte(strconv.Itoa(cmd.Process.Pid))
	if err := os.WriteFile(pidFile, pidBytes, 0644); err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("failed to save PID: %w", err)
	}

	pathFile := getOutputPathFile()
	if err := os.WriteFile(pathFile, []byte(outputPath), 0644); err != nil {
		cmd.Process.Kill()
		os.Remove(pidFile)
		return fmt.Errorf("failed to save output path: %w", err)
	}

	go func() {
		cmd.Wait()
		os.Remove(pidFile)
		os.Remove(pathFile)
	}()

	time.Sleep(500 * time.Millisecond)

	if !isRecording() {
		return fmt.Errorf("recording process failed to start")
	}

	notify("Recording Started", filename)

	return nil
}

func stopRecording() error {
	if !isRecording() {
		return fmt.Errorf("no recording in progress")
	}

	pidFile := getPIDFile()
	pathFile := getOutputPathFile()

	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		return fmt.Errorf("invalid PID:  %w", err)
	}

	outputPath, err := os.ReadFile(pathFile)
	if err != nil {
		return fmt.Errorf("failed to read output path:  %w", err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := process.Signal(syscall.SIGINT); err != nil {
		return fmt.Errorf("failed to stop recording: %w", err)
	}

	time.Sleep(1 * time.Second)

	os.Remove(pidFile)
	os.Remove(pathFile)

	filename := filepath.Base(string(outputPath))

	notify("Recording Stopped", filename)

	return nil
}

func isRecording() bool {
	pidFile := getPIDFile()
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		return false
	}

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

	err = process.Signal(syscall.Signal(0))
	if err != nil {
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
	if _, err := exec.LookPath("notify-send"); err == nil {
		exec.Command("notify-send", title, message).Run()
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

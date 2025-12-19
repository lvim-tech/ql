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
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/utils"
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

	notifCfg := ctx.Config().GetNotificationConfig()

	for {
		options := []string{
			"← Back",
			"Start Recording",
			"Stop Recording",
		}

		choice, err := ctx.Show(options, "Audio Record")
		if err != nil {
			// ESC pressed - exit completely
			return commands.CommandResult{Success: false}
		}

		if choice == "← Back" {
			// Back pressed - return to module menu
			return commands.CommandResult{
				Success: false,
				Error:   commands.ErrBack,
			}
		}

		var actionErr error
		switch choice {
		case "Start Recording":
			actionErr = startRecording(&cfg, &notifCfg)
		case "Stop Recording":
			actionErr = stopRecording(&notifCfg)
		default:
			utils.ShowErrorNotificationWithConfig(&notifCfg, "Audio Record Error", fmt.Sprintf("Unknown choice: %s", choice))
			continue
		}

		if actionErr != nil {
			// Error occurred - show notification and loop back to menu
			utils.ShowErrorNotificationWithConfig(&notifCfg, "Audio Record Error", actionErr.Error())
			continue
		}

		// Action succeeded - exit
		return commands.CommandResult{Success: true}
	}
}

func startRecording(cfg *Config, notifCfg *config.NotificationConfig) error {
	if isRecording() {
		return fmt.Errorf("recording already in progress")
	}

	if !utils.CommandExists("ffmpeg") {
		return fmt.Errorf("ffmpeg is not installed")
	}

	saveDir := utils.ExpandHomeDir(cfg.SaveDir)
	if err := utils.EnsureDir(saveDir); err != nil {
		return fmt.Errorf("failed to create save directory: %w", err)
	}

	timestamp := utils.GetTimestamp()
	filename := fmt.Sprintf("%s_%s.%s", cfg.FilePrefix, timestamp, cfg.Format)
	outputPath := filepath.Join(saveDir, filename)

	if utils.FileExists(outputPath) {
		timestamp = utils.GetTimestampMillis()
		filename = fmt.Sprintf("%s_%s.%s", cfg.FilePrefix, timestamp, cfg.Format)
		outputPath = filepath.Join(saveDir, filename)
	}

	args := []string{
		"-f", "pulse",
		"-i", "default",
		"-q: a", cfg.Quality,
		"-y",
		outputPath,
	}

	cmd := exec.Command("ffmpeg", args...)

	if utils.IsTerminal() && notifCfg.ShowInTerminal {
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
	} else {
		devNull, err := os.Open(os.DevNull)
		if err == nil {
			cmd.Stderr = devNull
			cmd.Stdout = devNull
			defer devNull.Close()
		}
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start recording: %w", err)
	}

	pidFile := getPIDFile()
	pidBytes := []byte(strconv.Itoa(cmd.Process.Pid))
	if err := os.WriteFile(pidFile, pidBytes, 0644); err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("failed to save PID:  %w", err)
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

	utils.NotifyWithConfig(notifCfg, "Recording Started", filename)

	return nil
}

func stopRecording(notifCfg *config.NotificationConfig) error {
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

	utils.NotifyWithConfig(notifCfg, "Recording Stopped", filename)

	return nil
}

func isRecording() bool {
	pidFile := getPIDFile()
	if !utils.FileExists(pidFile) {
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
	return "/tmp/ql_audiorecord. pid"
}

func getOutputPathFile() string {
	return "/tmp/ql_audiorecord_output.txt"
}

// Package videorecord provides screen recording functionality for ql.
// It supports both X11 (via ffmpeg) and Wayland (via wf-recorder).
package videorecord

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/mitchellh/mapstructure"
)

func init() {
	commands.Register(commands.Command{
		Name:        "videorecord",
		Description: "Record screen video",
		Run:         Run,
	})
}

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfgInterface := ctx.Config().GetVideoRecordConfig()

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
			Error:   fmt.Errorf("videorecord module is disabled in config"),
		}
	}

	for {
		options := []string{
			"← Back",
			"Start Recording",
			"Stop Recording",
		}

		choice, err := ctx.Show(options, "Video Record")
		if err != nil {
			return commands.CommandResult{Success: false}
		}

		if choice == "← Back" {
			return commands.CommandResult{Success: false}
		}

		var actionErr error
		switch choice {
		case "Start Recording":
			actionErr = startRecording(ctx, &cfg)
		case "Stop Recording":
			actionErr = stopRecording(&cfg)
		default:
			showErrorNotification("Video Record Error", fmt.Sprintf("Unknown choice: %s", choice))
			continue
		}

		if actionErr != nil {
			showErrorNotification("Video Record Error", actionErr.Error())
			continue
		}

		return commands.CommandResult{Success: true}
	}
}

func startRecording(ctx commands.LauncherContext, cfg *Config) error {
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

	isWayland := os.Getenv("WAYLAND_DISPLAY") != ""

	regionOptions := []string{
		"← Back",
		"Fullscreen",
		"Active Window",
		"Select Region",
	}

	regionChoice, err := ctx.Show(regionOptions, "Recording Region")
	if err != nil {
		return fmt.Errorf("cancelled")
	}

	if regionChoice == "← Back" {
		return fmt.Errorf("cancelled")
	}

	var cmd *exec.Cmd

	if isWayland {
		cmd, err = buildWaylandCommand(regionChoice, outputPath, cfg)
		if err != nil {
			return err
		}
	} else {
		cmd, err = buildX11Command(regionChoice, outputPath, cfg)
		if err != nil {
			return err
		}
	}

	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	pidFile := "/tmp/ql_videorecord.pid"

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start recording: %w", err)
	}

	pidData := fmt.Sprintf("%d\n%s", cmd.Process.Pid, outputPath)
	if err := os.WriteFile(pidFile, []byte(pidData), 0644); err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	if cfg.ShowNotify {
		notify("Video recording started", filename)
	}

	cmd.Process.Release()

	return nil
}

func buildWaylandCommand(region, outputPath string, cfg *Config) (*exec.Cmd, error) {
	if _, err := exec.LookPath("wf-recorder"); err != nil {
		return nil, fmt.Errorf("wf-recorder is not installed (required for Wayland)")
	}

	args := []string{
		"-f", outputPath,
		"-c", cfg.Wayland.VideoCodec,
		"-p", fmt.Sprintf("preset=%s", cfg.Wayland.Preset),
		"-p", fmt.Sprintf("crf=%s", cfg.Quality),
		"-r", fmt.Sprintf("%d", cfg.Wayland.Framerate),
	}

	if cfg.RecordAudio {
		args = append(args, "--audio")
		args = append(args, "-a", cfg.Wayland.AudioCodec)
	}

	switch region {
	case "Fullscreen":

	case "Active Window":
		windowGeometry, err := getWaylandActiveWindow()
		if err != nil {
			if cfg.ShowNotify {
				notify("Warning", "Active window not supported, using fullscreen")
			}
		} else {
			args = append(args, "-g", windowGeometry)
		}

	case "Select Region":
		if _, err := exec.LookPath("slurp"); err != nil {
			return nil, fmt.Errorf("slurp is not installed (required for region selection)")
		}

		slurpCmd := exec.Command("slurp")
		geometry, err := slurpCmd.Output()
		if err != nil {
			return nil, fmt.Errorf("region selection cancelled")
		}

		args = append(args, "-g", strings.TrimSpace(string(geometry)))
	}

	return exec.Command("wf-recorder", args...), nil
}

func buildX11Command(region, outputPath string, cfg *Config) (*exec.Cmd, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("ffmpeg is not installed")
	}

	args := []string{
		"-f", "x11grab",
		"-framerate", fmt.Sprintf("%d", cfg.X11.Framerate),
	}

	switch region {
	case "Fullscreen":
		resolution := getScreenResolution()
		args = append(args, "-video_size", resolution)
		args = append(args, "-i", ":0.0")

	case "Active Window":
		geometry, offset, err := getActiveWindowGeometry()
		if err != nil {
			return nil, fmt.Errorf("failed to get active window:  %w", err)
		}
		args = append(args, "-video_size", geometry)
		args = append(args, "-i", fmt.Sprintf(":0.0+%s", offset))

	case "Select Region":
		if _, err := exec.LookPath("slop"); err != nil {
			return nil, fmt.Errorf("slop is not installed (required for region selection)")
		}

		slopCmd := exec.Command("slop", "-f", "%wx%h %x,%y")
		geometry, err := slopCmd.Output()
		if err != nil {
			return nil, fmt.Errorf("region selection cancelled")
		}

		geometryStr := strings.TrimSpace(string(geometry))
		parts := strings.Fields(geometryStr)

		if len(parts) == 2 {
			args = append(args, "-video_size", parts[0])
			args = append(args, "-i", fmt.Sprintf(":0.0+%s", parts[1]))
		} else {
			return nil, fmt.Errorf("invalid geometry from slop")
		}
	}

	if cfg.RecordAudio {
		audioDevice := detectAudioDevice()
		if audioDevice != "" {
			args = append(args, "-f", audioDevice, "-i", "default")
		}
	}

	args = append(args,
		"-r", fmt.Sprintf("%d", cfg.X11.OutputFPS),
		"-c:v", cfg.X11.VideoCodec,
		"-crf", cfg.Quality,
		"-preset", cfg.X11.Preset,
	)

	if cfg.RecordAudio {
		args = append(args, "-c:a", cfg.X11.AudioCodec)
	}

	args = append(args, outputPath)

	return exec.Command("ffmpeg", args...), nil
}

func getWaylandActiveWindow() (string, error) {
	if _, err := exec.LookPath("swaymsg"); err == nil {
		cmd := exec.Command("swaymsg", "-t", "get_tree")
		output, err := cmd.Output()
		if err == nil {
			_ = output
		}
	}

	if _, err := exec.LookPath("hyprctl"); err == nil {
		cmd := exec.Command("hyprctl", "activewindow", "-j")
		output, err := cmd.Output()
		if err == nil {
			_ = output
		}
	}

	return "", fmt.Errorf("unable to get active window on Wayland")
}

func getActiveWindowGeometry() (string, string, error) {
	if _, err := exec.LookPath("xdotool"); err != nil {
		return "", "", fmt.Errorf("xdotool not installed")
	}

	cmd := exec.Command("xdotool", "getactivewindow")
	windowIDBytes, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("no active window")
	}

	windowID := strings.TrimSpace(string(windowIDBytes))

	cmd = exec.Command("xdotool", "getwindowgeometry", "--shell", windowID)
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to get window geometry")
	}

	lines := strings.Split(string(output), "\n")
	var width, height, x, y string

	for _, line := range lines {
		if val, found := strings.CutPrefix(line, "WIDTH="); found {
			width = val
		} else if val, found := strings.CutPrefix(line, "HEIGHT="); found {
			height = val
		} else if val, found := strings.CutPrefix(line, "X="); found {
			x = val
		} else if val, found := strings.CutPrefix(line, "Y="); found {
			y = val
		}
	}

	geometry := fmt.Sprintf("%sx%s", width, height)
	offset := fmt.Sprintf("%s,%s", x, y)

	return geometry, offset, nil
}

func stopRecording(cfg *Config) error {
	pidFile := "/tmp/ql_videorecord.pid"

	data, err := os.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("no recording in progress")
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) < 2 {
		return fmt.Errorf("invalid PID file")
	}

	var pid int
	if _, err := fmt.Sscanf(lines[0], "%d", &pid); err != nil {
		return fmt.Errorf("invalid PID file")
	}

	outputPath := strings.TrimSpace(lines[1])

	process, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(pidFile)
		return fmt.Errorf("recording process not found")
	}

	if err := process.Signal(syscall.SIGINT); err != nil {
		os.Remove(pidFile)
		return fmt.Errorf("failed to stop recording: %w", err)
	}

	time.Sleep(2 * time.Second)

	os.Remove(pidFile)

	if cfg.ShowNotify {
		notify("Video recording stopped", fmt.Sprintf("Saved to:\n%s", outputPath))
	}

	return nil
}

func getScreenResolution() string {
	cmd := exec.Command("xrandr")
	output, err := cmd.Output()
	if err != nil {
		return "1920x1080"
	}

	outputStr := string(output)
	startIdx := 0

	for {
		lineEnd := strings.IndexByte(outputStr[startIdx:], '\n')
		if lineEnd == -1 {
			break
		}

		line := outputStr[startIdx : startIdx+lineEnd]

		if strings.Contains(line, "*") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				return fields[0]
			}
		}

		startIdx += lineEnd + 1
	}

	return "1920x1080"
}

func detectAudioDevice() string {
	if _, err := exec.LookPath("pw-cli"); err == nil {
		return "pulse"
	}

	if _, err := exec.LookPath("pactl"); err == nil {
		return "pulse"
	}

	return ""
}

func notify(title, message string) {
	if _, err := exec.LookPath("notify-send"); err == nil {
		exec.Command("notify-send", "ql videorecord", fmt.Sprintf("%s\n%s", title, message)).Run()
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

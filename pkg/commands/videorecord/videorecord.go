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
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/launcher"
)

func init() {
	commands.Register(commands.Command{
		Name:        "videorecord",
		Description: "Record screen video",
		Run:         Run,
	})
}

func Run(ctx *launcher.Context) error {
	cfg := config.Get()
	videoCfg := cfg.Commands.VideoRecord

	// Провери дали е enabled
	if !videoCfg.Enabled {
		return fmt.Errorf("videorecord module is disabled in config")
	}

	// Меню опции
	options := []string{
		"Start Recording",
		"Stop Recording",
	}

	choice, err := ctx.Show(options, "Video Record")
	if err != nil {
		return err
	}

	switch choice {
	case "Start Recording":
		return startRecording(ctx, &videoCfg)
	case "Stop Recording":
		return stopRecording(&videoCfg)
	default:
		return fmt.Errorf("unknown choice: %s", choice)
	}
}

// startRecording започва video запис
func startRecording(ctx *launcher.Context, cfg *config.VideoRecordConfig) error {
	// Expand save dir
	saveDir := cfg.SaveDir
	if len(saveDir) >= 2 && saveDir[:2] == "~/" {
		saveDir = filepath.Join(os.Getenv("HOME"), saveDir[2:])
	}

	// Създай директорията ако не съществува
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return fmt.Errorf("failed to create save directory: %w", err)
	}

	// Генерирай име на файл
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s_%s.%s", cfg.FilePrefix, timestamp, cfg.Format)
	outputPath := filepath.Join(saveDir, filename)

	// Провери дали е Wayland или X11
	isWayland := os.Getenv("WAYLAND_DISPLAY") != ""

	// Меню за избор на region (общо за X11 и Wayland)
	regionOptions := []string{
		"Fullscreen",
		"Active Window",
		"Select Region",
	}

	regionChoice, err := ctx.Show(regionOptions, "Recording Region")
	if err != nil {
		return err
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

	// Detach процеса - no stdin, stdout, stderr
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Създай нов process group
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	// Запиши PID в temp файл
	pidFile := "/tmp/ql_videorecord. pid"

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start recording:  %w", err)
	}

	// Запиши PID и output path
	pidData := fmt.Sprintf("%d\n%s", cmd.Process.Pid, outputPath)
	if err := os.WriteFile(pidFile, []byte(pidData), 0644); err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	if cfg.ShowNotify {
		notify("Video recording started", filename)
	}

	// Release процеса - ql може да излезе, записът продължава
	cmd.Process.Release()

	return nil
}

// buildWaylandCommand build wf-recorder command
func buildWaylandCommand(region, outputPath string, cfg *config.VideoRecordConfig) (*exec.Cmd, error) {
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

	// Audio
	if cfg.RecordAudio {
		args = append(args, "--audio")
		args = append(args, "-a", cfg.Wayland.AudioCodec)
	}

	switch region {
	case "Fullscreen":
		// No extra args needed

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

// buildX11Command build ffmpeg command for X11
func buildX11Command(region, outputPath string, cfg *config.VideoRecordConfig) (*exec.Cmd, error) {
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

	// Audio
	if cfg.RecordAudio {
		audioDevice := detectAudioDevice()
		if audioDevice != "" {
			args = append(args, "-f", audioDevice, "-i", "default")
		}
	}

	// Video encoding - FROM CONFIG
	args = append(args,
		"-r", fmt.Sprintf("%d", cfg.X11.OutputFPS),
		"-c:v", cfg.X11.VideoCodec,
		"-crf", cfg.Quality,
		"-preset", cfg.X11.Preset,
	)

	// Audio codec
	if cfg.RecordAudio {
		args = append(args, "-c:a", cfg.X11.AudioCodec)
	}

	args = append(args, outputPath)

	return exec.Command("ffmpeg", args...), nil
}

// getWaylandActiveWindow tries to get active window geometry on Wayland
func getWaylandActiveWindow() (string, error) {
	// Try Sway
	if _, err := exec.LookPath("swaymsg"); err == nil {
		cmd := exec.Command("swaymsg", "-t", "get_tree")
		output, err := cmd.Output()
		if err == nil {
			// Parse focused window (simplified - needs proper JSON parsing)
			_ = output
			// For now, return error to fallback to fullscreen
		}
	}

	// Try Hyprland
	if _, err := exec.LookPath("hyprctl"); err == nil {
		cmd := exec.Command("hyprctl", "activewindow", "-j")
		output, err := cmd.Output()
		if err == nil {
			// Parse active window (simplified - needs proper JSON parsing)
			_ = output
			// For now, return error to fallback to fullscreen
		}
	}

	return "", fmt.Errorf("unable to get active window on Wayland")
}

// getActiveWindowGeometry get active window geometry for X11
func getActiveWindowGeometry() (string, string, error) {
	// Check if xdotool is installed
	if _, err := exec.LookPath("xdotool"); err != nil {
		return "", "", fmt.Errorf("xdotool not installed")
	}

	// Get active window ID
	cmd := exec.Command("xdotool", "getactivewindow")
	windowIDBytes, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("no active window")
	}

	windowID := strings.TrimSpace(string(windowIDBytes))

	// Get window geometry
	cmd = exec.Command("xdotool", "getwindowgeometry", "--shell", windowID)
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to get window geometry")
	}

	// Parse geometry (WIDTH=1920\nHEIGHT=1080\nX=0\nY=0)
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

// stopRecording спира video запис
func stopRecording(cfg *config.VideoRecordConfig) error {
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

	// Изпрати SIGINT за graceful shutdown
	process, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(pidFile)
		return fmt.Errorf("recording process not found")
	}

	if err := process.Signal(syscall.SIGINT); err != nil {
		// Process might already be dead
		os.Remove(pidFile)
		return fmt.Errorf("failed to stop recording: %w", err)
	}

	// Изчакай малко за finalization
	time.Sleep(2 * time.Second)

	// Изтрий PID file
	os.Remove(pidFile)

	if cfg.ShowNotify {
		notify("Video recording stopped", fmt.Sprintf("Saved to:\n%s", outputPath))
	}

	return nil
}

// getScreenResolution връща screen resolution за X11
func getScreenResolution() string {
	cmd := exec.Command("xrandr")
	output, err := cmd.Output()
	if err != nil {
		return "1920x1080" // fallback
	}

	// Parse primary display resolution
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "*") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				return fields[0]
			}
		}
	}

	return "1920x1080" // fallback
}

// detectAudioDevice discovers audio device (pulse or pipewire)
func detectAudioDevice() string {
	// Check for PipeWire
	if _, err := exec.LookPath("pw-cli"); err == nil {
		return "pulse" // PipeWire has pulse compatibility
	}

	// Check for PulseAudio
	if _, err := exec.LookPath("pactl"); err == nil {
		return "pulse"
	}

	return ""
}

// notify sends notification
func notify(title, message string) {
	if _, err := exec.LookPath("notify-send"); err == nil {
		exec.Command("notify-send", "ql videorecord", fmt.Sprintf("%s\n%s", title, message)).Run()
	}
}

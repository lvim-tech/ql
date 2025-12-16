// Package screenshot provides screenshot functionality for ql.
// It supports both X11 (maim) and Wayland (grim/slurp) with multiple capture modes.
package screenshot

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lvim-tech/ql/internal/utils"
	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/launcher"
)

func init() {
	commands.Register(commands.Command{
		Name:        "screenshot",
		Description: "Take screenshots",
		Run:         Run,
	})
}

// CaptureMode представлява тип screenshot
type CaptureMode string

const (
	CaptureModeFullscreen CaptureMode = "Fullscreen"
	CaptureModeWindow     CaptureMode = "Active window"
	CaptureModeRegion     CaptureMode = "Selected region"
	CaptureModeOutput     CaptureMode = "Current output"
)

// Destination представлява къде да се запази screenshot-а
type Destination string

const (
	DestinationFile      Destination = "File"
	DestinationClipboard Destination = "Clipboard"
	DestinationBoth      Destination = "Both"
)

// Config за screenshot
type Config struct {
	SaveDir    string
	FilePrefix string
}

// Run показва screenshot menu
func Run(ctx *launcher.Context) error {
	// Detect platform
	platform := detectPlatform()
	if platform == "" {
		return fmt.Errorf("no screenshot tool found (install maim for X11 or grim/slurp for Wayland)")
	}

	// Load config
	cfg := getScreenshotConfig()

	// Ensure screenshot directory exists
	if err := utils.EnsureDir(cfg.SaveDir); err != nil {
		return fmt.Errorf("failed to create screenshot directory: %w", err)
	}

	// Select capture mode
	modes := getCaptureModes(platform)
	mode, err := selectCaptureMode(ctx, modes)
	if err != nil {
		return err
	}

	// Select delay
	delay, err := selectDelay(ctx)
	if err != nil {
		return err
	}

	// Select destination
	dest, err := selectDestination(ctx)
	if err != nil {
		return err
	}

	// Generate filename
	filename := generateFilename(cfg, string(mode))

	// Take screenshot
	return takeScreenshot(platform, mode, delay, dest, filename)
}

// detectPlatform определя дали сме на X11 или Wayland
func detectPlatform() string {
	sessionType := utils.GetEnvOrDefault("XDG_SESSION_TYPE", "")

	switch sessionType {
	case "wayland":
		if utils.CommandExists("grim") && utils.CommandExists("slurp") {
			return "wayland"
		}
	case "x11":
		if utils.CommandExists("maim") {
			return "x11"
		}
	default:
		// Fallback detection
		if utils.CommandExists("grim") && utils.CommandExists("slurp") {
			return "wayland"
		}
		if utils.CommandExists("maim") {
			return "x11"
		}
	}

	return ""
}

// getCaptureModes връща достъпните режими за платформата
func getCaptureModes(platform string) []CaptureMode {
	modes := []CaptureMode{
		CaptureModeFullscreen,
		CaptureModeWindow,
		CaptureModeRegion,
	}

	// Wayland поддържа и Current output
	if platform == "wayland" {
		modes = append(modes, CaptureModeOutput)
	}

	return modes
}

// selectCaptureMode избира режим на screenshot
func selectCaptureMode(ctx *launcher.Context, modes []CaptureMode) (CaptureMode, error) {
	options := make([]string, len(modes))
	for i, mode := range modes {
		options[i] = string(mode)
	}

	choice, err := ctx.Show(options, "Take screenshot of:")
	if err != nil {
		return "", err
	}

	return CaptureMode(choice), nil
}

// selectDelay избира delay преди screenshot
func selectDelay(ctx *launcher.Context) (int, error) {
	options := []string{"0", "1", "2", "3", "4", "5"}
	choice, err := ctx.Show(options, "Delay (in seconds):")
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(choice)
}

// selectDestination избира къде да се запази screenshot-а
func selectDestination(ctx *launcher.Context) (Destination, error) {
	options := []string{
		string(DestinationFile),
		string(DestinationClipboard),
		string(DestinationBoth),
	}

	choice, err := ctx.Show(options, "Destination:")
	if err != nil {
		return "", err
	}

	return Destination(choice), nil
}

// getScreenshotConfig връща screenshot конфигурация
func getScreenshotConfig() Config {
	cfg := config.Get()

	return Config{
		SaveDir:    utils.ExpandPath(cfg.Commands.Screenshot.SaveDir),
		FilePrefix: cfg.Commands.Screenshot.FilePrefix,
	}
}

// generateFilename генерира име на файл
func generateFilename(cfg Config, fileType string) string {
	// Replace spaces with dashes
	fileType = strings.ReplaceAll(strings.ToLower(fileType), " ", "-")
	timestamp := utils.GetTimestamp()
	filename := fmt.Sprintf("%s-%s-%s.png", cfg.FilePrefix, fileType, timestamp)
	return filepath.Join(cfg.SaveDir, filename)
}

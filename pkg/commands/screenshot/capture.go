package screenshot

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/lvim-tech/ql/internal/utils"
)

// takeScreenshot изпълнява screenshot командата
func takeScreenshot(platform string, mode CaptureMode, delay int, dest Destination, filename string) error {
	// Add delay if needed
	if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Second)
	}

	switch platform {
	case "wayland":
		return takeScreenshotWayland(mode, dest, filename)
	case "x11":
		return takeScreenshotX11(mode, delay, dest, filename)
	default:
		return fmt.Errorf("unsupported platform: %s", platform)
	}
}

// takeScreenshotWayland с grim/slurp
func takeScreenshotWayland(mode CaptureMode, dest Destination, filename string) error {
	var grimArgs []string

	switch mode {
	case CaptureModeFullscreen:
		// No args needed

	case CaptureModeWindow:
		// Try to get focused window geometry with swaymsg
		output, err := utils.RunCommand("swaymsg", "-t", "get_tree")
		if err == nil && output != "" {
			// Use jq to extract geometry
			geo, err := utils.RunCommand("sh", "-c",
				`echo '`+output+`' | jq -r '..  | select(.focused?) | .rect | "\(.x),\(.y) \(.width)x\(.height)"'`)
			if err == nil && geo != "" {
				grimArgs = []string{"-g", strings.TrimSpace(geo)}
			} else {
				// Fallback to slurp
				return takeScreenshotWithSlurp(dest, filename)
			}
		} else {
			// Fallback to slurp
			return takeScreenshotWithSlurp(dest, filename)
		}

	case CaptureModeRegion:
		return takeScreenshotWithSlurp(dest, filename)

	case CaptureModeOutput:
		// Get current output
		output, err := utils.RunCommand("swaymsg", "-t", "get_outputs")
		if err == nil {
			outputName, _ := utils.RunCommand("sh", "-c",
				`echo '`+output+`' | jq -r '. [] | select(.focused) | .name'`)
			outputName = strings.TrimSpace(outputName)
			if outputName != "" {
				grimArgs = []string{"-o", outputName}
			}
		}
		// Fallback
		if len(grimArgs) == 0 {
			grimArgs = []string{"-o", "eDP-1"}
		}
	}

	return executeGrim(grimArgs, dest, filename)
}

// takeScreenshotWithSlurp използва slurp за region selection
func takeScreenshotWithSlurp(dest Destination, filename string) error {
	// Run slurp first to get geometry
	slurpCmd := exec.Command("slurp")
	output, err := slurpCmd.Output()
	if err != nil {
		return fmt.Errorf("slurp failed (did you cancel selection?): %w", err)
	}

	geometry := strings.TrimSpace(string(output))
	if geometry == "" {
		return fmt.Errorf("no region selected")
	}

	// Now run grim with the geometry
	return executeGrim([]string{"-g", geometry}, dest, filename)
}

// executeGrim изпълнява grim командата
func executeGrim(args []string, dest Destination, filename string) error {
	var cmd *exec.Cmd

	switch dest {
	case DestinationFile:
		args = append(args, filename)
		cmd = exec.Command("grim", args...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("grim failed: %w", err)
		}
		utils.Notify("Screenshot Saved", filename)

	case DestinationClipboard:
		// grim - | wl-copy -t image/png
		grimCmd := exec.Command("grim", append(args, "-")...)
		wlCopyCmd := exec.Command("wl-copy", "-t", "image/png")

		pipe, err := grimCmd.StdoutPipe()
		if err != nil {
			return err
		}
		wlCopyCmd.Stdin = pipe

		if err := grimCmd.Start(); err != nil {
			return err
		}
		if err := wlCopyCmd.Start(); err != nil {
			return err
		}
		if err := grimCmd.Wait(); err != nil {
			return err
		}
		if err := wlCopyCmd.Wait(); err != nil {
			return err
		}
		utils.Notify("Screenshot Saved", "Clipboard")

	case DestinationBoth:
		// grim - | tee file | wl-copy -t image/png
		grimCmd := exec.Command("grim", append(args, "-")...)
		teeCmd := exec.Command("tee", filename)
		wlCopyCmd := exec.Command("wl-copy", "-t", "image/png")

		pipe1, err := grimCmd.StdoutPipe()
		if err != nil {
			return err
		}
		teeCmd.Stdin = pipe1

		pipe2, err := teeCmd.StdoutPipe()
		if err != nil {
			return err
		}
		wlCopyCmd.Stdin = pipe2

		if err := grimCmd.Start(); err != nil {
			return err
		}
		if err := teeCmd.Start(); err != nil {
			return err
		}
		if err := wlCopyCmd.Start(); err != nil {
			return err
		}

		if err := grimCmd.Wait(); err != nil {
			return err
		}
		if err := teeCmd.Wait(); err != nil {
			return err
		}
		if err := wlCopyCmd.Wait(); err != nil {
			return err
		}
		utils.Notify("Screenshot Saved", filename+" and Clipboard")
	}

	return nil
}

// takeScreenshotX11 с maim
func takeScreenshotX11(mode CaptureMode, delay int, dest Destination, filename string) error {
	var maimArgs []string

	// Add delay (maim uses --delay)
	if delay > 0 {
		maimArgs = append(maimArgs, fmt.Sprintf("--delay=%d", delay))
	} else {
		maimArgs = append(maimArgs, "--delay=0.5")
	}

	// Quality
	maimArgs = append(maimArgs, "-q")

	switch mode {
	case CaptureModeFullscreen:
		// No extra args
	case CaptureModeWindow:
		// Get active window ID
		output, err := utils.RunCommand("xdotool", "getactivewindow")
		if err != nil {
			return fmt.Errorf("failed to get active window: %w", err)
		}
		windowID := strings.TrimSpace(output)
		maimArgs = append(maimArgs, "-i", windowID)
	case CaptureModeRegion:
		maimArgs = append(maimArgs, "-s")
	}

	return executeMaim(maimArgs, dest, filename)
}

// executeMaim изпълнява maim командата
func executeMaim(args []string, dest Destination, filename string) error {
	var cmd *exec.Cmd

	switch dest {
	case DestinationFile:
		args = append(args, filename)
		cmd = exec.Command("maim", args...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("maim failed: %w", err)
		}
		utils.Notify("Screenshot Saved", filename)

	case DestinationClipboard:
		// maim | xclip -selection clipboard -t image/png
		maimCmd := exec.Command("maim", args...)
		xclipCmd := exec.Command("xclip", "-selection", "clipboard", "-t", "image/png")

		pipe, err := maimCmd.StdoutPipe()
		if err != nil {
			return err
		}
		xclipCmd.Stdin = pipe

		if err := maimCmd.Start(); err != nil {
			return err
		}
		if err := xclipCmd.Start(); err != nil {
			return err
		}
		if err := maimCmd.Wait(); err != nil {
			return err
		}
		if err := xclipCmd.Wait(); err != nil {
			return err
		}
		utils.Notify("Screenshot Saved", "Clipboard")

	case DestinationBoth:
		// maim | tee file | xclip -selection clipboard -t image/png
		maimCmd := exec.Command("maim", args...)
		teeCmd := exec.Command("tee", filename)
		xclipCmd := exec.Command("xclip", "-selection", "clipboard", "-t", "image/png")

		pipe1, err := maimCmd.StdoutPipe()
		if err != nil {
			return err
		}
		teeCmd.Stdin = pipe1

		pipe2, err := teeCmd.StdoutPipe()
		if err != nil {
			return err
		}
		xclipCmd.Stdin = pipe2

		if err := maimCmd.Start(); err != nil {
			return err
		}
		if err := teeCmd.Start(); err != nil {
			return err
		}
		if err := xclipCmd.Start(); err != nil {
			return err
		}

		if err := maimCmd.Wait(); err != nil {
			return err
		}
		if err := teeCmd.Wait(); err != nil {
			return err
		}
		if err := xclipCmd.Wait(); err != nil {
			return err
		}
		utils.Notify("Screenshot Saved", filename+" and Clipboard")
	}

	return nil
}

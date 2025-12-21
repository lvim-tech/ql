package launcher

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"github.com/lvim-tech/ql/pkg/config"
)

type Rofi struct {
	baseLauncher // <-- ДОБАВИ ТОВА
}

func NewRofi(cfg *config.Config) *Rofi {
	return &Rofi{
		baseLauncher: baseLauncher{cfg: cfg},
	}
}

func (r *Rofi) Show(options []string, prompt string) (string, error) {
	launcherCfg := r.cfg.GetLauncherConfig("rofi")
	args := append(launcherCfg.Args, prompt)

	cmd := exec.Command("rofi", args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start rofi: %w", err)
	}

	// Write options to stdin
	for _, option := range options {
		fmt.Fprintln(stdin, option)
	}
	stdin.Close()

	// Read selection
	scanner := bufio.NewScanner(stdout)
	var choice string
	if scanner.Scan() {
		choice = strings.TrimSpace(scanner.Text())
	}

	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("rofi exited with error: %w", err)
	}

	if choice == "" {
		return "", fmt.Errorf("no selection made")
	}

	return choice, nil
}

// Config() вече идва от baseLauncher - премахни го

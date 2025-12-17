package launcher

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"github.com/lvim-tech/ql/pkg/config"
)

type Rofi struct {
	config *config.Config
}

func NewRofi(cfg *config.Config) *Rofi {
	return &Rofi{config: cfg}
}

func (r *Rofi) Show(options []string, prompt string) (string, error) {
	launcherCfg := r.config.GetLauncherConfig("rofi")
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

func (r *Rofi) Config() *config.Config {
	return r.config
}

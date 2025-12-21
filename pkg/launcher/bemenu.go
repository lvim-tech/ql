package launcher

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"github.com/lvim-tech/ql/pkg/config"
)

type Bemenu struct {
	baseLauncher
}

func NewBemenu(cfg *config.Config) *Bemenu {
	return &Bemenu{
		baseLauncher: baseLauncher{cfg: cfg},
	}
}

func (b *Bemenu) Show(options []string, prompt string) (string, error) {
	launcherCfg := b.cfg.GetLauncherConfig("bemenu")
	args := append(launcherCfg.Args, "-p", prompt)

	cmd := exec.Command("bemenu", args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start bemenu: %w", err)
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
		return "", fmt.Errorf("bemenu exited with error:  %w", err)
	}

	if choice == "" {
		return "", fmt.Errorf("no selection made")
	}

	return choice, nil
}

// Config() вече идва от baseLauncher - премахни го

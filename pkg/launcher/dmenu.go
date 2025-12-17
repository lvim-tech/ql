package launcher

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"github.com/lvim-tech/ql/pkg/config"
)

type Dmenu struct {
	config *config.Config
}

func NewDmenu(cfg *config.Config) *Dmenu {
	return &Dmenu{config: cfg}
}

func (d *Dmenu) Show(options []string, prompt string) (string, error) {
	launcherCfg := d.config.GetLauncherConfig("dmenu")
	args := append(launcherCfg.Args, "-p", prompt)

	cmd := exec.Command("dmenu", args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe:  %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start dmenu: %w", err)
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
		return "", fmt.Errorf("dmenu exited with error: %w", err)
	}

	if choice == "" {
		return "", fmt.Errorf("no selection made")
	}

	return choice, nil
}

func (d *Dmenu) Config() *config.Config {
	return d.config
}

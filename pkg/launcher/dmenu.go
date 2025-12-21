package launcher

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"github.com/lvim-tech/ql/pkg/config"
)

type Dmenu struct {
	baseLauncher
}

func NewDmenu(cfg *config.Config) *Dmenu {
	return &Dmenu{
		baseLauncher: baseLauncher{cfg: cfg},
	}
}

func (d *Dmenu) Show(options []string, prompt string) (string, error) {
	launcherCfg := d.cfg.GetLauncherConfig("dmenu")
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

	for _, option := range options {
		fmt.Fprintln(stdin, option)
	}
	stdin.Close()

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

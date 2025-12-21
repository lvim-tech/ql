package launcher

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"github.com/lvim-tech/ql/pkg/config"
)

type Fuzzel struct {
	baseLauncher
}

func NewFuzzel(cfg *config.Config) *Fuzzel {
	return &Fuzzel{
		baseLauncher: baseLauncher{cfg: cfg},
	}
}

func (f *Fuzzel) Show(options []string, prompt string) (string, error) {
	launcherCfg := f.cfg.GetLauncherConfig("fuzzel")
	args := launcherCfg.Args

	cmd := exec.Command("fuzzel", args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe:  %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe:  %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start fuzzel: %w", err)
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
		return "", fmt.Errorf("fuzzel exited with error: %w", err)
	}

	if choice == "" {
		return "", fmt.Errorf("no selection made")
	}

	return choice, nil
}

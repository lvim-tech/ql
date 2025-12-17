package launcher

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/lvim-tech/ql/pkg/config"
)

type Fzf struct {
	config *config.Config
}

func NewFzf(cfg *config.Config) *Fzf {
	return &Fzf{config: cfg}
}

func (f *Fzf) Show(options []string, prompt string) (string, error) {
	launcherCfg := f.config.GetLauncherConfig("fzf")
	args := append(launcherCfg.Args, "--prompt", prompt+"> ")

	cmd := exec.Command("fzf", args...)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe:  %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start fzf: %w", err)
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
		return "", fmt.Errorf("fzf exited with error: %w", err)
	}

	if choice == "" {
		return "", fmt.Errorf("no selection made")
	}

	return choice, nil
}

func (f *Fzf) Config() *config.Config {
	return f.config
}

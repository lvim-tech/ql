package launcher

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/lvim-tech/ql/pkg/config"
)

func init() {
	Register(&FzfLauncher{})
}

type FzfLauncher struct {
	command string
	args    []string
}

func (f *FzfLauncher) Name() string {
	return "fzf"
}

func (f *FzfLauncher) Flag() string {
	return "f"
}

func (f *FzfLauncher) Description() string {
	return "Use fzf launcher"
}

func (f *FzfLauncher) IsAvailable() bool {
	if f.command != "" {
		return commandExists(f.command)
	}
	return commandExists("fzf")
}

func (f *FzfLauncher) SetCommand(command string, args []string) {
	f.command = command
	f.args = args
}

func (f *FzfLauncher) Show(options []string, prompt string) (string, error) {
	input := strings.Join(options, "\n")

	// Зареди от config
	if f.command == "" {
		cfg := config.Get()
		launcherCmd := cfg.GetLauncherCommand("fzf")
		if launcherCmd != nil {
			f.command = launcherCmd.Command
			f.args = launcherCmd.Args
		} else {
			// Fallback defaults
			f.command = "fzf"
			f.args = []string{"--height", "40%", "--reverse"}
		}
	}

	// Построй командата - fzf използва --prompt вместо -p
	cmdArgs := append(f.args, "--prompt", prompt+"> ")
	cmd := exec.Command(f.command, cmdArgs...)
	cmd.Stdin = strings.NewReader(input)

	var out bytes.Buffer
	cmd.Stdout = &out

	// FZF използва stderr за UI, така че го пренасочваме
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		// Exit status 130 означава Ctrl+C, 1 означава ESC
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 || exitErr.ExitCode() == 130 {
				return "", ErrCancelled
			}
		}
		return "", err
	}

	result := strings.TrimSpace(out.String())
	if result == "" {
		return "", ErrCancelled
	}

	return result, nil
}

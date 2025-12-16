package launcher

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/lvim-tech/ql/pkg/config"
)

func init() {
	Register(&FuzzelLauncher{})
}

type FuzzelLauncher struct {
	command string
	args    []string
}

func (f *FuzzelLauncher) Name() string {
	return "fuzzel"
}

func (f *FuzzelLauncher) Flag() string {
	return "z"
}

func (f *FuzzelLauncher) Description() string {
	return "Use fuzzel launcher"
}

func (f *FuzzelLauncher) IsAvailable() bool {
	if f.command != "" {
		return commandExists(f.command)
	}
	return commandExists("fuzzel")
}

func (f *FuzzelLauncher) SetCommand(command string, args []string) {
	f.command = command
	f.args = args
}

func (f *FuzzelLauncher) Show(options []string, prompt string) (string, error) {
	input := strings.Join(options, "\n")

	// Зареди от config
	if f.command == "" {
		cfg := config.Get()
		launcherCmd := cfg.GetLauncherCommand("fuzzel")
		if launcherCmd != nil {
			f.command = launcherCmd.Command
			f.args = launcherCmd.Args
		} else {
			// Fallback defaults
			f.command = "fuzzel"
			f.args = []string{"--dmenu"}
		}
	}

	// Построй командата - fuzzel използва --prompt или -p
	cmdArgs := append(f.args, "--prompt", prompt)
	cmd := exec.Command(f.command, cmdArgs...)
	cmd.Stdin = strings.NewReader(input)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		// Exit status 1 обикновено означава ESC/Cancel
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", ErrCancelled
		}
		return "", err
	}

	result := strings.TrimSpace(out.String())
	if result == "" {
		return "", ErrCancelled
	}

	return result, nil
}

package launcher

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/lvim-tech/ql/pkg/config"
)

func init() {
	Register(&BemenuLauncher{})
}

type BemenuLauncher struct {
	command string
	args    []string
}

func (b *BemenuLauncher) Name() string {
	return "bemenu"
}

func (b *BemenuLauncher) Flag() string {
	return "b"
}

func (b *BemenuLauncher) Description() string {
	return "Use bemenu launcher"
}

func (b *BemenuLauncher) IsAvailable() bool {
	if b.command != "" {
		return commandExists(b.command)
	}
	return commandExists("bemenu")
}

func (b *BemenuLauncher) SetCommand(command string, args []string) {
	b.command = command
	b.args = args
}

func (b *BemenuLauncher) Show(options []string, prompt string) (string, error) {
	input := strings.Join(options, "\n")

	// Зареди от config
	if b.command == "" {
		cfg := config.Get()
		launcherCmd := cfg.GetLauncherCommand("bemenu")
		if launcherCmd != nil {
			b.command = launcherCmd.Command
			b.args = launcherCmd.Args
		} else {
			// Fallback defaults
			b.command = "bemenu"
			b.args = []string{"-i", "-l", "20"}
		}
	}

	// Построй командата - bemenu използва -p за prompt
	cmdArgs := append(b.args, "-p", prompt)
	cmd := exec.Command(b.command, cmdArgs...)
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

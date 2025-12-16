package launcher

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/lvim-tech/ql/pkg/config"
)

func init() {
	Register(&DmenuLauncher{})
}

type DmenuLauncher struct {
	command string
	args    []string
}

func (d *DmenuLauncher) Name() string {
	return "dmenu"
}

func (d *DmenuLauncher) Flag() string {
	return "d"
}

func (d *DmenuLauncher) Description() string {
	return "Use dmenu launcher"
}

func (d *DmenuLauncher) IsAvailable() bool {
	if d.command != "" {
		return commandExists(d.command)
	}
	return commandExists("dmenu")
}

func (d *DmenuLauncher) SetCommand(command string, args []string) {
	d.command = command
	d.args = args
}

func (d *DmenuLauncher) Show(options []string, prompt string) (string, error) {
	input := strings.Join(options, "\n")

	// Зареди от config
	if d.command == "" {
		cfg := config.Get()
		launcherCmd := cfg.GetLauncherCommand("dmenu")
		if launcherCmd != nil {
			d.command = launcherCmd.Command
			d.args = launcherCmd.Args
		} else {
			d.command = "dmenu"
			d.args = []string{"-i", "-l", "20"}
		}
	}

	// Построй командата
	cmdArgs := append(d.args, "-p", prompt)
	cmd := exec.Command(d.command, cmdArgs...)
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

package launcher

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/lvim-tech/ql/pkg/config"
)

func init() {
	Register(&RofiLauncher{})
}

type RofiLauncher struct {
	command string
	args    []string
}

func (r *RofiLauncher) Name() string {
	return "rofi"
}

func (r *RofiLauncher) Flag() string {
	return "r"
}

func (r *RofiLauncher) Description() string {
	return "Use rofi launcher"
}

func (r *RofiLauncher) IsAvailable() bool {
	if r.command != "" {
		return commandExists(r.command)
	}
	return commandExists("rofi")
}

func (r *RofiLauncher) SetCommand(command string, args []string) {
	r.command = command
	r.args = args
}

func (r *RofiLauncher) Show(options []string, prompt string) (string, error) {
	input := strings.Join(options, "\n")

	// Зареди от config ако няма custom set
	if r.command == "" {
		cfg := config.Get()
		launcherCmd := cfg.GetLauncherCommand("rofi")
		if launcherCmd != nil {
			r.command = launcherCmd.Command
			r.args = launcherCmd.Args
		} else {
			// Fallback на defaults
			r.command = "rofi"
			r.args = []string{"-dmenu", "-i"}
		}
	}

	// Построй командата
	cmdArgs := append(r.args, "-p", prompt)
	cmd := exec.Command(r.command, cmdArgs...)
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

package launcher

import (
	"os/exec"
	"strings"
)

type Dmenu struct {
	args []string
}

func NewDmenu(args []string) *Dmenu {
	return &Dmenu{args: args}
}

func (d *Dmenu) Show(options []string, prompt string) (string, error) {
	args := append([]string{}, d.args...)
	args = append(args, "-p", prompt)

	cmd := exec.Command("dmenu", args...)
	cmd.Stdin = strings.NewReader(strings.Join(options, "\n"))

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", ErrCancelled
		}
		return "", err
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		return "", ErrCancelled
	}

	return result, nil
}

func (d *Dmenu) Name() string {
	return "dmenu"
}

func (d *Dmenu) Args() []string {
	return d.args
}

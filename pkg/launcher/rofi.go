package launcher

import (
	"os/exec"
	"strings"
)

type Rofi struct {
	args []string
}

func NewRofi(args []string) *Rofi {
	return &Rofi{args: args}
}

func (r *Rofi) Show(options []string, prompt string) (string, error) {
	args := append([]string{}, r.args...)
	args = append(args, "-p", prompt, "-dmenu")

	cmd := exec.Command("rofi", args...)
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

func (r *Rofi) Name() string {
	return "rofi"
}

func (r *Rofi) Args() []string {
	return r.args
}

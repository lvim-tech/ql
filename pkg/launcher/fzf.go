package launcher

import (
	"os/exec"
	"strings"
)

type Fzf struct {
	args []string
}

func NewFzf(args []string) *Fzf {
	return &Fzf{args: args}
}

func (f *Fzf) Show(options []string, prompt string) (string, error) {
	args := append([]string{}, f.args...)
	args = append(args, "--prompt", prompt+" ")

	cmd := exec.Command("fzf", args...)
	cmd.Stdin = strings.NewReader(strings.Join(options, "\n"))

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
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

func (f *Fzf) Name() string {
	return "fzf"
}

func (f *Fzf) Args() []string {
	return f.args
}

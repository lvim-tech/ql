package launcher

import (
	"os/exec"
	"strings"
)

type Fuzzel struct {
	args []string
}

func NewFuzzel(args []string) *Fuzzel {
	return &Fuzzel{args: args}
}

func (f *Fuzzel) Show(options []string, prompt string) (string, error) {
	args := append([]string{}, f.args...)
	args = append(args, "--dmenu", "--prompt", prompt)

	cmd := exec.Command("fuzzel", args...)
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

func (f *Fuzzel) Name() string {
	return "fuzzel"
}

func (f *Fuzzel) Args() []string {
	return f.args
}

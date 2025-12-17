package launcher

import (
	"os/exec"
	"strings"
)

type Bemenu struct {
	args []string
}

func NewBemenu(args []string) *Bemenu {
	return &Bemenu{args: args}
}

func (b *Bemenu) Show(options []string, prompt string) (string, error) {
	args := append([]string{}, b.args...)
	args = append(args, "-p", prompt)

	cmd := exec.Command("bemenu", args...)
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

func (b *Bemenu) Name() string {
	return "bemenu"
}

func (b *Bemenu) Args() []string {
	return b.args
}

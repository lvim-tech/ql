// Package launcher provides an abstraction layer for different launcher programs.
// It supports dmenu, rofi, fzf, bemenu, and fuzzel with a unified interface.
// Launchers can be selected via flags or configuration, and the package handles
// command execution and error handling automatically.
package launcher

import (
	"fmt"
	"os/exec"
	"strings"
)

// Context съдържа избрания launcher за команда
type Context struct {
	Launcher Launcher
}

// Show показва menu с опции
func (c *Context) Show(options []string, prompt string) (string, error) {
	if c.Launcher == nil {
		return "", ErrNoLauncher
	}
	return c.Launcher.Show(options, prompt)
}

// ShowInput показва input prompt (за парола, текст и т.н.)
func (c *Context) ShowInput(prompt, defaultValue string) (string, error) {
	l := c.Launcher
	if l == nil {
		return "", ErrNoLauncher
	}

	switch l.Name() {
	case "rofi":
		return rofiInput(prompt, defaultValue, l.Args())
	case "dmenu":
		return dmenuInput(prompt, defaultValue, l.Args())
	case "fzf":
		return fzfInput(prompt, defaultValue, l.Args())
	case "bemenu":
		return bemenuInput(prompt, defaultValue, l.Args())
	case "fuzzel":
		return fuzzelInput(prompt, defaultValue, l.Args())
	default:
		return "", fmt.Errorf("input not supported for launcher: %s", l.Name())
	}
}

// LauncherName връща името на текущия launcher
func (c *Context) LauncherName() string {
	if c.Launcher == nil {
		return "none"
	}
	return c.Launcher.Name()
}

// Helper functions за input

func rofiInput(prompt, defaultValue string, baseArgs []string) (string, error) {
	args := append([]string{}, baseArgs...)
	args = append(args, "-p", prompt, "-dmenu")

	cmd := exec.Command("rofi", args...)

	if defaultValue != "" {
		cmd.Stdin = strings.NewReader(defaultValue)
	}

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", ErrCancelled
		}
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func dmenuInput(prompt, defaultValue string, baseArgs []string) (string, error) {
	args := append([]string{}, baseArgs...)
	args = append(args, "-p", prompt)

	cmd := exec.Command("dmenu", args...)

	if defaultValue != "" {
		cmd.Stdin = strings.NewReader(defaultValue)
	}

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", ErrCancelled
		}
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func fzfInput(prompt, defaultValue string, baseArgs []string) (string, error) {
	args := append([]string{}, baseArgs...)
	args = append(args, "--print-query", "--prompt", prompt+" ")

	cmd := exec.Command("fzf", args...)

	if defaultValue != "" {
		cmd.Stdin = strings.NewReader(defaultValue)
	}

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return "", ErrCancelled
		}
		return "", err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) > 0 {
		return lines[0], nil
	}

	return "", nil
}

func bemenuInput(prompt, defaultValue string, baseArgs []string) (string, error) {
	args := append([]string{}, baseArgs...)
	args = append(args, "-p", prompt)

	cmd := exec.Command("bemenu", args...)

	if defaultValue != "" {
		cmd.Stdin = strings.NewReader(defaultValue)
	}

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", ErrCancelled
		}
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func fuzzelInput(prompt, defaultValue string, baseArgs []string) (string, error) {
	args := append([]string{}, baseArgs...)
	args = append(args, "--dmenu", "--prompt", prompt)

	cmd := exec.Command("fuzzel", args...)

	if defaultValue != "" {
		cmd.Stdin = strings.NewReader(defaultValue)
	}

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", ErrCancelled
		}
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

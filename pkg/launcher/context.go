// Package launcher provides an abstraction layer for different launcher programs.
// It supports dmenu, rofi, fzf, bemenu, and fuzzel with a unified interface.
// Launchers can be selected via flags or configuration, and the package handles
// command execution and error handling automatically.
package launcher

import "fmt"

// Context съдържа избрания launcher за команда
type Context struct {
	launcher Launcher
}

// NewContextFromFlags създава context от CLI flags
func NewContextFromFlags(flags map[string]bool) *Context {
	// Провери всеки флаг
	for flag, enabled := range flags {
		if enabled {
			if l := GetByFlag(flag); l != nil {
				return &Context{launcher: l}
			}
		}
	}

	// Ако няма флаг - auto-detect
	return &Context{launcher: DetectAvailable()}
}

// Show показва menu с опции
func (c *Context) Show(options []string, prompt string) (string, error) {
	if c.launcher == nil {
		return "", fmt.Errorf("no launcher available - please install dmenu, rofi, fzf, bemenu, or fuzzel")
	}
	return c.launcher.Show(options, prompt)
}

// LauncherName връща името на текущия launcher
func (c *Context) LauncherName() string {
	if c.launcher == nil {
		return "none"
	}
	return c.launcher.Name()
}

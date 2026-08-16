// Package utils provides common utility functions for ql commands.
// It includes helpers for file operations, command execution,
// display server detection, and process management.
package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// ============================================================================
// Display Server Detection
// ============================================================================

// ServerType represents the display server type
type ServerType int

const (
	Unknown ServerType = iota
	X11
	Wayland
)

// DetectDisplayServer detects the current display server
func DetectDisplayServer() ServerType {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return Wayland
	}
	if os.Getenv("DISPLAY") != "" {
		return X11
	}
	return Unknown
}

// String returns string representation of ServerType
func (s ServerType) String() string {
	switch s {
	case X11:
		return "X11"
	case Wayland:
		return "Wayland"
	default:
		return "Unknown"
	}
}

// IsX11 checks if display server is X11
func (s ServerType) IsX11() bool {
	return s == X11
}

// IsWayland checks if display server is Wayland
func (s ServerType) IsWayland() bool {
	return s == Wayland
}

// IsUnknown checks if display server is unknown
func (s ServerType) IsUnknown() bool {
	return s == Unknown
}

// GetCurrentDesktop returns XDG_CURRENT_DESKTOP environment variable
func GetCurrentDesktop() string {
	return os.Getenv("XDG_CURRENT_DESKTOP")
}

// ============================================================================
// Command Utilities
// ============================================================================

// CommandExists checks if a command exists in PATH
func CommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// StartDetachedProcess starts a process completely detached (daemon mode)
func StartDetachedProcess(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}
	return cmd.Start()
}

// ============================================================================
// Process Management
// ============================================================================

// KillProcessByName kills all processes with given name
func KillProcessByName(name string) error {
	cmd := exec.Command("pkill", "-9", name)
	return cmd.Run()
}

// ============================================================================
// File System Utilities
// ============================================================================

// ExpandHomeDir expands ~ in paths
func ExpandHomeDir(path string) string {
	if len(path) > 0 && path[0] == '~' {
		return filepath.Join(GetHomeDir(), path[1:])
	}
	return path
}

// EnsureDir creates directory if it doesn't exist
func EnsureDir(path string) error {
	path = ExpandHomeDir(path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

// FileExists checks if file exists
func FileExists(path string) bool {
	path = ExpandHomeDir(path)
	_, err := os.Stat(path)
	return err == nil
}

// ============================================================================
// Runtime State
// ============================================================================

// RuntimeDir returns the per-user directory for ql's runtime state, creating
// it if needed.
//
// Modules used to keep their state at fixed names in /tmp
// (/tmp/ql_videorecord.pid and friends). /tmp is world-writable, so any other
// user could pre-create those paths as symlinks and have ql clobber the target
// on its own behalf — and, worse, plant a PID that ql would then read back and
// signal. XDG_RUNTIME_DIR is per-user and mode 0700, which removes both.
//
// The fallback is a 0700 directory of our own under TempDir rather than a bare
// /tmp name, so the guarantee holds even in a session that sets no
// XDG_RUNTIME_DIR.
func RuntimeDir() string {
	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" {
		base = filepath.Join(os.TempDir(), fmt.Sprintf("ql-%d", os.Getuid()))
	}

	dir := filepath.Join(base, "ql")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return ""
	}
	return dir
}

// RuntimeStatePath returns the path of a runtime state file named name.
// Returns "" when no runtime directory could be prepared.
func RuntimeStatePath(name string) string {
	dir := RuntimeDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, name)
}

// ============================================================================
// Shell Quoting
// ============================================================================

// ShellQuote wraps s so that `sh -c` sees it as one literal word.
//
// Go's %q is NOT a shell quote: it produces a double-quoted string, and inside
// double quotes the shell still expands $, backticks and backslashes. A path
// or a manpage name carrying $(...) is therefore executed. Single quotes
// suppress
// every expansion, and the only character that cannot appear inside them is the
// single quote itself — closed, escaped, reopened here.
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ============================================================================
// Timestamp Utilities
// ============================================================================

// GetTimestamp returns timestamp for filenames (YYYY-MM-DD_HH-MM-SS)
func GetTimestamp() string {
	return time.Now().Format("2006-01-02_15-04-05")
}

// GetTimestampMillis returns timestamp with milliseconds
func GetTimestampMillis() string {
	return time.Now().Format("2006-01-02_15-04-05.000")
}

// ============================================================================
// Environment Utilities
// ============================================================================

// GetHomeDir returns home directory
func GetHomeDir() string {
	return os.Getenv("HOME")
}

// ============================================================================
// Password Input Utilities
// ============================================================================

// PromptPassword shows password prompt with appropriate launcher
func PromptPassword(prompt string) (string, error) {
	// Try rofi first (best password support)
	if CommandExists("rofi") {
		cmd := exec.Command("rofi", "-dmenu", "-password", "-p", prompt)
		output, err := cmd.Output()
		if err == nil {
			password := strings.TrimSpace(string(output))
			if password != "" {
				return password, nil
			}
		}
	}

	// Try zenity (GUI password dialog)
	if CommandExists("zenity") {
		cmd := exec.Command("zenity", "--password", "--title", prompt)
		output, err := cmd.Output()
		if err == nil {
			password := strings.TrimSpace(string(output))
			if password != "" {
				return password, nil
			}
		}
	}

	// Try dmenu (no password masking, but works)
	if CommandExists("dmenu") {
		cmd := exec.Command("dmenu", "-p", prompt)
		output, err := cmd.Output()
		if err == nil {
			password := strings.TrimSpace(string(output))
			if password != "" {
				return password, nil
			}
		}
	}

	return "", fmt.Errorf("no password prompt tool found (rofi, zenity, dmenu)")
}

// ============================================================================
// Terminal Detection
// ============================================================================

// IsTerminal checks if program is running in a terminal
func IsTerminal() bool {
	stdinInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	stdinIsTTY := (stdinInfo.Mode() & os.ModeCharDevice) != 0

	if !stdinIsTTY {
		return false
	}

	tty, err := os.Open("/dev/tty")
	if err != nil {
		return false
	}
	tty.Close()

	return true
}

// extraBinDirs are the directories searched after PATH.
//
// A compositor keybinding, a waybar module, a .desktop entry — none of them get the interactive
// shell's PATH, and this machine deliberately keeps clipack's bin out of the session's (see
// ~/.zprofile). So a program installed by clipack is invisible to anything launched that way, and
// the failure is silent: DetectTerminal simply moved down its list and returned the next terminal
// that happened to be a distribution package. Mail opened in the wrong emulator and nothing said
// why.
//
// The base path is read from clipack's own config rather than assumed, because that is where it is
// configured; ~/clipack is only the fallback.
func extraBinDirs() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	base := filepath.Join(home, "clipack")
	if data, err := os.ReadFile(filepath.Join(home, ".config", "clipack", "config.yaml")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if _, value, found := strings.Cut(strings.TrimSpace(line), "base:"); found {
				if v := strings.TrimSpace(value); v != "" {
					base = v
				}
				break
			}
		}
	}
	return []string{filepath.Join(base, "bin")}
}

// Look resolves a command to a path, searching PATH first and then the directories PATH does not
// carry. Returns "" when it is nowhere.
//
// The ABSOLUTE path is what callers want: finding a program and then exec'ing it by bare name
// would fail all over again in the environment that could not find it in the first place.
func Look(cmd string) string {
	if p, err := exec.LookPath(cmd); err == nil {
		return p
	}
	for _, dir := range extraBinDirs() {
		candidate := filepath.Join(dir, cmd)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return candidate
		}
	}
	return ""
}

// TerminalArgs are the options a terminal needs before `-e`, given what is about to run in it.
//
// kitty gets `term=kitty-direct`. Its default TERM is xterm-kitty, whose terminfo declares 256
// colours; a program that emits `#rrggbb` — neomutt's colour commands, which this desktop's config
// uses throughout — then either prints them as errors or falls back to an approximation. The
// direct-colour entry is the same terminal with 24-bit colour declared, and it ships with kitty.
//
// Setting TERM in the environment does not work: kitty sets its child's TERM from its own `term`
// option, so the parent's value is overwritten. The override has to be passed to kitty itself.
func TerminalArgs(terminal string) []string {
	if filepath.Base(terminal) == "kitty" {
		return []string{"-o", "term=kitty-direct"}
	}
	return nil
}

// DetectTerminal returns the path of the first terminal emulator it finds.
func DetectTerminal() string {
	terminals := []string{
		"kitty",
		"alacritty",
		"foot",
		"wezterm",
		"gnome-terminal",
		"konsole",
		"xterm",
	}

	for _, term := range terminals {
		if p := Look(term); p != "" {
			return p
		}
	}

	return ""
}

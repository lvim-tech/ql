// Package utils provides common utility functions for ql commands.
// It includes helpers for file operations, command execution,
// display server detection, and process management.
package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

// GetSessionType returns XDG_SESSION_TYPE environment variable
func GetSessionType() string {
	return os.Getenv("XDG_SESSION_TYPE")
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

// RunCommand executes a command and returns output
func RunCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// RunCommandBackground executes a command in background
func RunCommandBackground(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Start()
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

// IsProcessRunning checks if a process with given name is running
func IsProcessRunning(name string) bool {
	cmd := exec.Command("pgrep", name)
	return cmd.Run() == nil
}

// GetProcessPID returns PID of process by name (first found)
func GetProcessPID(name string) (int, error) {
	cmd := exec.Command("pgrep", name)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	pidStr := strings.TrimSpace(string(output))
	if pidStr == "" {
		return 0, fmt.Errorf("no process found")
	}

	lines := strings.Split(pidStr, "\n")
	return strconv.Atoi(lines[0])
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

// EnsureDirExpanded creates directory after home expansion
// Alias for EnsureDir (for backward compatibility)
func EnsureDirExpanded(path string) error {
	return EnsureDir(path)
}

// FileExists checks if file exists
func FileExists(path string) bool {
	path = ExpandHomeDir(path)
	_, err := os.Stat(path)
	return err == nil
}

// IsDirectory checks if path is a directory
func IsDirectory(path string) bool {
	path = ExpandHomeDir(path)
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
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

// GetTimestampCustom returns timestamp with custom format
func GetTimestampCustom(format string) string {
	return time.Now().Format(format)
}

// ============================================================================
// Environment Utilities
// ============================================================================

// GetEnvOrDefault returns environment variable or default value
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetHomeDir returns home directory
func GetHomeDir() string {
	return os.Getenv("HOME")
}

// GetConfigDir returns XDG config directory
func GetConfigDir() string {
	if configDir := os.Getenv("XDG_CONFIG_HOME"); configDir != "" {
		return configDir
	}
	return filepath.Join(GetHomeDir(), ".config")
}

// GetDataDir returns XDG data directory
func GetDataDir() string {
	if dataDir := os.Getenv("XDG_DATA_HOME"); dataDir != "" {
		return dataDir
	}
	return filepath.Join(GetHomeDir(), ".local", "share")
}

// GetCacheDir returns XDG cache directory
func GetCacheDir() string {
	if cacheDir := os.Getenv("XDG_CACHE_HOME"); cacheDir != "" {
		return cacheDir
	}
	return filepath.Join(GetHomeDir(), ".cache")
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

// DetectTerminal detects available terminal emulator
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
		if CommandExists(term) {
			return term
		}
	}

	return ""
}

// Package utils provides common utility functions for ql commands.
// It includes helpers for file operations, command execution, notifications,
// display server detection, and more.
package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// ============================================================================
// Display Server Detection
// ============================================================================

// ServerType представлява типа display server
type ServerType int

const (
	Unknown ServerType = iota
	X11
	Wayland
)

// DetectDisplayServer открива текущия display server
func DetectDisplayServer() ServerType {
	// Провери за Wayland
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return Wayland
	}

	// Провери за X11
	if os.Getenv("DISPLAY") != "" {
		return X11
	}

	return Unknown
}

// String връща string представяне на ServerType
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

// IsX11 проверява дали е X11
func (s ServerType) IsX11() bool {
	return s == X11
}

// IsWayland проверява дали е Wayland
func (s ServerType) IsWayland() bool {
	return s == Wayland
}

// IsUnknown проверява дали е неизвестен
func (s ServerType) IsUnknown() bool {
	return s == Unknown
}

// GetSessionType връща XDG_SESSION_TYPE ако е зададен
func GetSessionType() string {
	return os.Getenv("XDG_SESSION_TYPE")
}

// GetCurrentDesktop връща XDG_CURRENT_DESKTOP ако е зададен
func GetCurrentDesktop() string {
	return os.Getenv("XDG_CURRENT_DESKTOP")
}

// ============================================================================
// Command Utilities
// ============================================================================

// CommandExists проверява дали команда съществува в PATH
func CommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// RunCommand изпълнява команда и връща output
func RunCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// RunCommandBackground изпълнява команда в background
func RunCommandBackground(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Start()
}

// RunCommandWithEnv изпълнява команда с custom environment
func RunCommandWithEnv(env []string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Env = env
	return cmd.Run()
}

// ============================================================================
// File System Utilities
// ============================================================================

// EnsureDir създава директория ако не съществува
func EnsureDir(path string) error {
	// Разшири ~ за home directory
	path = ExpandPath(path)

	// Провери дали съществува
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}

	return nil
}

// ExpandPath разширява ~ и environment variables в път
func ExpandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home := os.Getenv("HOME")
		path = filepath.Join(home, path[1:])
	}
	return os.ExpandEnv(path)
}

// FileExists проверява дали файл съществува
func FileExists(path string) bool {
	path = ExpandPath(path)
	_, err := os.Stat(path)
	return err == nil
}

// IsDirectory проверява дали пътят е директория
func IsDirectory(path string) bool {
	path = ExpandPath(path)
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ============================================================================
// Timestamp Utilities
// ============================================================================

// GetTimestamp връща timestamp за имена на файлове
func GetTimestamp() string {
	return time.Now().Format("2006-01-02_15-04-05")
}

// GetTimestampMillis връща timestamp с милисекунди
func GetTimestampMillis() string {
	return time.Now().Format("2006-01-02_15-04-05.000")
}

// GetTimestampCustom връща timestamp с custom format
func GetTimestampCustom(format string) string {
	return time.Now().Format(format)
}

// ============================================================================
// Notification Utilities
// ============================================================================

// Notify изпраща desktop notification
func Notify(title, message string) error {
	return NotifyWithOptions(title, message, NotifyOptions{})
}

// NotifyWithIcon изпраща notification с икона
func NotifyWithIcon(title, message, icon string) error {
	return NotifyWithOptions(title, message, NotifyOptions{Icon: icon})
}

// NotifyWithUrgency изпраща notification с urgency level
func NotifyWithUrgency(title, message, urgency string) error {
	return NotifyWithOptions(title, message, NotifyOptions{Urgency: urgency})
}

// NotifyOptions представлява опции за notification
type NotifyOptions struct {
	Icon     string
	Urgency  string // "low", "normal", "critical"
	Timeout  int    // milliseconds
	Category string
}

// NotifyWithOptions изпраща notification с пълни опции
func NotifyWithOptions(title, message string, opts NotifyOptions) error {
	// Опитай dunstify първо (по-мощен)
	if CommandExists("dunstify") {
		return sendDunstify(title, message, opts)
	}

	// Fallback към notify-send
	if CommandExists("notify-send") {
		return sendNotifySend(title, message, opts)
	}

	// Няма notification daemon
	return nil
}

func sendDunstify(title, message string, opts NotifyOptions) error {
	args := []string{}

	if opts.Icon != "" {
		args = append(args, "-i", opts.Icon)
	}

	if opts.Urgency != "" {
		args = append(args, "-u", opts.Urgency)
	}

	if opts.Timeout > 0 {
		args = append(args, "-t", fmt.Sprintf("%d", opts.Timeout))
	}

	args = append(args, title, message)

	cmd := exec.Command("dunstify", args...)
	return cmd.Run()
}

func sendNotifySend(title, message string, opts NotifyOptions) error {
	args := []string{}

	if opts.Icon != "" {
		args = append(args, "-i", opts.Icon)
	}

	if opts.Urgency != "" {
		args = append(args, "-u", opts.Urgency)
	}

	if opts.Timeout > 0 {
		args = append(args, "-t", fmt.Sprintf("%d", opts.Timeout))
	}

	args = append(args, title, message)

	cmd := exec.Command("notify-send", args...)
	return cmd.Run()
}

// ============================================================================
// UI Utilities
// ============================================================================

// Confirm пита потребителя за потвърждение (yes/no)
func Confirm(ctx any, prompt string) (bool, error) {
	// Type assertion за launcher context
	type Shower interface {
		Show([]string, string) (string, error)
	}

	shower, ok := ctx.(Shower)
	if !ok {
		return false, fmt.Errorf("invalid context type")
	}

	options := []string{"Yes", "No"}
	choice, err := shower.Show(options, prompt)
	if err != nil {
		return false, err
	}

	return choice == "Yes", nil
}

// ============================================================================
// Environment Utilities
// ============================================================================

// GetEnvOrDefault връща environment variable или default стойност
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetHomeDir връща home директорията
func GetHomeDir() string {
	return os.Getenv("HOME")
}

// GetConfigDir връща XDG config директорията
func GetConfigDir() string {
	if configDir := os.Getenv("XDG_CONFIG_HOME"); configDir != "" {
		return configDir
	}
	return filepath.Join(GetHomeDir(), ".config")
}

// GetDataDir връща XDG data директорията
func GetDataDir() string {
	if dataDir := os.Getenv("XDG_DATA_HOME"); dataDir != "" {
		return dataDir
	}
	return filepath.Join(GetHomeDir(), ".local", "share")
}

// GetCacheDir връща XDG cache директорията
func GetCacheDir() string {
	if cacheDir := os.Getenv("XDG_CACHE_HOME"); cacheDir != "" {
		return cacheDir
	}
	return filepath.Join(GetHomeDir(), ".cache")
}

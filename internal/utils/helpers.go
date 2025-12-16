// Package utils provides common utility functions for ql commands.
// It includes helpers for file operations, command execution, notifications, and more.
package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// CommandExists проверява дали команда съществува в PATH
func CommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// EnsureDir създава директория ако не съществува
func EnsureDir(path string) error {
	// Разшири ~ за home directory
	if len(path) > 0 && path[0] == '~' {
		home := os.Getenv("HOME")
		path = filepath.Join(home, path[1:])
	}

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

// GetTimestamp връща timestamp за имена на файлове
func GetTimestamp() string {
	return time.Now().Format("2006-01-02_15-04-05")
}

// GetTimestampMillis връща timestamp с милисекунди
func GetTimestampMillis() string {
	return time.Now().Format("2006-01-02_15-04-05.000")
}

// Notify изпраща desktop notification
func Notify(title, message string) error {
	if !CommandExists("notify-send") {
		// Няма notify-send - skip notification
		return nil
	}

	cmd := exec.Command("notify-send", title, message)
	return cmd.Run()
}

// NotifyWithIcon изпраща notification с икона
func NotifyWithIcon(title, message, icon string) error {
	if !CommandExists("notify-send") {
		return nil
	}

	cmd := exec.Command("notify-send", "-i", icon, title, message)
	return cmd.Run()
}

// NotifyWithUrgency изпраща notification с urgency level
func NotifyWithUrgency(title, message, urgency string) error {
	if !CommandExists("notify-send") {
		return nil
	}

	cmd := exec.Command("notify-send", "-u", urgency, title, message)
	return cmd.Run()
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

// GetEnvOrDefault връща environment variable или default стойност
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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

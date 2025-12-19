// Package utils provides notification utilities for ql.
// Supports configurable notification behavior via NotificationConfig.
package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/lvim-tech/ql/pkg/config"
)

// NotifyWithConfig sends a notification using the provided config
func NotifyWithConfig(cfg *config.NotificationConfig, title, message string) {
	if cfg == nil || !cfg.Enabled {
		return
	}

	// If in terminal and ShowInTerminal is enabled, print to stdout
	if cfg.ShowInTerminal && IsTerminal() {
		fmt.Printf("[%s] %s\n", title, message)
		return
	}

	// Determine which notification tool to use
	tool := cfg.Tool
	if tool == "" || tool == "auto" {
		tool = detectNotificationTool()
	}

	// Send notification
	sendNotification(tool, title, message, cfg.Timeout, cfg.Urgency, "normal")
}

// ShowErrorNotificationWithConfig sends an error notification using the provided config
func ShowErrorNotificationWithConfig(cfg *config.NotificationConfig, title, message string) {
	if cfg == nil || !cfg.Enabled {
		return
	}

	// If in terminal and ShowInTerminal is enabled, print to stderr
	if cfg.ShowInTerminal && IsTerminal() {
		fmt.Fprintf(os.Stderr, "[ERROR] [%s] %s\n", title, message)
		return
	}

	// Determine which notification tool to use
	tool := cfg.Tool
	if tool == "" || tool == "auto" {
		tool = detectNotificationTool()
	}

	// Send error notification with critical urgency
	sendNotification(tool, title, message, cfg.Timeout, "critical", "critical")
}

// ShowPersistentNotificationWithConfig shows a persistent notification that doesn't auto-close
// Returns notification ID for closing later
func ShowPersistentNotificationWithConfig(cfg *config.NotificationConfig, title, message string) int {
	if cfg == nil || !cfg.Enabled {
		return 0
	}

	// If in terminal and ShowInTerminal is enabled, print to stdout
	if cfg.ShowInTerminal && IsTerminal() {
		fmt.Printf("[PERSISTENT] [%s] %s\n", title, message)
		return 0
	}

	notifyID := int(time.Now().UnixNano() % 1000000)

	// Determine which notification tool to use
	tool := cfg.Tool
	if tool == "" || tool == "auto" {
		tool = detectNotificationTool()
	}

	if tool == "dunstify" {
		cmd := exec.Command("dunstify",
			"-u", cfg.Urgency,
			"-t", "0",
			"-r", strconv.Itoa(notifyID),
			title,
			message)
		cmd.Env = os.Environ()
		cmd.Start()
		return notifyID
	}

	if tool == "notify-send" {
		cmd := exec.Command("notify-send",
			"-u", cfg.Urgency,
			"-t", "0",
			title,
			message)
		cmd.Env = os.Environ()
		cmd.Start()
		return notifyID
	}

	return notifyID
}

// ClosePersistentNotificationWithConfig closes a persistent notification by ID
func ClosePersistentNotificationWithConfig(cfg *config.NotificationConfig, notifyID int) {
	if cfg == nil || !cfg.Enabled || notifyID == 0 {
		return
	}

	// Determine which notification tool to use
	tool := cfg.Tool
	if tool == "" || tool == "auto" {
		tool = detectNotificationTool()
	}

	if tool == "dunstify" {
		cmd := exec.Command("dunstify", "-C", strconv.Itoa(notifyID))
		cmd.Env = os.Environ()
		cmd.Run()
	}
}

// ============================================================================
// Internal Helper Functions
// ============================================================================

// detectNotificationTool detects which notification tool is available
func detectNotificationTool() string {
	if CommandExists("dunstify") {
		return "dunstify"
	}
	if CommandExists("notify-send") {
		return "notify-send"
	}
	return ""
}

// sendNotification sends a notification using the specified tool
func sendNotification(tool, title, message string, timeout int, urgency, fallbackUrgency string) {
	if tool == "" {
		return
	}

	// Use fallback urgency if urgency is not set
	if urgency == "" {
		urgency = fallbackUrgency
	}

	// Default timeout
	if timeout <= 0 {
		timeout = 5000
	}

	var cmd *exec.Cmd

	switch tool {
	case "dunstify":
		cmd = exec.Command("dunstify",
			"-u", urgency,
			"-t", strconv.Itoa(timeout),
			title,
			message)

	case "notify-send":
		cmd = exec.Command("notify-send",
			"-u", urgency,
			"-t", strconv.Itoa(timeout),
			title,
			message)

	default:
		return
	}

	if cmd != nil {
		cmd.Env = os.Environ()
		cmd.Start()
	}
}

// ============================================================================
// Backward Compatibility Helpers (deprecated, use WithConfig versions)
// ============================================================================

// Notify sends a notification using default settings
// Deprecated: Use NotifyWithConfig with proper config instead
func Notify(title, message string) {
	defaultCfg := &config.NotificationConfig{
		Enabled:        true,
		Tool:           "auto",
		Timeout:        5000,
		Urgency:        "normal",
		ShowInTerminal: false,
	}
	NotifyWithConfig(defaultCfg, title, message)
}

// ShowErrorNotification sends an error notification using default settings
// Deprecated: Use ShowErrorNotificationWithConfig with proper config instead
func ShowErrorNotification(title, message string) {
	defaultCfg := &config.NotificationConfig{
		Enabled:        true,
		Tool:           "auto",
		Timeout:        10000,
		Urgency:        "critical",
		ShowInTerminal: false,
	}
	ShowErrorNotificationWithConfig(defaultCfg, title, message)
}

// ShowPersistentNotification shows a persistent notification using default settings
// Deprecated:  Use ShowPersistentNotificationWithConfig with proper config instead
func ShowPersistentNotification(title, message string) int {
	defaultCfg := &config.NotificationConfig{
		Enabled:        true,
		Tool:           "auto",
		Timeout:        0,
		Urgency:        "normal",
		ShowInTerminal: false,
	}
	return ShowPersistentNotificationWithConfig(defaultCfg, title, message)
}

// ClosePersistentNotification closes a persistent notification using default settings
// Deprecated: Use ClosePersistentNotificationWithConfig with proper config instead
func ClosePersistentNotification(notifyID int) {
	defaultCfg := &config.NotificationConfig{
		Enabled: true,
		Tool:    "auto",
	}
	ClosePersistentNotificationWithConfig(defaultCfg, notifyID)
}

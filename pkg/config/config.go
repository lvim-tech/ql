// Package config provides configuration management for ql.
package config

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"

	_ "embed"

	"github.com/BurntSushi/toml"
)

//go:embed default.toml
var defaultConfig string

// Config represents the main configuration structure
type Config struct {
	DefaultLauncher   string                    `toml:"default_launcher"`
	MenuStyle         string                    `toml:"menu_style"`
	PdfViewer         string                    `toml:"pdf_viewer"`
	Browser           string                    `toml:"browser"`
	Editor            string                    `toml:"editor"`
	ManViewer         string                    `toml:"man_viewer"`
	ModuleOrder       []string                  `toml:"module_order"`
	ModuleGroupsOrder []string                  `toml:"module_groups_order"`
	ModuleGroups      map[string]ModuleGroup    `toml:"module_groups"`
	Launchers         map[string]LauncherConfig `toml:"launchers"`
	Notifications     NotificationConfig        `toml:"notifications"`
	Commands          map[string]map[string]any `toml:"commands"`
}

// ModuleGroup represents a group of related modules
type ModuleGroup struct {
	Name    string   `toml:"name"`
	Enabled bool     `toml:"enabled"`
	Modules []string `toml:"modules"`
}

// LauncherConfig represents launcher-specific configuration
type LauncherConfig struct {
	Args []string `toml:"args"`
}

// NotificationConfig controls notification behavior
type NotificationConfig struct {
	Enabled        bool   `toml:"enabled"`
	Tool           string `toml:"tool"`
	Timeout        int    `toml:"timeout"`
	Urgency        string `toml:"urgency"`
	ShowInTerminal bool   `toml:"show_in_terminal"`
}

// Load loads configuration from default and user config
func Load() (*Config, error) {
	var defaultCfg Config
	if err := toml.Unmarshal([]byte(defaultConfig), &defaultCfg); err != nil {
		return nil, fmt.Errorf("failed to decode default config: %w", err)
	}

	userConfigPath := GetUserConfigPath()
	if _, err := os.Stat(userConfigPath); os.IsNotExist(err) {
		return &defaultCfg, nil
	}

	var userCfg Config
	if _, err := toml.DecodeFile(userConfigPath, &userCfg); err != nil {
		return nil, fmt.Errorf("failed to decode user config: %w", err)
	}

	mergedCfg := mergeConfigs(defaultCfg, userCfg)
	return &mergedCfg, nil
}

// InitUserConfig creates user config from default
func InitUserConfig() error {
	configPath := GetUserConfigPath()
	configDir := filepath.Dir(configPath)

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetUserConfigPath returns the path to user config
func GetUserConfigPath() string {
	home := os.Getenv("HOME")
	return filepath.Join(home, ".config", "ql", "config.toml")
}

// mergeConfigs deep merges user config into default config
func mergeConfigs(defaultCfg, userCfg Config) Config {
	result := defaultCfg

	// Merge simple string fields
	if userCfg.DefaultLauncher != "" {
		result.DefaultLauncher = userCfg.DefaultLauncher
	}
	if userCfg.MenuStyle != "" {
		result.MenuStyle = userCfg.MenuStyle
	}
	if userCfg.PdfViewer != "" {
		result.PdfViewer = userCfg.PdfViewer
	}
	if userCfg.Browser != "" {
		result.Browser = userCfg.Browser
	}
	if userCfg.Editor != "" {
		result.Editor = userCfg.Editor
	}
	if userCfg.ManViewer != "" {
		result.ManViewer = userCfg.ManViewer
	}

	// Merge arrays
	if len(userCfg.ModuleOrder) > 0 {
		result.ModuleOrder = userCfg.ModuleOrder
	}
	if len(userCfg.ModuleGroupsOrder) > 0 {
		result.ModuleGroupsOrder = userCfg.ModuleGroupsOrder
	}

	// Merge maps
	if result.ModuleGroups == nil {
		result.ModuleGroups = make(map[string]ModuleGroup)
	}
	maps.Copy(result.ModuleGroups, userCfg.ModuleGroups)

	if result.Launchers == nil {
		result.Launchers = make(map[string]LauncherConfig)
	}
	maps.Copy(result.Launchers, userCfg.Launchers)

	// Merge notification config
	if userCfg.Notifications.Tool != "" {
		result.Notifications.Tool = userCfg.Notifications.Tool
	}
	if userCfg.Notifications.Timeout != 0 {
		result.Notifications.Timeout = userCfg.Notifications.Timeout
	}
	if userCfg.Notifications.Urgency != "" {
		result.Notifications.Urgency = userCfg.Notifications.Urgency
	}
	result.Notifications.Enabled = userCfg.Notifications.Enabled || result.Notifications.Enabled
	result.Notifications.ShowInTerminal = userCfg.Notifications.ShowInTerminal

	// Merge commands
	if result.Commands == nil {
		result.Commands = make(map[string]map[string]any)
	}
	for cmdName, userCmdConfig := range userCfg.Commands {
		if result.Commands[cmdName] == nil {
			result.Commands[cmdName] = make(map[string]any)
		}
		maps.Copy(result.Commands[cmdName], userCmdConfig)
	}

	return result
}

// ============================================================================
// GLOBAL GETTERS
// ============================================================================

func (c *Config) GetDefaultLauncher() string {
	return c.DefaultLauncher
}

func (c *Config) GetMenuStyle() string {
	if c.MenuStyle == "" {
		return "flat"
	}
	return c.MenuStyle
}

func (c *Config) GetPdfViewer() string {
	if c.PdfViewer == "" {
		return "zathura"
	}
	return c.PdfViewer
}

func (c *Config) GetBrowser() string {
	if c.Browser == "" {
		return "firefox"
	}
	return c.Browser
}

func (c *Config) GetEditor() string {
	if c.Editor == "" {
		return "vim"
	}
	return c.Editor
}

func (c *Config) GetManViewer() string {
	if c.ManViewer == "" {
		return "less"
	}
	return c.ManViewer
}

// ============================================================================
// MODULE ORDER & GROUPS
// ============================================================================

func (c *Config) GetModuleOrder() []string {
	return c.ModuleOrder
}

func (c *Config) GetModuleGroupsOrder() []string {
	if len(c.ModuleGroupsOrder) > 0 {
		return c.ModuleGroupsOrder
	}
	return []string{"system", "network", "media", "info"}
}

func (c *Config) GetModuleGroups() map[string]ModuleGroup {
	result := make(map[string]ModuleGroup)
	for key, group := range c.ModuleGroups {
		if !group.Enabled {
			continue
		}
		result[key] = group
	}
	return result
}

// ============================================================================
// LAUNCHER & NOTIFICATIONS
// ============================================================================

func (c *Config) GetLauncherConfig(name string) LauncherConfig {
	if cfg, ok := c.Launchers[name]; ok {
		return cfg
	}
	return LauncherConfig{}
}

func (c *Config) GetNotificationConfig() NotificationConfig {
	return c.Notifications
}

// ============================================================================
// MODULE CONFIGS (alphabetically sorted)
// ============================================================================

func (c *Config) GetAudioRecordConfig() any {
	return c.Commands["audiorecord"]
}

func (c *Config) GetClipboardConfig() any {
	return c.Commands["clipboard"]
}

func (c *Config) GetKillConfig() any {
	return c.Commands["kill"]
}

func (c *Config) GetManConfig() any {
	return c.Commands["man"]
}

func (c *Config) GetMpcConfig() any {
	return c.Commands["mpc"]
}

func (c *Config) GetPowerConfig() any {
	return c.Commands["power"]
}

func (c *Config) GetRadioConfig() any {
	return c.Commands["radio"]
}

func (c *Config) GetScreenshotConfig() any {
	return c.Commands["screenshot"]
}

func (c *Config) GetVideoRecordConfig() any {
	return c.Commands["videorecord"]
}

func (c *Config) GetWeatherConfig() any {
	return c.Commands["weather"]
}

func (c *Config) GetWifiConfig() any {
	return c.Commands["wifi"]
}

func (c *Config) GetNetstatConfig() any {
	return c.Commands["netstat"]
}

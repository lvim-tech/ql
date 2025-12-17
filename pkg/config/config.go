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

// Config представлява главната конфигурация
type Config struct {
	DefaultLauncher string                    `toml:"default_launcher"`
	MenuStyle       string                    `toml:"menu_style"` // "flat" или "grouped"
	ModuleOrder     []string                  `toml:"module_order"`
	ModuleGroups    map[string]ModuleGroup    `toml:"module_groups"`
	Launchers       map[string]LauncherConfig `toml:"launchers"`
	Commands        map[string]map[string]any `toml:"commands"`
}

// ModuleGroup представлява група от модули
type ModuleGroup struct {
	Name        string   `toml:"name"`
	Icon        string   `toml:"icon"`
	Description string   `toml:"description"`
	Modules     []string `toml:"modules"`
}

// LauncherConfig представлява конфигурация за launcher
type LauncherConfig struct {
	Args []string `toml:"args"`
}

// Load зарежда конфигурацията
func Load() (*Config, error) {
	// Decode default config
	var defaultCfg Config
	if err := toml.Unmarshal([]byte(defaultConfig), &defaultCfg); err != nil {
		return nil, fmt.Errorf("failed to decode default config:  %w", err)
	}

	// Check if user config exists
	userConfigPath := GetUserConfigPath()
	if _, err := os.Stat(userConfigPath); os.IsNotExist(err) {
		return &defaultCfg, nil
	}

	// Load user config
	var userCfg Config
	if _, err := toml.DecodeFile(userConfigPath, &userCfg); err != nil {
		return nil, fmt.Errorf("failed to decode user config:  %w", err)
	}

	// Deep merge configs
	mergedCfg := mergeConfigs(defaultCfg, userCfg)

	return &mergedCfg, nil
}

// mergeConfigs deep merges user config into default config
func mergeConfigs(defaultCfg, userCfg Config) Config {
	result := defaultCfg

	// Override simple fields
	if userCfg.DefaultLauncher != "" {
		result.DefaultLauncher = userCfg.DefaultLauncher
	}

	if userCfg.MenuStyle != "" {
		result.MenuStyle = userCfg.MenuStyle
	}

	// Override module order
	if len(userCfg.ModuleOrder) > 0 {
		result.ModuleOrder = userCfg.ModuleOrder
	}

	// Merge module groups
	if result.ModuleGroups == nil {
		result.ModuleGroups = make(map[string]ModuleGroup)
	}
	maps.Copy(result.ModuleGroups, userCfg.ModuleGroups)

	// Merge launchers
	if result.Launchers == nil {
		result.Launchers = make(map[string]LauncherConfig)
	}
	maps.Copy(result.Launchers, userCfg.Launchers)

	// Deep merge commands
	if result.Commands == nil {
		result.Commands = make(map[string]map[string]any)
	}

	for cmdName, userCmdConfig := range userCfg.Commands {
		if result.Commands[cmdName] == nil {
			// No default config, use user config entirely
			result.Commands[cmdName] = make(map[string]any)
		}

		// Merge/override each field from user config
		maps.Copy(result.Commands[cmdName], userCmdConfig)
	}

	return result
}

// GetUserConfigPath returns the path to user config
func GetUserConfigPath() string {
	home := os.Getenv("HOME")
	return filepath.Join(home, ".config", "ql", "config.toml")
}

// InitUserConfig creates user config from default
func InitUserConfig() error {
	configPath := GetUserConfigPath()
	configDir := filepath.Dir(configPath)

	// Create config directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write default config
	if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetDefaultLauncher returns the default launcher name
func (c *Config) GetDefaultLauncher() string {
	return c.DefaultLauncher
}

// GetMenuStyle returns the menu style (flat or grouped)
func (c *Config) GetMenuStyle() string {
	if c.MenuStyle == "" {
		return "flat"
	}
	return c.MenuStyle
}

// GetModuleOrder returns the module order
func (c *Config) GetModuleOrder() []string {
	return c.ModuleOrder
}

// GetModuleGroups returns the module groups
func (c *Config) GetModuleGroups() map[string]ModuleGroup {
	return c.ModuleGroups
}

// GetLauncherConfig returns the launcher configuration
func (c *Config) GetLauncherConfig(name string) LauncherConfig {
	if cfg, ok := c.Launchers[name]; ok {
		return cfg
	}
	return LauncherConfig{}
}

// GetPowerConfig returns the power module configuration
func (c *Config) GetPowerConfig() any {
	return c.Commands["power"]
}

// GetScreenshotConfig returns the screenshot module configuration
func (c *Config) GetScreenshotConfig() any {
	return c.Commands["screenshot"]
}

// GetRadioConfig returns the radio module configuration
func (c *Config) GetRadioConfig() any {
	return c.Commands["radio"]
}

// GetWifiConfig returns the wifi module configuration
func (c *Config) GetWifiConfig() any {
	return c.Commands["wifi"]
}

// GetMpcConfig returns the mpc module configuration
func (c *Config) GetMpcConfig() any {
	return c.Commands["mpc"]
}

// GetAudioRecordConfig returns the audiorecord module configuration
func (c *Config) GetAudioRecordConfig() any {
	return c.Commands["audiorecord"]
}

// GetVideoRecordConfig returns the videorecord module configuration
func (c *Config) GetVideoRecordConfig() any {
	return c.Commands["videorecord"]
}

// GetWeatherConfig returns the weather module configuration
func (c *Config) GetWeatherConfig() any {
	return c.Commands["weather"]
}

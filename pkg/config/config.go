// Package config provides configuration management for ql.
// It handles loading, merging, and accessing configuration from default and user config files.
package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

//go:embed default.toml
var defaultConfigData string

// Config структура
type Config struct {
	DefaultLauncher string         `toml:"default_launcher"`
	Launchers       LauncherConfig `toml:"launchers"`
	Commands        CommandsConfig `toml:"commands"`
}

// LauncherConfig за всеки launcher
type LauncherConfig struct {
	Dmenu  LauncherCommand `toml:"dmenu"`
	Rofi   LauncherCommand `toml:"rofi"`
	Fzf    LauncherCommand `toml:"fzf"`
	Bemenu LauncherCommand `toml:"bemenu"`
	Fuzzel LauncherCommand `toml:"fuzzel"`
}

// LauncherCommand описва как да се стартира launcher
type LauncherCommand struct {
	Command string   `toml:"command"`
	Args    []string `toml:"args"`
}

// CommandsConfig за всички команди
type CommandsConfig struct {
	Power      PowerConfig      `toml:"power"`
	Screenshot ScreenshotConfig `toml:"screenshot"`
}

// PowerConfig за power management
type PowerConfig struct {
	Enabled bool `toml:"enabled"`

	// Show options
	ShowLogout    bool `toml:"show_logout"`
	ShowSuspend   bool `toml:"show_suspend"`
	ShowHibernate bool `toml:"show_hibernate"`
	ShowReboot    bool `toml:"show_reboot"`
	ShowShutdown  bool `toml:"show_shutdown"`

	// Confirm options
	ConfirmLogout    bool `toml:"confirm_logout"`
	ConfirmSuspend   bool `toml:"confirm_suspend"`
	ConfirmHibernate bool `toml:"confirm_hibernate"`
	ConfirmReboot    bool `toml:"confirm_reboot"`
	ConfirmShutdown  bool `toml:"confirm_shutdown"`

	// Custom commands
	LogoutCommand    string `toml:"logout_command"`
	SuspendCommand   string `toml:"suspend_command"`
	HibernateCommand string `toml:"hibernate_command"`
	RebootCommand    string `toml:"reboot_command"`
	ShutdownCommand  string `toml:"shutdown_command"`
}

// ScreenshotConfig за screenshot
type ScreenshotConfig struct {
	Enabled    bool   `toml:"enabled"`
	SaveDir    string `toml:"save_dir"`
	FilePrefix string `toml:"file_prefix"`
}

// PowerConfigFile е за четене от TOML (с pointers за optional полета)
type PowerConfigFile struct {
	Enabled *bool `toml:"enabled"`

	// Show options
	ShowLogout    *bool `toml:"show_logout"`
	ShowSuspend   *bool `toml:"show_suspend"`
	ShowHibernate *bool `toml:"show_hibernate"`
	ShowReboot    *bool `toml:"show_reboot"`
	ShowShutdown  *bool `toml:"show_shutdown"`

	// Confirm options
	ConfirmLogout    *bool `toml:"confirm_logout"`
	ConfirmSuspend   *bool `toml:"confirm_suspend"`
	ConfirmHibernate *bool `toml:"confirm_hibernate"`
	ConfirmReboot    *bool `toml:"confirm_reboot"`
	ConfirmShutdown  *bool `toml:"confirm_shutdown"`

	// Custom commands
	LogoutCommand    *string `toml:"logout_command"`
	SuspendCommand   *string `toml:"suspend_command"`
	HibernateCommand *string `toml:"hibernate_command"`
	RebootCommand    *string `toml:"reboot_command"`
	ShutdownCommand  *string `toml:"shutdown_command"`
}

// ScreenshotConfigFile е за четене от TOML
type ScreenshotConfigFile struct {
	Enabled    *bool   `toml:"enabled"`
	SaveDir    *string `toml:"save_dir"`
	FilePrefix *string `toml:"file_prefix"`
}

// CommandsConfigFile за четене от TOML
type CommandsConfigFile struct {
	Power      PowerConfigFile      `toml:"power"`
	Screenshot ScreenshotConfigFile `toml:"screenshot"`
}

// ConfigFile е за четене от TOML файл
type ConfigFile struct {
	DefaultLauncher *string            `toml:"default_launcher"`
	Launchers       LauncherConfig     `toml:"launchers"`
	Commands        CommandsConfigFile `toml:"commands"`
}

var globalConfig *Config

// GetUserConfigPath връща пътя до user config
func GetUserConfigPath() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "ql", "config.toml")
}

// GetSystemConfigPath връща пътя до system config
func GetSystemConfigPath() string {
	return "/etc/ql/config.toml"
}

// Load зарежда config с merge на defaults + user config
func Load() (*Config, error) {
	if globalConfig != nil {
		return globalConfig, nil
	}

	// 1. Зареди defaults
	defaultCfg, err := loadDefaultConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load default config: %w", err)
	}

	// 2. Опитай да заредиш user config
	userConfigPath := GetUserConfigPath()
	if _, err := os.Stat(userConfigPath); err == nil {
		userCfg, err := loadConfigFromFile(userConfigPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load user config: %v\n", err)
			fmt.Fprintf(os.Stderr, "Using default configuration\n")
			globalConfig = defaultCfg
			return globalConfig, nil
		}
		// Merge user config с defaults
		globalConfig = mergeConfigs(defaultCfg, userCfg)
		return globalConfig, nil
	}

	// 3. Опитай да заредиш system config
	systemConfigPath := GetSystemConfigPath()
	if _, err := os.Stat(systemConfigPath); err == nil {
		systemCfg, err := loadConfigFromFile(systemConfigPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load system config: %v\n", err)
			globalConfig = defaultCfg
			return globalConfig, nil
		}
		globalConfig = mergeConfigs(defaultCfg, systemCfg)
		return globalConfig, nil
	}

	// 4. Няма user/system config - използвай defaults
	globalConfig = defaultCfg
	return globalConfig, nil
}

// loadDefaultConfig зарежда вградения default config
func loadDefaultConfig() (*Config, error) {
	var cfg Config
	if _, err := toml.Decode(defaultConfigData, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// loadConfigFromFile зарежда config от файл
func loadConfigFromFile(path string) (*ConfigFile, error) {
	var cfg ConfigFile
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// mergeConfigs merge user config с defaults (user override defaults)
func mergeConfigs(defaultCfg *Config, userCfg *ConfigFile) *Config {
	merged := *defaultCfg

	// Override default_launcher ако е зададен
	if userCfg.DefaultLauncher != nil && *userCfg.DefaultLauncher != "" {
		merged.DefaultLauncher = *userCfg.DefaultLauncher
	}

	// Merge launcher configs
	if userCfg.Launchers.Dmenu.Command != "" {
		merged.Launchers.Dmenu = userCfg.Launchers.Dmenu
	}
	if userCfg.Launchers.Rofi.Command != "" {
		merged.Launchers.Rofi = userCfg.Launchers.Rofi
	}
	if userCfg.Launchers.Fzf.Command != "" {
		merged.Launchers.Fzf = userCfg.Launchers.Fzf
	}
	if userCfg.Launchers.Bemenu.Command != "" {
		merged.Launchers.Bemenu = userCfg.Launchers.Bemenu
	}
	if userCfg.Launchers.Fuzzel.Command != "" {
		merged.Launchers.Fuzzel = userCfg.Launchers.Fuzzel
	}

	// Merge Power settings
	mergePowerConfig(&merged.Commands.Power, &userCfg.Commands.Power)

	// Merge Screenshot settings
	mergeScreenshotConfig(&merged.Commands.Screenshot, &userCfg.Commands.Screenshot)

	return &merged
}

// mergePowerConfig мерджва power конфигурация
func mergePowerConfig(merged *PowerConfig, user *PowerConfigFile) {
	// Enabled
	if user.Enabled != nil {
		merged.Enabled = *user.Enabled
	}

	// Show options
	if user.ShowLogout != nil {
		merged.ShowLogout = *user.ShowLogout
	}
	if user.ShowSuspend != nil {
		merged.ShowSuspend = *user.ShowSuspend
	}
	if user.ShowHibernate != nil {
		merged.ShowHibernate = *user.ShowHibernate
	}
	if user.ShowReboot != nil {
		merged.ShowReboot = *user.ShowReboot
	}
	if user.ShowShutdown != nil {
		merged.ShowShutdown = *user.ShowShutdown
	}

	// Confirm options
	if user.ConfirmLogout != nil {
		merged.ConfirmLogout = *user.ConfirmLogout
	}
	if user.ConfirmSuspend != nil {
		merged.ConfirmSuspend = *user.ConfirmSuspend
	}
	if user.ConfirmHibernate != nil {
		merged.ConfirmHibernate = *user.ConfirmHibernate
	}
	if user.ConfirmReboot != nil {
		merged.ConfirmReboot = *user.ConfirmReboot
	}
	if user.ConfirmShutdown != nil {
		merged.ConfirmShutdown = *user.ConfirmShutdown
	}

	// Commands
	if user.LogoutCommand != nil && *user.LogoutCommand != "" {
		merged.LogoutCommand = *user.LogoutCommand
	}
	if user.SuspendCommand != nil && *user.SuspendCommand != "" {
		merged.SuspendCommand = *user.SuspendCommand
	}
	if user.HibernateCommand != nil && *user.HibernateCommand != "" {
		merged.HibernateCommand = *user.HibernateCommand
	}
	if user.RebootCommand != nil && *user.RebootCommand != "" {
		merged.RebootCommand = *user.RebootCommand
	}
	if user.ShutdownCommand != nil && *user.ShutdownCommand != "" {
		merged.ShutdownCommand = *user.ShutdownCommand
	}
}

// mergeScreenshotConfig мерджва screenshot конфигурация
func mergeScreenshotConfig(merged *ScreenshotConfig, user *ScreenshotConfigFile) {
	// Enabled
	if user.Enabled != nil {
		merged.Enabled = *user.Enabled
	}

	if user.SaveDir != nil && *user.SaveDir != "" {
		merged.SaveDir = *user.SaveDir
	}
	if user.FilePrefix != nil && *user.FilePrefix != "" {
		merged.FilePrefix = *user.FilePrefix
	}
}

// Get връща глобалния config (lazy load)
func Get() *Config {
	if globalConfig == nil {
		globalConfig, _ = Load()
	}
	return globalConfig
}

// GetLauncherCommand връща команда за конкретен launcher
func (c *Config) GetLauncherCommand(name string) *LauncherCommand {
	switch name {
	case "dmenu":
		return &c.Launchers.Dmenu
	case "rofi":
		return &c.Launchers.Rofi
	case "fzf":
		return &c.Launchers.Fzf
	case "bemenu":
		return &c.Launchers.Bemenu
	case "fuzzel":
		return &c.Launchers.Fuzzel
	default:
		return nil
	}
}

// InitUserConfig копира default config в user config директорията
func InitUserConfig() error {
	userConfigPath := GetUserConfigPath()
	userConfigDir := filepath.Dir(userConfigPath)

	// Провери дали вече съществува
	if _, err := os.Stat(userConfigPath); err == nil {
		return fmt.Errorf("config already exists: %s", userConfigPath)
	}

	// Създай директорията
	if err := os.MkdirAll(userConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Запиши default config
	if err := os.WriteFile(userConfigPath, []byte(defaultConfigData), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetDefaultConfigContent връща съдържанието на default config
func GetDefaultConfigContent() string {
	return defaultConfigData
}

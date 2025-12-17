// Package config provides configuration management for ql.
// It handles loading, merging, and accessing configuration from default and user config files.
// Configuration can be loaded from embedded defaults, user config (~/.config/ql/config. toml),
// or system config (/etc/ql/config. toml), with user settings overriding defaults.
package config

import (
	_ "embed"
	"fmt"
	"maps"
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

// CommandsConfig за всички команди
type CommandsConfig struct {
	Power       PowerConfig       `toml:"power"`
	Screenshot  ScreenshotConfig  `toml:"screenshot"`
	Radio       RadioConfig       `toml:"radio"`
	Wifi        WifiConfig        `toml:"wifi"`
	Mpc         MpcConfig         `toml:"mpc"`
	AudioRecord AudioRecordConfig `toml:"audiorecord"`
	VideoRecord VideoRecordConfig `toml:"videorecord"`
}

// PowerConfig за power management
type PowerConfig struct {
	Enabled          bool   `toml:"enabled"`
	ShowLogout       bool   `toml:"show_logout"`
	ShowSuspend      bool   `toml:"show_suspend"`
	ShowHibernate    bool   `toml:"show_hibernate"`
	ShowReboot       bool   `toml:"show_reboot"`
	ShowShutdown     bool   `toml:"show_shutdown"`
	ConfirmLogout    bool   `toml:"confirm_logout"`
	ConfirmSuspend   bool   `toml:"confirm_suspend"`
	ConfirmHibernate bool   `toml:"confirm_hibernate"`
	ConfirmReboot    bool   `toml:"confirm_reboot"`
	ConfirmShutdown  bool   `toml:"confirm_shutdown"`
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

// RadioConfig за radio
type RadioConfig struct {
	Enabled       bool              `toml:"enabled"`
	Volume        int               `toml:"volume"`
	RadioStations map[string]string `toml:"stations"`
}

// WifiConfig за wifi
type WifiConfig struct {
	Enabled    bool   `toml:"enabled"`
	TestHost   string `toml:"test_host"`
	TestCount  int    `toml:"test_count"`
	TestWait   int    `toml:"test_wait"`
	ShowNotify bool   `toml:"show_notify"`
}

// MpcConfig за mpc
type MpcConfig struct {
	Enabled              bool   `toml:"enabled"`
	CurrentPlaylistCache string `toml:"current_playlist_cache"`
}

// AudioRecordConfig за audio recording
type AudioRecordConfig struct {
	Enabled    bool   `toml:"enabled"`
	SaveDir    string `toml:"save_dir"`
	FilePrefix string `toml:"file_prefix"`
	Format     string `toml:"format"`
	Quality    string `toml:"quality"`
}

// VideoRecordConfig за video recording
type VideoRecordConfig struct {
	Enabled     bool   `toml:"enabled"`
	SaveDir     string `toml:"save_dir"`
	FilePrefix  string `toml:"file_prefix"`
	Format      string `toml:"format"`
	Quality     string `toml:"quality"`
	RecordAudio bool   `toml:"record_audio"`
	ShowNotify  bool   `toml:"show_notify"`

	// Platform specific settings
	X11     VideoRecordX11Config     `toml:"x11"`
	Wayland VideoRecordWaylandConfig `toml:"wayland"`
}

// VideoRecordX11Config за X11 specific settings
type VideoRecordX11Config struct {
	Framerate  int    `toml:"framerate"`
	OutputFPS  int    `toml:"output_fps"`
	Preset     string `toml:"preset"`
	VideoCodec string `toml:"video_codec"`
	AudioCodec string `toml:"audio_codec"`
}

// VideoRecordWaylandConfig за Wayland specific settings
type VideoRecordWaylandConfig struct {
	Framerate  int    `toml:"framerate"`
	OutputFPS  int    `toml:"output_fps"`
	Preset     string `toml:"preset"`
	VideoCodec string `toml:"video_codec"`
	AudioCodec string `toml:"audio_codec"`
}

// CommandsConfigFile за четене от TOML
type CommandsConfigFile struct {
	Power       PowerConfigFile       `toml:"power"`
	Screenshot  ScreenshotConfigFile  `toml:"screenshot"`
	Radio       RadioConfigFile       `toml:"radio"`
	Wifi        WifiConfigFile        `toml:"wifi"`
	Mpc         MpcConfigFile         `toml:"mpc"`
	AudioRecord AudioRecordConfigFile `toml:"audiorecord"`
	VideoRecord VideoRecordConfigFile `toml:"videorecord"`
}

// PowerConfigFile с pointers
type PowerConfigFile struct {
	Enabled          *bool   `toml:"enabled"`
	ShowLogout       *bool   `toml:"show_logout"`
	ShowSuspend      *bool   `toml:"show_suspend"`
	ShowHibernate    *bool   `toml:"show_hibernate"`
	ShowReboot       *bool   `toml:"show_reboot"`
	ShowShutdown     *bool   `toml:"show_shutdown"`
	ConfirmLogout    *bool   `toml:"confirm_logout"`
	ConfirmSuspend   *bool   `toml:"confirm_suspend"`
	ConfirmHibernate *bool   `toml:"confirm_hibernate"`
	ConfirmReboot    *bool   `toml:"confirm_reboot"`
	ConfirmShutdown  *bool   `toml:"confirm_shutdown"`
	LogoutCommand    *string `toml:"logout_command"`
	SuspendCommand   *string `toml:"suspend_command"`
	HibernateCommand *string `toml:"hibernate_command"`
	RebootCommand    *string `toml:"reboot_command"`
	ShutdownCommand  *string `toml:"shutdown_command"`
}

// ScreenshotConfigFile с pointers
type ScreenshotConfigFile struct {
	Enabled    *bool   `toml:"enabled"`
	SaveDir    *string `toml:"save_dir"`
	FilePrefix *string `toml:"file_prefix"`
}

// RadioConfigFile с pointers
type RadioConfigFile struct {
	Enabled       *bool             `toml:"enabled"`
	Volume        *int              `toml:"volume"`
	RadioStations map[string]string `toml:"stations"`
}

// WifiConfigFile с pointers
type WifiConfigFile struct {
	Enabled    *bool   `toml:"enabled"`
	TestHost   *string `toml:"test_host"`
	TestCount  *int    `toml:"test_count"`
	TestWait   *int    `toml:"test_wait"`
	ShowNotify *bool   `toml:"show_notify"`
}

// MpcConfigFile с pointers
type MpcConfigFile struct {
	Enabled              *bool   `toml:"enabled"`
	CurrentPlaylistCache *string `toml:"current_playlist_cache"`
}

// AudioRecordConfigFile с pointers
type AudioRecordConfigFile struct {
	Enabled    *bool   `toml:"enabled"`
	SaveDir    *string `toml:"save_dir"`
	FilePrefix *string `toml:"file_prefix"`
	Format     *string `toml:"format"`
	Quality    *string `toml:"quality"`
}

// VideoRecordConfigFile с pointers
type VideoRecordConfigFile struct {
	Enabled     *bool   `toml:"enabled"`
	SaveDir     *string `toml:"save_dir"`
	FilePrefix  *string `toml:"file_prefix"`
	Format      *string `toml:"format"`
	Quality     *string `toml:"quality"`
	RecordAudio *bool   `toml:"record_audio"`
	ShowNotify  *bool   `toml:"show_notify"`

	// Platform specific settings
	X11     VideoRecordX11ConfigFile     `toml:"x11"`
	Wayland VideoRecordWaylandConfigFile `toml:"wayland"`
}

// VideoRecordX11ConfigFile с pointers
type VideoRecordX11ConfigFile struct {
	Framerate  *int    `toml:"framerate"`
	OutputFPS  *int    `toml:"output_fps"`
	Preset     *string `toml:"preset"`
	VideoCodec *string `toml:"video_codec"`
	AudioCodec *string `toml:"audio_codec"`
}

// VideoRecordWaylandConfigFile с pointers
type VideoRecordWaylandConfigFile struct {
	Framerate  *int    `toml:"framerate"`
	OutputFPS  *int    `toml:"output_fps"`
	Preset     *string `toml:"preset"`
	VideoCodec *string `toml:"video_codec"`
	AudioCodec *string `toml:"audio_codec"`
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
	mergeLauncherConfigs(&merged.Launchers, &userCfg.Launchers)

	// Merge command settings
	mergePowerConfig(&merged.Commands.Power, &userCfg.Commands.Power)
	mergeScreenshotConfig(&merged.Commands.Screenshot, &userCfg.Commands.Screenshot)
	mergeRadioConfig(&merged.Commands.Radio, &userCfg.Commands.Radio)
	mergeWifiConfig(&merged.Commands.Wifi, &userCfg.Commands.Wifi)
	mergeMpcConfig(&merged.Commands.Mpc, &userCfg.Commands.Mpc)
	mergeAudioRecordConfig(&merged.Commands.AudioRecord, &userCfg.Commands.AudioRecord)
	mergeVideoRecordConfig(&merged.Commands.VideoRecord, &userCfg.Commands.VideoRecord)

	return &merged
}

// mergePowerConfig мерджва power конфигурация
func mergePowerConfig(merged *PowerConfig, user *PowerConfigFile) {
	if user.Enabled != nil {
		merged.Enabled = *user.Enabled
	}
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

// mergeRadioConfig мерджва radio конфигурация
func mergeRadioConfig(merged *RadioConfig, user *RadioConfigFile) {
	if user.Enabled != nil {
		merged.Enabled = *user.Enabled
	}
	if user.Volume != nil {
		merged.Volume = *user.Volume
	}
	if len(user.RadioStations) > 0 {
		maps.Copy(merged.RadioStations, user.RadioStations)
	}
}

// mergeWifiConfig мерджва wifi конфигурация
func mergeWifiConfig(merged *WifiConfig, user *WifiConfigFile) {
	if user.Enabled != nil {
		merged.Enabled = *user.Enabled
	}
	if user.TestHost != nil && *user.TestHost != "" {
		merged.TestHost = *user.TestHost
	}
	if user.TestCount != nil {
		merged.TestCount = *user.TestCount
	}
	if user.TestWait != nil {
		merged.TestWait = *user.TestWait
	}
	if user.ShowNotify != nil {
		merged.ShowNotify = *user.ShowNotify
	}
}

// mergeMpcConfig мерджва mpc конфигурация
func mergeMpcConfig(merged *MpcConfig, user *MpcConfigFile) {
	if user.Enabled != nil {
		merged.Enabled = *user.Enabled
	}
	if user.CurrentPlaylistCache != nil && *user.CurrentPlaylistCache != "" {
		merged.CurrentPlaylistCache = *user.CurrentPlaylistCache
	}
}

// mergeAudioRecordConfig мерджва audiorecord конфигурация
func mergeAudioRecordConfig(merged *AudioRecordConfig, user *AudioRecordConfigFile) {
	if user.Enabled != nil {
		merged.Enabled = *user.Enabled
	}
	if user.SaveDir != nil && *user.SaveDir != "" {
		merged.SaveDir = *user.SaveDir
	}
	if user.FilePrefix != nil && *user.FilePrefix != "" {
		merged.FilePrefix = *user.FilePrefix
	}
	if user.Format != nil && *user.Format != "" {
		merged.Format = *user.Format
	}
	if user.Quality != nil && *user.Quality != "" {
		merged.Quality = *user.Quality
	}
}

// mergeVideoRecordConfig мерджва videorecord конфигурация
func mergeVideoRecordConfig(merged *VideoRecordConfig, user *VideoRecordConfigFile) {
	if user.Enabled != nil {
		merged.Enabled = *user.Enabled
	}
	if user.SaveDir != nil && *user.SaveDir != "" {
		merged.SaveDir = *user.SaveDir
	}
	if user.FilePrefix != nil && *user.FilePrefix != "" {
		merged.FilePrefix = *user.FilePrefix
	}
	if user.Format != nil && *user.Format != "" {
		merged.Format = *user.Format
	}
	if user.Quality != nil && *user.Quality != "" {
		merged.Quality = *user.Quality
	}
	if user.RecordAudio != nil {
		merged.RecordAudio = *user.RecordAudio
	}
	if user.ShowNotify != nil {
		merged.ShowNotify = *user.ShowNotify
	}

	// Merge X11 settings
	mergeVideoRecordX11Config(&merged.X11, &user.X11)

	// Merge Wayland settings
	mergeVideoRecordWaylandConfig(&merged.Wayland, &user.Wayland)
}

// mergeVideoRecordX11Config мерджва X11 конфигурация
func mergeVideoRecordX11Config(merged *VideoRecordX11Config, user *VideoRecordX11ConfigFile) {
	if user.Framerate != nil {
		merged.Framerate = *user.Framerate
	}
	if user.OutputFPS != nil {
		merged.OutputFPS = *user.OutputFPS
	}
	if user.Preset != nil && *user.Preset != "" {
		merged.Preset = *user.Preset
	}
	if user.VideoCodec != nil && *user.VideoCodec != "" {
		merged.VideoCodec = *user.VideoCodec
	}
	if user.AudioCodec != nil && *user.AudioCodec != "" {
		merged.AudioCodec = *user.AudioCodec
	}
}

// mergeVideoRecordWaylandConfig мерджва Wayland конфигурация
func mergeVideoRecordWaylandConfig(merged *VideoRecordWaylandConfig, user *VideoRecordWaylandConfigFile) {
	if user.Framerate != nil {
		merged.Framerate = *user.Framerate
	}
	if user.OutputFPS != nil {
		merged.OutputFPS = *user.OutputFPS
	}
	if user.Preset != nil && *user.Preset != "" {
		merged.Preset = *user.Preset
	}
	if user.VideoCodec != nil && *user.VideoCodec != "" {
		merged.VideoCodec = *user.VideoCodec
	}
	if user.AudioCodec != nil && *user.AudioCodec != "" {
		merged.AudioCodec = *user.AudioCodec
	}
}

// Get връща глобалния config (lazy load)
func Get() *Config {
	if globalConfig == nil {
		globalConfig, _ = Load()
	}
	return globalConfig
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

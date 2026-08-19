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
	Theme             ThemeConfig               `toml:"theme"`
	Commands          map[string]map[string]any `toml:"commands"`
}

// ThemeConfig selects the look of the built-in TUI. Name is a built-in preset
// ("default", "mono") or a file in ~/.config/ql/themes/<name>.yaml; the palette
// itself lives in that YAML file, not here. Border and Icons override the
// preset's structural knobs without touching the palette.
type ThemeConfig struct {
	Name   string `toml:"name"`
	Border string `toml:"border,omitempty"`
	Icons  string `toml:"icons,omitempty"`
}

// ModuleGroup represents a group of related modules
type ModuleGroup struct {
	Name    string   `toml:"name"`
	Enabled bool     `toml:"enabled"`
	Modules []string `toml:"modules"`
}

// LauncherConfig represents launcher-specific configuration
type LauncherConfig struct {
	// Command is the executable to run, when it is not simply the launcher's
	// name on PATH. A menu installed outside the session's PATH cannot be named
	// any other way: `--launcher` takes a name, and a process started from a
	// compositor keybind carries only the login PATH — which on the machine this
	// was written for is ~/.local/bin, ~/bin, /usr/local/bin, /usr/bin, /bin.
	// A rofi built into ~/clipack/bin is invisible to every one of them.
	//
	// A leading ~ is expanded, because a config file is where people write one.
	Command string   `toml:"command,omitempty"`
	Args    []string `toml:"args"`
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
	meta, err := toml.DecodeFile(userConfigPath, &userCfg)
	if err != nil {
		// A broken user config must not brick the launcher — it is usually
		// bound to a key, where a hard exit looks like nothing happened at
		// all. Complain where it can be seen and run on the defaults.
		fmt.Fprintf(os.Stderr, "ql: ignoring malformed %s: %v\n", userConfigPath, err)
		return &defaultCfg, nil
	}

	mergedCfg := mergeConfigs(defaultCfg, userCfg, meta)
	return &mergedCfg, nil
}

// InitUserConfig creates user config from default
func InitUserConfig() error {
	configPath := GetUserConfigPath()
	configDir := filepath.Dir(configPath)

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 0600, not 0644: the config carries secret-bearing fields (the mpc
	// password is documented right in the default file), so it must not be
	// created world-readable.
	if err := os.WriteFile(configPath, []byte(defaultConfig), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetUserConfigPath returns the path to user config
func GetUserConfigPath() string {
	home := os.Getenv("HOME")
	return filepath.Join(home, ".config", "ql", "config.toml")
}

// mergeConfigs deep merges user config into default config. meta says which
// keys the user actually WROTE — the only way to tell `enabled = false` from
// an omitted key, since both decode to the zero value.
func mergeConfigs(defaultCfg, userCfg Config, meta toml.MetaData) Config {
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

	// Merge maps. Groups merge FIELD BY FIELD: a whole-value copy meant that
	// [module_groups.system] with only `name = "Sys"` wiped the group's
	// modules and disabled it — partial override was impossible.
	if result.ModuleGroups == nil {
		result.ModuleGroups = make(map[string]ModuleGroup)
	}
	for key, userGroup := range userCfg.ModuleGroups {
		group, exists := result.ModuleGroups[key]
		if !exists {
			result.ModuleGroups[key] = userGroup
			continue
		}
		if meta.IsDefined("module_groups", key, "name") {
			group.Name = userGroup.Name
		}
		if meta.IsDefined("module_groups", key, "enabled") {
			group.Enabled = userGroup.Enabled
		}
		if meta.IsDefined("module_groups", key, "modules") {
			group.Modules = userGroup.Modules
		}
		result.ModuleGroups[key] = group
	}

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
	// Booleans only when the key was written: the old `user || default`
	// with a true default made `enabled = false` impossible to express.
	if meta.IsDefined("notifications", "enabled") {
		result.Notifications.Enabled = userCfg.Notifications.Enabled
	}
	if meta.IsDefined("notifications", "show_in_terminal") {
		result.Notifications.ShowInTerminal = userCfg.Notifications.ShowInTerminal
	}

	// Merge theme (structural strings only; the palette is in the theme file).
	if userCfg.Theme.Name != "" {
		result.Theme.Name = userCfg.Theme.Name
	}
	if userCfg.Theme.Border != "" {
		result.Theme.Border = userCfg.Theme.Border
	}
	if userCfg.Theme.Icons != "" {
		result.Theme.Icons = userCfg.Theme.Icons
	}

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

func (c *Config) GetBrowser() string {
	if c.Browser == "" {
		return "firefox"
	}
	return c.Browser
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

// GetTheme returns the TUI theme selection, defaulting the preset name when it
// was left unset.
func (c *Config) GetTheme() ThemeConfig {
	t := c.Theme
	if t.Name == "" {
		t.Name = "default"
	}
	return t
}

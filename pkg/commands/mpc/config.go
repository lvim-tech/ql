package mpc

// Config за mpc модула
type Config struct {
	Enabled              bool   `toml:"enabled"`
	CurrentPlaylistCache string `toml:"current_playlist_cache"`
}

// ConfigFile за четене от TOML
type ConfigFile struct {
	Enabled              *bool   `toml:"enabled"`
	CurrentPlaylistCache *string `toml:"current_playlist_cache"`
}

// MergeConfig мерджва mpc конфигурация
func MergeConfig(merged *Config, user *ConfigFile) {
	if user.Enabled != nil {
		merged.Enabled = *user.Enabled
	}

	if user.CurrentPlaylistCache != nil && *user.CurrentPlaylistCache != "" {
		merged.CurrentPlaylistCache = *user.CurrentPlaylistCache
	}
}

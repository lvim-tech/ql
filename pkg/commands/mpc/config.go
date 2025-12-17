package mpc

// Config за mpc
type Config struct {
	Enabled              bool   `toml:"enabled" mapstructure:"enabled"`
	CurrentPlaylistCache string `toml:"current_playlist_cache" mapstructure:"current_playlist_cache"`
}

// DefaultConfig връща default настройки
func DefaultConfig() Config {
	return Config{
		Enabled:              true,
		CurrentPlaylistCache: "~/.cache/ql_current_playlist",
	}
}

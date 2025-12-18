package mpc

type Config struct {
	Enabled              bool   `mapstructure:"enabled"`
	ConnectionType       string `mapstructure:"connection_type"` // "tcp" or "socket"
	Host                 string `mapstructure:"host"`
	Port                 string `mapstructure:"port"`
	Socket               string `mapstructure:"socket"`
	Password             string `mapstructure:"password"`
	CurrentPlaylistCache string `mapstructure:"current_playlist_cache"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:              true,
		ConnectionType:       "tcp",
		Host:                 "localhost",
		Port:                 "6600",
		Socket:               "~/.config/mpd/socket",
		Password:             "",
		CurrentPlaylistCache: "~/.cache/ql/mpc_current_playlist.txt",
	}
}

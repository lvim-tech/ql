package radio

// Config за radio
type Config struct {
	Enabled       bool              `toml:"enabled" mapstructure:"enabled"`
	Volume        int64             `toml:"volume" mapstructure:"volume"`
	RadioStations map[string]string `toml:"stations" mapstructure:"stations"`
}

// DefaultConfig връща default настройки
func DefaultConfig() Config {
	return Config{
		Enabled: true,
		Volume:  70,
		RadioStations: map[string]string{
			"Jazz FM":    "http://live.musictradio.com/JazzFMHigh",
			"Classic FM": "http://media-ice.musicradio. com/ClassicFMMP3",
			"Smooth FM":  "http://live.musictradio.com/SmoothFMHigh",
		},
	}
}

package weather

// Config holds weather module configuration
type Config struct {
	Enabled   bool     `toml:"enabled" mapstructure:"enabled"`
	Locations []string `toml:"locations" mapstructure:"locations"`
	Options   string   `toml:"options" mapstructure:"options"`
	Timeout   int      `toml:"timeout" mapstructure:"timeout"` // Timeout in seconds
}

// DefaultConfig returns default weather configuration
func DefaultConfig() Config {
	return Config{
		Enabled: true,
		Locations: []string{
			"Sofia",
			"London",
			"New York",
			"Tokyo",
		},
		Options: "",
		Timeout: 30,
	}
}

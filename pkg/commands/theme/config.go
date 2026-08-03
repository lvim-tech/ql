package theme

// Config holds the theme module settings.
type Config struct {
	Enabled bool `mapstructure:"enabled"`
	// Bin is the themer binary; a bare name resolves through PATH.
	Bin string `mapstructure:"bin"`
}

// DefaultConfig returns the built-in defaults.
func DefaultConfig() Config {
	return Config{
		Enabled: true,
		Bin:     "themer",
	}
}

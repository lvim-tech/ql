package sysinfo

// Config holds the sysinfo module settings.
type Config struct {
	Enabled bool `mapstructure:"enabled"`
	// Lines caps the process listings.
	Lines int `mapstructure:"lines"`
}

// DefaultConfig returns the built-in defaults.
func DefaultConfig() Config {
	return Config{
		Enabled: true,
		Lines:   25,
	}
}

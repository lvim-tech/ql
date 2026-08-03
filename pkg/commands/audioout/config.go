package audioout

// Config holds the audioout module settings.
type Config struct {
	Enabled bool `mapstructure:"enabled"`
}

// DefaultConfig returns the built-in defaults.
func DefaultConfig() Config {
	return Config{Enabled: true}
}

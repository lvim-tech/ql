package clipboard

// Config represents clipboard module configuration
type Config struct {
	Enabled  bool `mapstructure:"enabled"`
	MaxItems int  `mapstructure:"max_items"`
}

// DefaultConfig returns default clipboard configuration
func DefaultConfig() Config {
	return Config{
		Enabled:  true,
		MaxItems: 50,
	}
}

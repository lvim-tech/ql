package man

// Config represents man module configuration
type Config struct {
	Enabled          bool   `mapstructure:"enabled"`
	ShowDescriptions bool   `mapstructure:"show_descriptions"`
	MaxResults       int    `mapstructure:"max_results"`
	Terminal         string `mapstructure:"terminal"`
}

// DefaultConfig returns default man configuration
func DefaultConfig() Config {
	return Config{
		Enabled:          true,
		ShowDescriptions: true,
		MaxResults:       100,
		Terminal:         "",
	}
}

package bookman

// Source defines a bookmark/quickmark source in config.toml
type Source struct {
	Name   string `toml:"name" mapstructure:"name"`
	Path   string `toml:"path" mapstructure:"path"`
	Format string `toml:"format" mapstructure:"format"`
}

// Config holds bookman module configuration
type Config struct {
	Enabled bool     `toml:"enabled" mapstructure:"enabled"`
	Sources []Source `toml:"sources" mapstructure:"sources"`
}

// DefaultConfig returns default bookman configuration
func DefaultConfig() Config {
	return Config{
		Enabled: true,
		Sources: []Source{
			{
				Name:   "Qutebrowser Quickmarks",
				Path:   "~/.config/qutebrowser/quickmarks",
				Format: "qutebrowser_quickmarks",
			},
			{
				Name:   "Qutebrowser Bookmarks",
				Path:   "~/.config/qutebrowser/bookmarks/urls",
				Format: "qutebrowser_bookmarks",
			},
		},
	}
}

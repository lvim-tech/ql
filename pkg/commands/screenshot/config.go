package screenshot

// Config holds the screenshot module settings.
type Config struct {
	Enabled    bool   `toml:"enabled" mapstructure:"enabled"`
	SaveDir    string `toml:"save_dir" mapstructure:"save_dir"`
	FilePrefix string `toml:"file_prefix" mapstructure:"file_prefix"`
}

// DefaultConfig returns the built-in defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:    true,
		SaveDir:    "~/Pictures/Screenshots",
		FilePrefix: "screenshot",
	}
}

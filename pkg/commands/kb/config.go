package kb

// Config holds the kb module settings.
type Config struct {
	Enabled bool `mapstructure:"enabled"`
	// Layouts OVERRIDES detection; leave empty and the module reads what
	// the session has configured (mango's config file, Hyprland/sway IPC,
	// setxkbmap, localectl — in that order). Order matters for the
	// backends that switch by index.
	Layouts []string `mapstructure:"layouts"`
}

// DefaultConfig returns the built-in defaults.
func DefaultConfig() Config {
	return Config{Enabled: true}
}

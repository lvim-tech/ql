package pass

// Config holds the pass module settings.
type Config struct {
	Enabled bool `mapstructure:"enabled"`
	// Store is the password-store root.
	Store string `mapstructure:"store"`
	// TypeDelayMs waits for the menu to close and focus to return before
	// wtype starts, so the password lands in the intended field.
	TypeDelayMs int `mapstructure:"type_delay_ms"`
	// Keyforge is the generator/audit TUI opened by "Open keyforge".
	Keyforge string `mapstructure:"keyforge"`
}

// DefaultConfig returns the built-in defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:     true,
		Store:       "~/.password-store",
		TypeDelayMs: 200,
		Keyforge:    "keyforge",
	}
}

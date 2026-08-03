package updates

// Config holds the updates module settings.
type Config struct {
	Enabled bool `mapstructure:"enabled"`
	// ListCmd prints the pending updates (empty output = nothing pending).
	ListCmd string `mapstructure:"list_cmd"`
	// UpdateCmd installs them; runs interactively in a terminal.
	UpdateCmd string `mapstructure:"update_cmd"`
}

// DefaultConfig returns the built-in defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:   true,
		ListCmd:   "zypper -q lu",
		UpdateCmd: "sudo zypper up",
	}
}

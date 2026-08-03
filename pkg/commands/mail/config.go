package mail

// Config holds the mail module settings.
type Config struct {
	Enabled bool `mapstructure:"enabled"`
	// StatusCmd prints the unread summary; the default reuses the user's
	// own script rather than reimplementing its mailbox map.
	StatusCmd string `mapstructure:"status_cmd"`
	// SyncCmd fetches mail; runs visibly in a terminal.
	SyncCmd string `mapstructure:"sync_cmd"`
	// ClientCmd is the mail reader, also run in a terminal.
	ClientCmd string `mapstructure:"client_cmd"`
}

// DefaultConfig returns the built-in defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:   true,
		StatusCmd: "~/bin/mailstatus",
		SyncCmd:   "mbsync -a",
		ClientCmd: "neomutt",
	}
}

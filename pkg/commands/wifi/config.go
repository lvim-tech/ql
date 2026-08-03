package wifi

// Config holds the wifi module settings.
type Config struct {
	Enabled    bool   `toml:"enabled" mapstructure:"enabled"`
	TestHost   string `toml:"test_host" mapstructure:"test_host"`
	TestCount  int64  `toml:"test_count" mapstructure:"test_count"`
	TestWait   int64  `toml:"test_wait" mapstructure:"test_wait"`
	ShowNotify bool   `toml:"show_notify" mapstructure:"show_notify"`
}

// DefaultConfig returns the built-in defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:    true,
		TestHost:   "8.8.8.8",
		TestCount:  3,
		TestWait:   2,
		ShowNotify: true,
	}
}

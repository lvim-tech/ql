package wifi

// Config за wifi модула
type Config struct {
	Enabled    bool   `toml:"enabled"`
	TestHost   string `toml:"test_host"`
	TestCount  int    `toml:"test_count"`
	TestWait   int    `toml:"test_wait"`
	ShowNotify bool   `toml:"show_notify"`
}

// ConfigFile за четене от TOML
type ConfigFile struct {
	Enabled    *bool   `toml:"enabled"`
	TestHost   *string `toml:"test_host"`
	TestCount  *int    `toml:"test_count"`
	TestWait   *int    `toml:"test_wait"`
	ShowNotify *bool   `toml:"show_notify"`
}

// MergeConfig мерджва wifi конфигурация
func MergeConfig(merged *Config, user *ConfigFile) {
	if user.Enabled != nil {
		merged.Enabled = *user.Enabled
	}

	if user.TestHost != nil && *user.TestHost != "" {
		merged.TestHost = *user.TestHost
	}

	if user.TestCount != nil {
		merged.TestCount = *user.TestCount
	}

	if user.TestWait != nil {
		merged.TestWait = *user.TestWait
	}

	if user.ShowNotify != nil {
		merged.ShowNotify = *user.ShowNotify
	}
}

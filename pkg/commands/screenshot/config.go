package screenshot

// Config за screenshot
type Config struct {
	Enabled    bool   `toml:"enabled"`
	SaveDir    string `toml:"save_dir"`
	FilePrefix string `toml:"file_prefix"`
}

// ConfigFile е за четене от TOML
type ConfigFile struct {
	Enabled    *bool   `toml:"enabled"`
	SaveDir    *string `toml:"save_dir"`
	FilePrefix *string `toml:"file_prefix"`
}

// Merge мерджва screenshot конфигурация
func (c *Config) Merge(user *ConfigFile) {
	// Enabled
	if user.Enabled != nil {
		c.Enabled = *user.Enabled
	}

	if user.SaveDir != nil && *user.SaveDir != "" {
		c.SaveDir = *user.SaveDir
	}
	if user.FilePrefix != nil && *user.FilePrefix != "" {
		c.FilePrefix = *user.FilePrefix
	}
}

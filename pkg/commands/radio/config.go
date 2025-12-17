package radio

import "maps"

// Config за radio модула
type Config struct {
	Enabled       bool              `toml:"enabled"`
	Volume        int               `toml:"volume"`
	RadioStations map[string]string `toml:"stations"`
}

// ConfigFile за четене от TOML
type ConfigFile struct {
	Enabled       *bool             `toml:"enabled"`
	Volume        *int              `toml:"volume"`
	RadioStations map[string]string `toml:"stations"`
}

// Merge мерджва radio конфигурация
func (c *Config) Merge(user *ConfigFile) {
	// Enabled
	if user.Enabled != nil {
		c.Enabled = *user.Enabled
	}

	// Volume
	if user.Volume != nil {
		c.Volume = *user.Volume
	}

	// Radio stations (merge map)
	if len(user.RadioStations) > 0 {
		maps.Copy(c.RadioStations, user.RadioStations)
	}
}

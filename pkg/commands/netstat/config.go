package netstat

// Config represents netstat module configuration
type Config struct {
	Enabled        bool `toml:"enabled"`
	ShowNotify     bool `toml:"show_notify"`
	UpdateInterval int  `toml:"update_interval"` // seconds for live monitor
	PreferVnstat   bool `toml:"prefer_vnstat"`   // prefer vnstat over /sys/class/net
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		Enabled:        true,
		ShowNotify:     true,
		UpdateInterval: 1,
		PreferVnstat:   true,
	}
}

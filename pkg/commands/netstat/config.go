package netstat

// Config represents netstat module configuration.
//
// show_notify and update_interval used to live here too. Nothing read them:
// netstat renders into a GUI window and has no live monitor, so both described
// behaviour that does not exist. They are gone rather than left to imply it.
type Config struct {
	Enabled      bool `toml:"enabled"`
	PreferVnstat bool `toml:"prefer_vnstat"` // prefer vnstat over /sys/class/net
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		Enabled:      true,
		PreferVnstat: true,
	}
}

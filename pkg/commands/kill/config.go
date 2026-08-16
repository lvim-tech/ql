package kill

// Config represents kill module configuration.
//
// show_user_processes used to sit here as well. Nothing read it, and it could
// not be given a coherent meaning: show_all_processes = false already means
// "only my own processes", so the two were the same switch under two names.
type Config struct {
	Enabled          bool     `mapstructure:"enabled"`
	ShowAllProcesses bool     `mapstructure:"show_all_processes"`
	ExcludeProcesses []string `mapstructure:"exclude_processes"`
	ConfirmKill      bool     `mapstructure:"confirm_kill"`
}

// DefaultConfig returns default kill configuration
func DefaultConfig() Config {
	return Config{
		Enabled:          true,
		ShowAllProcesses: false,
		ExcludeProcesses: []string{
			"systemd",
			"init",
			"kthreadd",
		},
		ConfirmKill: true,
	}
}

package kill

// Config represents kill module configuration
type Config struct {
	Enabled           bool     `mapstructure:"enabled"`
	ShowUserProcesses bool     `mapstructure:"show_user_processes"`
	ShowAllProcesses  bool     `mapstructure:"show_all_processes"`
	ExcludeProcesses  []string `mapstructure:"exclude_processes"`
	ConfirmKill       bool     `mapstructure:"confirm_kill"`
}

// DefaultConfig returns default kill configuration
func DefaultConfig() Config {
	return Config{
		Enabled:           true,
		ShowUserProcesses: true,
		ShowAllProcesses:  false,
		ExcludeProcesses: []string{
			"systemd",
			"init",
			"kthreadd",
		},
		ConfirmKill: true,
	}
}

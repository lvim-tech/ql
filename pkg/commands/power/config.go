package power

// Config за power management
type Config struct {
	Enabled          bool   `toml:"enabled" mapstructure:"enabled"`
	ShowLogout       bool   `toml:"show_logout" mapstructure:"show_logout"`
	ShowSuspend      bool   `toml:"show_suspend" mapstructure:"show_suspend"`
	ShowHibernate    bool   `toml:"show_hibernate" mapstructure:"show_hibernate"`
	ShowReboot       bool   `toml:"show_reboot" mapstructure:"show_reboot"`
	ShowShutdown     bool   `toml:"show_shutdown" mapstructure:"show_shutdown"`
	ConfirmLogout    bool   `toml:"confirm_logout" mapstructure:"confirm_logout"`
	ConfirmSuspend   bool   `toml:"confirm_suspend" mapstructure:"confirm_suspend"`
	ConfirmHibernate bool   `toml:"confirm_hibernate" mapstructure:"confirm_hibernate"`
	ConfirmReboot    bool   `toml:"confirm_reboot" mapstructure:"confirm_reboot"`
	ConfirmShutdown  bool   `toml:"confirm_shutdown" mapstructure:"confirm_shutdown"`
	LogoutCommand    string `toml:"logout_command" mapstructure:"logout_command"`
	SuspendCommand   string `toml:"suspend_command" mapstructure:"suspend_command"`
	HibernateCommand string `toml:"hibernate_command" mapstructure:"hibernate_command"`
	RebootCommand    string `toml:"reboot_command" mapstructure:"reboot_command"`
	ShutdownCommand  string `toml:"shutdown_command" mapstructure:"shutdown_command"`
}

// DefaultConfig връща default настройки
func DefaultConfig() Config {
	return Config{
		Enabled:          true,
		ShowLogout:       true,
		ShowSuspend:      true,
		ShowHibernate:    true,
		ShowReboot:       true,
		ShowShutdown:     true,
		ConfirmLogout:    false,
		ConfirmSuspend:   false,
		ConfirmHibernate: true,
		ConfirmReboot:    true,
		ConfirmShutdown:  true,
		LogoutCommand:    "loginctl terminate-user $USER",
		SuspendCommand:   "systemctl suspend",
		HibernateCommand: "systemctl hibernate",
		RebootCommand:    "systemctl reboot",
		ShutdownCommand:  "systemctl poweroff",
	}
}

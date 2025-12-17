package power

// Config за power management
type Config struct {
	Enabled bool `toml:"enabled"`

	// Show options
	ShowLogout    bool `toml:"show_logout"`
	ShowSuspend   bool `toml:"show_suspend"`
	ShowHibernate bool `toml:"show_hibernate"`
	ShowReboot    bool `toml:"show_reboot"`
	ShowShutdown  bool `toml:"show_shutdown"`

	// Confirm options
	ConfirmLogout    bool `toml:"confirm_logout"`
	ConfirmSuspend   bool `toml:"confirm_suspend"`
	ConfirmHibernate bool `toml:"confirm_hibernate"`
	ConfirmReboot    bool `toml:"confirm_reboot"`
	ConfirmShutdown  bool `toml:"confirm_shutdown"`

	// Custom commands
	LogoutCommand    string `toml:"logout_command"`
	SuspendCommand   string `toml:"suspend_command"`
	HibernateCommand string `toml:"hibernate_command"`
	RebootCommand    string `toml:"reboot_command"`
	ShutdownCommand  string `toml:"shutdown_command"`
}

// ConfigFile е за четене от TOML (с pointers за optional полета)
type ConfigFile struct {
	Enabled *bool `toml:"enabled"`

	// Show options
	ShowLogout    *bool `toml:"show_logout"`
	ShowSuspend   *bool `toml:"show_suspend"`
	ShowHibernate *bool `toml:"show_hibernate"`
	ShowReboot    *bool `toml:"show_reboot"`
	ShowShutdown  *bool `toml:"show_shutdown"`

	// Confirm options
	ConfirmLogout    *bool `toml:"confirm_logout"`
	ConfirmSuspend   *bool `toml:"confirm_suspend"`
	ConfirmHibernate *bool `toml:"confirm_hibernate"`
	ConfirmReboot    *bool `toml:"confirm_reboot"`
	ConfirmShutdown  *bool `toml:"confirm_shutdown"`

	// Custom commands
	LogoutCommand    *string `toml:"logout_command"`
	SuspendCommand   *string `toml:"suspend_command"`
	HibernateCommand *string `toml:"hibernate_command"`
	RebootCommand    *string `toml:"reboot_command"`
	ShutdownCommand  *string `toml:"shutdown_command"`
}

// Merge мерджва power конфигурация
func (c *Config) Merge(user *ConfigFile) {
	// Enabled
	if user.Enabled != nil {
		c.Enabled = *user.Enabled
	}

	// Show options
	if user.ShowLogout != nil {
		c.ShowLogout = *user.ShowLogout
	}
	if user.ShowSuspend != nil {
		c.ShowSuspend = *user.ShowSuspend
	}
	if user.ShowHibernate != nil {
		c.ShowHibernate = *user.ShowHibernate
	}
	if user.ShowReboot != nil {
		c.ShowReboot = *user.ShowReboot
	}
	if user.ShowShutdown != nil {
		c.ShowShutdown = *user.ShowShutdown
	}

	// Confirm options
	if user.ConfirmLogout != nil {
		c.ConfirmLogout = *user.ConfirmLogout
	}
	if user.ConfirmSuspend != nil {
		c.ConfirmSuspend = *user.ConfirmSuspend
	}
	if user.ConfirmHibernate != nil {
		c.ConfirmHibernate = *user.ConfirmHibernate
	}
	if user.ConfirmReboot != nil {
		c.ConfirmReboot = *user.ConfirmReboot
	}
	if user.ConfirmShutdown != nil {
		c.ConfirmShutdown = *user.ConfirmShutdown
	}

	// Commands
	if user.LogoutCommand != nil && *user.LogoutCommand != "" {
		c.LogoutCommand = *user.LogoutCommand
	}
	if user.SuspendCommand != nil && *user.SuspendCommand != "" {
		c.SuspendCommand = *user.SuspendCommand
	}
	if user.HibernateCommand != nil && *user.HibernateCommand != "" {
		c.HibernateCommand = *user.HibernateCommand
	}
	if user.RebootCommand != nil && *user.RebootCommand != "" {
		c.RebootCommand = *user.RebootCommand
	}
	if user.ShutdownCommand != nil && *user.ShutdownCommand != "" {
		c.ShutdownCommand = *user.ShutdownCommand
	}
}

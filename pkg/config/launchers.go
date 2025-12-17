package config

// LauncherConfig за всеки launcher
type LauncherConfig struct {
	Dmenu  LauncherCommand `toml:"dmenu"`
	Rofi   LauncherCommand `toml:"rofi"`
	Fzf    LauncherCommand `toml:"fzf"`
	Bemenu LauncherCommand `toml:"bemenu"`
	Fuzzel LauncherCommand `toml:"fuzzel"`
}

// LauncherCommand описва как да се стартира launcher
type LauncherCommand struct {
	Args []string `toml:"args"`
}

// GetLauncherCommand връща команда за конкретен launcher
func (c *Config) GetLauncherCommand(name string) *LauncherCommand {
	switch name {
	case "dmenu":
		return &c.Launchers.Dmenu
	case "rofi":
		return &c.Launchers.Rofi
	case "fzf":
		return &c.Launchers.Fzf
	case "bemenu":
		return &c.Launchers.Bemenu
	case "fuzzel":
		return &c.Launchers.Fuzzel
	default:
		return nil
	}
}

// mergeLauncherConfigs merge launcher configs
func mergeLauncherConfigs(merged *LauncherConfig, user *LauncherConfig) {
	// Merge dmenu args
	if len(user.Dmenu.Args) > 0 {
		merged.Dmenu.Args = user.Dmenu.Args
	}

	// Merge rofi args
	if len(user.Rofi.Args) > 0 {
		merged.Rofi.Args = user.Rofi.Args
	}

	// Merge fzf args
	if len(user.Fzf.Args) > 0 {
		merged.Fzf.Args = user.Fzf.Args
	}

	// Merge bemenu args
	if len(user.Bemenu.Args) > 0 {
		merged.Bemenu.Args = user.Bemenu.Args
	}

	// Merge fuzzel args
	if len(user.Fuzzel.Args) > 0 {
		merged.Fuzzel.Args = user.Fuzzel.Args
	}
}

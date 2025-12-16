# ql - Quick Launch

A lightweight, modular command launcher for Linux with support for multiple menu systems (dmenu, rofi, fzf, bemenu, fuzzel).

## Features

- Modular Architecture - Each command is a separate module that can be enabled/disabled
- Multiple Launchers - Support for dmenu, rofi, fzf, bemenu, and fuzzel
- Highly Configurable - TOML-based configuration with sensible defaults
- Partial Config Merge - Override only what you need, keep defaults for the rest
- Easy to Extend - Add new command modules easily

## Installation

### From Source

```bash
git clone https://github.com/lvim-tech/ql.git
cd ql
go build -o ql cmd/ql/main.go
sudo mv ql /usr/local/bin/
```

### Core Dependencies

**Required:**

- Go 1.21+ (for building only)

**Runtime (at least one required):**

- dmenu - Simple X11 menu
- rofi - Feature-rich application launcher (X11/Wayland)
- fzf - Terminal-based fuzzy finder
- bemenu - Wayland-native menu
- fuzzel - Wayland-native application launcher

## Quick Start

```bash
ql init-config
nano ~/.config/ql/config.toml
ql
ql power
ql screenshot
ql --rofi
```

## Available Modules

### 1. Hub

Main command menu - shows all enabled modules

**Usage:**

```bash
ql
ql --rofi
ql hub
```

**Dependencies:** None (requires only a launcher)

---

### 2. Power Management

System power operations (logout, suspend, hibernate, reboot, shutdown)

**Usage:**

```bash
ql power
```

**Dependencies:**

- systemctl (systemd)
- loginctl (systemd-logind)

**Optional:**

- swaymsg (Sway)
- i3-msg (i3)
- hyprctl (Hyprland)

**Config:**

```toml
[commands.power]
enabled = true
show_logout = true
show_suspend = true
show_hibernate = false
show_reboot = true
show_shutdown = true
confirm_logout = true
confirm_suspend = false
confirm_hibernate = true
confirm_reboot = true
confirm_shutdown = true
logout_command = "loginctl terminate-user $USER"
suspend_command = "systemctl suspend"
hibernate_command = "systemctl hibernate"
reboot_command = "systemctl reboot"
shutdown_command = "systemctl poweroff"
```

---

### 3. Screenshot

Take screenshots with X11 and Wayland support

**Usage:**

```bash
ql screenshot
```

**Dependencies (Wayland):**

- grim - Screenshot utility
- slurp - Region selector
- wl-copy - Clipboard (wl-clipboard package)
- swaymsg - Window detection (Sway only, optional)
- jq - JSON parsing

**Dependencies (X11):**

- maim - Screenshot utility
- xdotool - Window detection
- xclip - Clipboard

**Screenshot Modes:**

- Fullscreen
- Active Window
- Selected Region
- Current Output (Wayland only)

**Destinations:**

- Save to File
- Copy to Clipboard
- Both

**Config:**

```toml
[commands.screenshot]
enabled = true
save_dir = "~/Pictures/Screenshots"
file_prefix = "screenshot"
```

---

## Configuration

### Config Files Priority

1. ~/.config/ql/config.toml (user)
2. /etc/ql/config.toml (system)
3. Embedded defaults

### Commands

```bash
ql init-config           # Create user config
ql show-config           # Show merged config
ql show-default-config   # Show defaults
```

### Partial Config Example

```toml
default_launcher = "rofi"

[commands.power]
show_hibernate = true

[commands.screenshot]
save_dir = "~/Screenshots"
```

### Full Default Config

```toml
default_launcher = "rofi"

[launchers.dmenu]
command = "dmenu"
args = ["-i", "-l", "20"]

[launchers.rofi]
command = "rofi"
args = ["-dmenu", "-i"]

[launchers.fzf]
command = "fzf"
args = ["--height", "40%", "--reverse", "--border"]

[launchers.bemenu]
command = "bemenu"
args = ["-i", "-l", "20"]

[launchers.fuzzel]
command = "fuzzel"
args = ["--dmenu"]

[commands. power]
enabled = true
show_logout = true
show_suspend = true
show_hibernate = false
show_reboot = true
show_shutdown = true
confirm_logout = true
confirm_suspend = false
confirm_hibernate = true
confirm_reboot = true
confirm_shutdown = true
logout_command = "loginctl terminate-user $USER"
suspend_command = "systemctl suspend"
hibernate_command = "systemctl hibernate"
reboot_command = "systemctl reboot"
shutdown_command = "systemctl poweroff"

[commands.screenshot]
enabled = true
save_dir = "~/Pictures/Screenshots"
file_prefix = "screenshot"
```

---

## Command Line Usage

### Commands

```bash
ql                      # Run hub
ql hub                  # Run hub explicitly
ql power                # Power menu
ql screenshot           # Screenshot menu
ql init-config          # Create user config
ql show-config          # Show merged config
ql show-default-config  # Show defaults
```

### Launcher Flags

```bash
--dmenu
--rofi
--fzf
--bemenu
--fuzzel
```

### Examples

```bash
ql power --rofi
ql screenshot --fzf
ql --dmenu
```

---

## Adding New Modules

### 1. Create Module File

`pkg/commands/yourmodule/yourmodule.go`:

```go
package yourmodule

import (
    "github.com/lvim-tech/ql/pkg/commands"
    "github. com/lvim-tech/ql/pkg/launcher"
)

func init() {
    commands.Register(commands.Command{
        Name:        "yourmodule",
        Description: "Your module description",
        Run:         Run,
    })
}

func Run(ctx *launcher.Context) error {
    options := []string{"Option 1", "Option 2", "Option 3"}
    choice, err := ctx.Show(options, "Select Option")
    if err != nil {
        return err
    }
    // Handle choice
    return nil
}
```

### 2. Import in main.go

```go
import (
    _ "github.com/lvim-tech/ql/pkg/commands/yourmodule"
)
```

### 3. Add Config Support

In `pkg/config/config.go`:

```go
type YourModuleConfig struct {
    Enabled bool   `toml:"enabled"`
    Option1 string `toml:"option1"`
}

type CommandsConfig struct {
    Power      PowerConfig      `toml:"power"`
    Screenshot ScreenshotConfig `toml:"screenshot"`
    YourModule YourModuleConfig `toml:"yourmodule"`
}
```

### 4. Update Hub

In `pkg/commands/hub/hub.go`:

```go
func isCommandEnabled(cfg *config. Config, cmdName string) bool {
    switch cmdName {
    case "power":
        return cfg.Commands.Power.Enabled
    case "screenshot":
        return cfg. Commands.Screenshot.Enabled
    case "yourmodule":
        return cfg.Commands.YourModule.Enabled
    default:
        return true
    }
}
```

---

## Project Structure

```
ql/
├── cmd/
│   └── ql/
│       └── main. go
├── pkg/
│   ├── commands/
│   │   ├── registry.go
│   │   ├── hub/
│   │   │   └── hub.go
│   │   ├── power/
│   │   │   └── power.go
│   │   └── screenshot/
│   │       └── screenshot.go
│   ├── config/
│   │   ├── config.go
│   │   └── default. toml
│   └── launcher/
│       ├── launcher.go
│       ├── dmenu.go
│       ├── rofi.go
│       ├── fzf.go
│       ├── bemenu. go
│       └── fuzzel.go
├── go.mod
├── go.sum
└── README.md
```

---

## License

MIT License

---

## Credits

Created by [lvim-tech](https://github.com/lvim-tech)

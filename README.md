# ql - Quick Launch

A lightweight, modular command launcher for Linux with support for multiple menu systems (dmenu, rofi, fzf, bemenu, fuzzel).

## Features

- **Modular Architecture** - Each command is a separate module that can be enabled/disabled
- **Multiple Launchers** - Support for dmenu, rofi, fzf, bemenu, and fuzzel
- **Highly Configurable** - TOML-based configuration with sensible defaults
- **Grouped & Flat Menus** - Organize commands in groups or use flat list
- **Partial Config Merge** - Override only what you need, keep defaults for the rest
- **Easy to Extend** - Add new command modules easily

## Installation

### From Source

git clone https://github.com/lvim-tech/ql.git
cd ql
go build -o ql cmd/ql/main.go
sudo mv ql /usr/local/bin/

### Core Dependencies

**Required:**

- Go 1.21+ (for building only)

**Runtime (at least one required):**

- **dmenu** - Simple X11 menu
- **rofi** - Feature-rich application launcher (X11/Wayland)
- **fzf** - Terminal-based fuzzy finder
- **bemenu** - Wayland-native menu
- **fuzzel** - Wayland-native application launcher

## Quick Start

ql --init
nano ~/.config/ql/config. toml
ql
ql --flat
ql --group media
ql --launcher rofi

---

## Available Modules

### 🔌 System Group

#### 1. Power Management

System power operations (logout, suspend, hibernate, reboot, shutdown)

**Usage:**

ql power
ql --group system

**Dependencies:**

- **systemctl** (systemd) - Required
- **loginctl** (systemd-logind) - Required

**Optional:**

- **swaymsg** (Sway)
- **i3-msg** (i3)
- **hyprctl** (Hyprland)

**Config:**

[commands.power]
enabled = true
show_logout = true
show_suspend = true
show_hibernate = false
show_reboot = true
show_shutdown = true
confirm_logout = false
confirm_suspend = false
confirm_hibernate = true
confirm_reboot = true
confirm_shutdown = true
logout_command = "loginctl terminate-user $USER"
suspend_command = "systemctl suspend"
hibernate_command = "systemctl hibernate"
reboot_command = "systemctl reboot"
shutdown_command = "systemctl poweroff"

---

#### 2. Screenshot

Take screenshots with X11 and Wayland support

**Usage:**

ql screenshot
ql --group system

**Dependencies (Wayland):**

- **grim** - Screenshot utility
- **slurp** - Region selector
- **swaymsg** (optional) - Window detection for Sway
- **hyprctl** (optional) - Window detection for Hyprland

**Dependencies (X11):**

- **maim** - Screenshot utility (recommended)
- **scrot** - Alternative screenshot tool
- **xdotool** - Window detection

**Screenshot Modes:**

- Fullscreen
- Active Window
- Select Region

**Config:**

[commands.screenshot]
enabled = true
save_dir = "~/Pictures/Screenshots"
file_prefix = "screenshot"

---

### 🌐 Network Group

#### 3. WiFi Manager

Manage WiFi connections with NetworkManager

**Usage:**

ql wifi
ql --group network

**Dependencies:**

- **nmcli** (NetworkManager) - Required
- **ping** (optional) - Connection testing

**Features:**

- Scan and connect to networks
- Disconnect from current network
- Show current connection status
- Toggle WiFi on/off
- Password input support
- Connection testing

**Config:**

[commands. wifi]
enabled = true
show_notify = true
test_host = "1.1.1.1"
test_count = 3
test_wait = 2

---

### 🎵 Media Group

#### 4. Radio Player

Internet radio streaming with mpv

**Usage:**

ql radio
ql --group media

**Dependencies:**

- **mpv** - Media player (required)

**Features:**

- Play internet radio stations
- Stop playback
- 50+ preconfigured stations
- Volume control
- Support for various genres (Chill, Electronic, Rock, Metal, Jazz, etc.)

**Config:**

[commands.radio]
enabled = true
volume = 70

[commands.radio. radio_stations]
"SomaFM Groove Salad" = "https://ice1.somafm.com/groovesalad-128-mp3"
"Radio Paradise Main Mix" = "https://stream.radioparadise.com/mp3-128"
"Jazz24" = "https://live.wostreaming.net/direct/ppm-jazz24mp3-ibc1"

---

#### 5. MPC (MPD Client)

Control Music Player Daemon (MPD)

**Usage:**

ql mpc
ql --group media

**Dependencies:**

- **mpc** - MPD client (required)
- **mpd** - Music Player Daemon (required)

**Features:**

- Play/Pause toggle
- Next/Previous track
- Stop playback
- Select playlist
- Select song from current playlist
- Show current track
- Socket and TCP connection support

**Config:**

[commands.mpc]
enabled = true
connection_type = "socket"
socket = "~/.local/state/mpd/socket"
host = "localhost"
port = "6600"
password = ""
current_playlist_cache = "~/.cache/ql/current_playlist"

---

#### 6. Audio Recording

Record audio from microphone using ffmpeg

**Usage:**

ql audiorecord
ql --group media

**Dependencies:**

- **ffmpeg** - Audio/video processing (required)
- **pulseaudio** or **pipewire** - Audio server (required)

**Features:**

- Start/stop recording
- Configurable format and quality
- Auto-timestamped filenames
- Process management

**Config:**

[commands.audiorecord]
enabled = true
save_dir = "~/Music/Recordings"
file_prefix = "recording"
format = "mp3"
quality = "2"

---

#### 7. Video Recording

Screen recording with X11 and Wayland support

**Usage:**

ql videorecord
ql --group media

**Dependencies (Wayland):**

- **wf-recorder** - Wayland screen recorder (required)
- **slurp** - Region selector (for region recording)

**Dependencies (X11):**

- **ffmpeg** - Video processing (required)
- **slop** - Region selector (for region recording)
- **xdotool** - Window detection (for window recording)

**Recording Modes:**

- Fullscreen
- Active Window
- Select Region

**Features:**

- Audio recording support
- Configurable codecs and quality
- Auto-timestamped filenames
- Separate Wayland/X11 codec settings

**Config:**

[commands. videorecord]
enabled = true
save_dir = "~/Videos/Recordings"
file_prefix = "screencast"
format = "mp4"
quality = "23"
record_audio = true
show_notify = true

[commands.videorecord.wayland]
video_codec = "libx264"
audio_codec = "aac"
preset = "fast"
framerate = 30

[commands.videorecord. x11]
video_codec = "libx264"
audio_codec = "aac"
preset = "fast"
framerate = 30
output_fps = 30

---

### ℹ️ Info Group

#### 8. Weather

Display weather information using wttr.in

**Usage:**

ql weather
ql --group info

**Dependencies:**

- **curl** - HTTP client (required)
- Internet connection

**Features:**

- Current weather conditions
- Multiple locations support
- Configurable display format
- Notification support
- Timeout control

**Config:**

[commands. weather]
enabled = true
locations = ["Sofia", "London", "New York"]
options = ""
timeout = 30

---

## Configuration

### Config Files Priority

1. `~/.config/ql/config. toml` (user)
2. `/etc/ql/config.toml` (system)
3. Embedded defaults

### Commands

ql --init # Create user config
ql --version # Show version
ql --help # Show help

### Menu Styles

**Grouped Menu (default):**

menu_style = "grouped"

[module_groups. system]
name = "System"
enabled = true
modules = ["power", "screenshot"]

[module_groups.network]
name = "Network"
enabled = true
modules = ["wifi"]

[module_groups.media]
name = "Media"
enabled = true
modules = ["radio", "mpc", "audiorecord", "videorecord"]

[module_groups.info]
name = "Info"
enabled = true
modules = ["weather"]

**Flat Menu:**

menu_style = "flat"
module_order = ["power", "screenshot", "wifi", "radio", "mpc", "weather"]

### Launcher Configuration

default_launcher = "auto"

[launchers.rofi]
args = ["-dmenu", "-i", "-p"]

[launchers.dmenu]
args = ["-i", "-p"]

[launchers. fzf]
args = ["--prompt", "--height=40%", "--reverse"]

[launchers.bemenu]
args = ["-i", "-p"]

[launchers.fuzzel]
args = ["--dmenu", "--prompt"]

### Notifications

[notifications]
enabled = true
tool = "auto"
timeout = 5000
urgency = "normal"
show_in_terminal = false

---

## Command Line Usage

### Basic Commands

ql # Run with grouped menu
ql --flat # Run with flat menu
ql --grouped # Force grouped menu
ql --launcher rofi # Use specific launcher
ql --group media # Show only media group
ql power # Run power module directly

### Examples

ql --flat --launcher rofi
ql --grouped
ql --group media
ql --group system --launcher fuzzel

---

## Project Structure

ql/
├── cmd/
│ └── ql/
│ └── main.go
├── pkg/
│ ├── commands/
│ │ ├── commands.go
│ │ ├── audiorecord/
│ │ │ ├── audiorecord.go
│ │ │ └── config.go
│ │ ├── mpc/
│ │ │ ├── mpc.go
│ │ │ └── config.go
│ │ ├── power/
│ │ │ ├── power.go
│ │ │ └── config.go
│ │ ├── radio/
│ │ │ ├── radio.go
│ │ │ └── config.go
│ │ ├── screenshot/
│ │ │ ├── screenshot.go
│ │ │ └── config.go
│ │ ├── videorecord/
│ │ │ ├── videorecord.go
│ │ │ └── config.go
│ │ ├── weather/
│ │ │ ├── weather. go
│ │ │ └── config.go
│ │ └── wifi/
│ │ ├── wifi. go
│ │ └── config.go
│ ├── config/
│ │ ├── config.go
│ │ ├── default.toml
│ │ ├── merge.go
│ │ └── provider.go
│ ├── launcher/
│ │ ├── launcher.go
│ │ ├── bemenu.go
│ │ ├── dmenu.go
│ │ ├── fzf.go
│ │ ├── fuzzel.go
│ │ └── rofi.go
│ └── utils/
│ ├── notifications.go
│ └── utils.go
├── go.mod
├── go.sum
├── LICENSE
└── README.md

---

## Adding New Modules

### 1. Create Module Files

pkg/commands/yourmodule/yourmodule.go

package yourmodule

import (
"fmt"
"github.com/lvim-tech/ql/pkg/commands"
"github.com/lvim-tech/ql/pkg/config"
)

func init() {
commands.Register(commands.Command{
Name: "yourmodule",
Description: "Your module description",
Run: Run,
})
}

func Run(ctx commands.LauncherContext) commands.CommandResult {
// Get config
cfg := getConfig(ctx. Config())

    for {
        options := []string{
            "← Back",
            "Option 1",
            "Option 2",
        }

        choice, err := ctx.Show(options, "Your Module")
        if err != nil {
            return commands.CommandResult{Success: false}
        }

        if choice == "← Back" {
            return commands.CommandResult{
                Success: false,
                Error:   commands.ErrBack,
            }
        }

        // Handle action
        var actionErr error
        switch choice {
        case "Option 1":
            actionErr = doSomething()
        case "Option 2":
            actionErr = doSomethingElse()
        }

        if actionErr != nil {
            // Show error and loop back
            continue
        }

        // Success - exit
        return commands.CommandResult{Success: true}
    }

}

pkg/commands/yourmodule/config.go

package yourmodule

import (
"github.com/lvim-tech/ql/pkg/config"
"github.com/mitchellh/mapstructure"
)

type Config struct {
Enabled bool `toml:"enabled"`
Option1 string `toml:"option1"`
Option2 int `toml:"option2"`
}

func DefaultConfig() Config {
return Config{
Enabled: true,
Option1: "default",
Option2: 42,
}
}

func getConfig(cfg \*config.Config) Config {
cfgInterface := cfg.GetYourModuleConfig()

    var result Config
    decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
        WeaklyTypedInput: true,
        Result:           &result,
    })

    if err != nil {
        return DefaultConfig()
    }

    if decodeErr := decoder.Decode(cfgInterface); decodeErr != nil {
        return DefaultConfig()
    }

    return result

}

### 2. Import in main.go

import (
\_ "github.com/lvim-tech/ql/pkg/commands/yourmodule"
)

### 3. Add Config Support in pkg/config/config.go

Add getter method:

func (c \*Config) GetYourModuleConfig() interface{} {
if cfg, exists := c.Commands["yourmodule"]; exists {
return cfg
}
return map[string]interface{}{
"enabled": true,
"option1": "default",
"option2": 42,
}
}

### 4. Add to Module Group in config.toml

[module_groups.custom]
name = "Custom"
enabled = true
modules = ["yourmodule"]

[commands.yourmodule]
enabled = true
option1 = "value"
option2 = 100

---

## Troubleshooting

### Common Issues

**Module not showing:**

- Check if enabled in config: `enabled = true`
- Verify dependencies are installed
- Check module group configuration
- Rebuild after adding new modules: `go build`

**Launcher not working:**

- Verify launcher is installed: `which rofi`
- Check launcher args in config
- Try different launcher: `ql --launcher fzf`
- Use auto-detection: `default_launcher = "auto"`

**WiFi connection fails:**

- Ensure NetworkManager is running: `systemctl status NetworkManager`
- Check nmcli works: `nmcli device wifi list`
- Verify WiFi is enabled: `nmcli radio wifi on`

**MPD connection fails:**

- Check MPD is running: `systemctl --user status mpd`
- Verify socket path: `ls ~/.local/state/mpd/socket`
- Test with mpc: `export MPD_HOST=~/.local/state/mpd/socket && mpc status`
- Check connection_type in config (socket vs tcp)

**Recording issues:**

- Check ffmpeg/wf-recorder installed
- Verify audio server running: `systemctl --user status pipewire` or `pulseaudio --check`
- Check file permissions in save directory
- Ensure save directory exists

**Radio not playing:**

- Verify mpv is installed: `which mpv`
- Test URL manually: `mpv --no-video "URL"`
- Check internet connection
- Verify station URL is valid

**Weather not showing:**

- Check curl is installed: `which curl`
- Test manually: `curl wttr.in/Sofia`
- Verify internet connection
- Check location spelling

---

## Development

### Building

go build -o ql cmd/ql/main. go

### Running Tests

go test ./...

### Installing Locally

go install cmd/ql/main.go

---

## License

MIT License

---

## Credits

Created by [lvim-tech](https://github.com/lvim-tech)

---

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch: `git checkout -b feature/amazing-feature`
3. Commit your changes: `git commit -am 'Add amazing feature'`
4. Push to the branch: `git push origin feature/amazing-feature`
5. Open a Pull Request

### Contribution Guidelines

- Follow existing code structure
- Add tests for new features
- Update documentation
- Use meaningful commit messages
- Keep modules independent and self-contained

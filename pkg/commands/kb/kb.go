// Package kb switches the keyboard layout — the lvim-linguistics way: a
// targeted SET of a named layout, never a blind toggle. `ql kb bg` on a
// keybind is idempotent and instant; the state lives in the compositor, not
// in the user's head.
package kb

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/utils"
)

func init() {
	commands.Register(commands.Command{
		Name:        "kb",
		Description: "Keyboard layout",
		Run:         Run,
	})
}

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfg := commands.DecodeConfig(ctx, "kb", DefaultConfig())

	if !cfg.Enabled {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("kb module is disabled in config"),
		}
	}

	notifCfg := ctx.Config().GetNotificationConfig()

	layouts := cfg.Layouts
	if len(layouts) == 0 {
		layouts = detectLayouts()
	}
	if len(layouts) == 0 {
		err := fmt.Errorf("could not detect the configured layouts — set them in [commands.kb] layouts")
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Keyboard Error", err.Error())
		return commands.CommandResult{Success: false, Error: err}
	}

	if args := ctx.Args(); len(args) > 0 {
		// "current" is the one-shot indicator; "watch" is the same thing
		// as a stream, for a status bar that would otherwise poll.
		switch strings.ToLower(args[0]) {
		case "current":
			fmt.Println(displayName(currentLayout(), layouts))
			return commands.CommandResult{Success: true}
		case "watch":
			watch(layouts)
			return commands.CommandResult{Success: true}
		}
		if err := switchTo(args[0], layouts); err != nil {
			utils.ShowErrorNotificationWithConfig(&notifCfg, "Keyboard Error", err.Error())
			return commands.CommandResult{Success: false, Error: err}
		}
		return commands.CommandResult{Success: true}
	}

	options := []string{}
	if !ctx.IsDirectLaunch() {
		options = append(options, "← Back")
	}
	options = append(options, layouts...)

	choice, err := ctx.Show(options, "Keyboard Layout")
	if err != nil {
		return commands.CommandResult{Success: false}
	}
	if choice == "← Back" {
		return commands.CommandResult{Success: false, Error: commands.ErrBack}
	}

	if err := switchTo(choice, layouts); err != nil {
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Keyboard Error", err.Error())
		return commands.CommandResult{Success: false, Error: err}
	}
	return commands.CommandResult{Success: true}
}

// detectLayouts asks the SESSION what is configured — nothing hardcoded.
// mango writes the list in its own config file, Hyprland and sway answer
// over IPC, X11 through setxkbmap; localectl is the last resort.
func detectLayouts() []string {
	// mango: xkb_rules_layout=us,bg in ~/.config/mango/*.conf
	if utils.CommandExists("mmsg") {
		home, _ := os.UserHomeDir()
		matches, _ := filepath.Glob(filepath.Join(home, ".config", "mango", "*.conf"))
		for _, path := range matches {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			for _, line := range strings.Split(string(data), "\n") {
				if value, ok := strings.CutPrefix(strings.TrimSpace(line), "xkb_rules_layout="); ok {
					if layouts := splitLayouts(value); len(layouts) > 0 {
						return layouts
					}
				}
			}
		}
	}

	if utils.CommandExists("hyprctl") {
		if out, err := exec.Command("hyprctl", "getoption", "input:kb_layout", "-j").Output(); err == nil {
			var opt struct {
				Str string `json:"str"`
			}
			if json.Unmarshal(out, &opt) == nil {
				if layouts := splitLayouts(opt.Str); len(layouts) > 0 {
					return layouts
				}
			}
		}
	}

	if utils.CommandExists("swaymsg") {
		if out, err := exec.Command("swaymsg", "-t", "get_inputs", "-r").Output(); err == nil {
			var inputs []struct {
				Type        string   `json:"type"`
				LayoutNames []string `json:"xkb_layout_names"`
			}
			if json.Unmarshal(out, &inputs) == nil {
				for _, in := range inputs {
					if in.Type == "keyboard" && len(in.LayoutNames) > 0 {
						return in.LayoutNames
					}
				}
			}
		}
	}

	if utils.CommandExists("setxkbmap") {
		if out, err := exec.Command("setxkbmap", "-query").Output(); err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if value, ok := strings.CutPrefix(strings.TrimSpace(line), "layout:"); ok {
					if layouts := splitLayouts(value); len(layouts) > 0 {
						return layouts
					}
				}
			}
		}
	}

	if out, err := exec.Command("localectl", "status").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if value, ok := strings.CutPrefix(strings.TrimSpace(line), "X11 Layout:"); ok {
				return splitLayouts(value)
			}
		}
	}

	return nil
}

func splitLayouts(value string) []string {
	var layouts []string
	for _, l := range strings.Split(value, ",") {
		if l = strings.TrimSpace(l); l != "" {
			layouts = append(layouts, l)
		}
	}
	return layouts
}

// switchTo sets a layout by name, or advances with "next"/"prev".
func switchTo(name string, layouts []string) error {
	if name == "next" || name == "prev" {
		target, err := neighbourLayout(name == "prev", layouts)
		if err != nil {
			return err
		}
		name = target
	}
	return setLayout(name, layouts)
}

// neighbourLayout reads the current layout and picks the adjacent one from
// the detected list — the portable way to spell "next", since not every
// compositor has a native token for it.
func neighbourLayout(backwards bool, layouts []string) (string, error) {
	if len(layouts) == 0 {
		return "", fmt.Errorf("no layouts detected")
	}
	current := currentLayout()
	index := 0
	for i, l := range layouts {
		if sameLayout(l, current) {
			index = i
			break
		}
	}
	step := 1
	if backwards {
		step = len(layouts) - 1
	}
	return layouts[(index+step)%len(layouts)], nil
}

// layoutAliases maps xkb spellings of the same layout onto each other:
// asking for "en" sets what the compositor then reports as "us".
var layoutAliases = map[string]string{"us": "en"}

func sameLayout(a, b string) bool {
	na, nb := strings.ToLower(a), strings.ToLower(b)
	if v, ok := layoutAliases[na]; ok {
		na = v
	}
	if v, ok := layoutAliases[nb]; ok {
		nb = v
	}
	return na == nb
}

// currentLayout asks the compositor which layout is active; empty when it
// cannot say (the neighbour computation then starts from the first entry).
func currentLayout() string {
	if utils.CommandExists("mmsg") {
		if out, err := exec.Command("mmsg", "-k").Output(); err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				fields := strings.Fields(line)
				for i, f := range fields {
					if f == "kb_layout" && i+1 < len(fields) {
						return fields[i+1]
					}
				}
			}
		}
	}
	if utils.CommandExists("xkb-switch") {
		if out, err := exec.Command("xkb-switch", "-p").Output(); err == nil {
			return strings.TrimSpace(string(out))
		}
	}
	return ""
}

// setLayout dispatches to the compositor — the same matrix lvim-linguistics
// uses, in Go: mango and X11 take the layout NAME, the others take its
// index in the detected list.
func setLayout(name string, layouts []string) error {
	desktop := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP") + " " + os.Getenv("DESKTOP_SESSION"))
	wayland := os.Getenv("WAYLAND_DISPLAY") != ""

	switch {
	case strings.Contains(desktop, "mango"), wayland && utils.CommandExists("mmsg"):
		return setLayoutByCycling(name, layouts)
	case strings.Contains(desktop, "hyprland"):
		return runQuiet("hyprctl", "switchxkblayout", "all", strconv.Itoa(layoutIndex(name, layouts)))
	case strings.Contains(desktop, "niri"), os.Getenv("NIRI_SOCKET") != "":
		return runQuiet("niri", "msg", "action", "switch-layout", strconv.Itoa(layoutIndex(name, layouts)))
	case strings.Contains(desktop, "sway"):
		return runQuiet("swaymsg", "input", "type:keyboard", "xkb_switch_layout", strconv.Itoa(layoutIndex(name, layouts)))
	case utils.CommandExists("xkb-switch"):
		return runQuiet("xkb-switch", "-s", name)
	}
	return fmt.Errorf("no known layout switcher for this session")
}

func layoutIndex(name string, layouts []string) int {
	for i, l := range layouts {
		if sameLayout(l, name) {
			return i
		}
	}
	return 0
}

func runQuiet(bin string, args ...string) error {
	out, err := exec.Command(bin, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", bin, strings.TrimSpace(string(out)))
	}
	return nil
}

// displayName is the short label a status bar shows: the xkb id uppercased,
// with the en/us pair spelled the way people read it. An empty reading
// falls back to the first configured layout rather than to nothing at all.
func displayName(layout string, layouts []string) string {
	if layout == "" {
		if len(layouts) == 0 {
			return "??"
		}
		layout = layouts[0]
	}
	if sameLayout(layout, "us") {
		return "EN"
	}
	name := strings.ToUpper(layout)
	if len(name) > 2 {
		name = name[:2]
	}
	return name
}

// watch prints the layout on every change, forever — a status bar holds
// this open instead of polling, which is where the visible lag came from: a
// 1-second interval meant the indicator trailed the switch by up to a
// second, and every tick spawned a process.
//
// Two sources feed it. mango's `mmsg -w` event stream makes a switch show
// up instantly, from ANY origin (this module, a compositor keybind,
// lvim-linguistics). A slow reconciliation poll runs alongside it, because
// a stream that misses an event — or a compositor that has none at all —
// would otherwise leave the bar showing the wrong layout indefinitely: it
// costs one tiny query a second and prints nothing while the two agree.
func watch(layouts []string) {
	lines := make(chan string, 8)

	if utils.CommandExists("mmsg") {
		go func() {
			cmd := exec.Command("mmsg", "-w")
			stdout, err := cmd.StdoutPipe()
			if err != nil || cmd.Start() != nil {
				return
			}
			defer cmd.Wait()
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				fields := strings.Fields(scanner.Text())
				for i, f := range fields {
					if f == "kb_layout" && i+1 < len(fields) {
						lines <- fields[i+1]
					}
				}
			}
		}()
	}

	// 200ms, not a second: the event stream only fires for switches made
	// through the compositor's DISPATCHER (ql, lvim-linguistics). A layout
	// bound inside xkb — mango's grp:alt_shift_toggle, and every desktop
	// with the same option — changes the layout with no event at all, so
	// this poll is the only thing that ever sees it. One tiny query five
	// times a second buys an indicator that looks instant either way; it
	// prints nothing while the reading is unchanged.
	go func() {
		for {
			time.Sleep(200 * time.Millisecond)
			lines <- currentLayout()
		}
	}()

	last := ""
	print := func(layout string) {
		name := displayName(layout, layouts)
		if name == last {
			return
		}
		last = name
		fmt.Println(name)
	}

	print(currentLayout())
	for layout := range lines {
		print(layout)
	}
}

// setLayoutByCycling implements a targeted SET on a compositor that only
// knows how to cycle. mango's switch_keyboard_layout IGNORES its argument
// entirely — measured: three consecutive requests for "bg" produced bg, us,
// bg — so asking for a layout by name toggled instead, and any key bound to
// a specific layout did the wrong thing half the time. Cycle until the
// compositor reports the layout that was asked for, at most one full lap so
// an unknown name cannot spin forever.
func setLayoutByCycling(name string, layouts []string) error {
	laps := len(layouts)
	if laps == 0 {
		laps = 4
	}
	for i := 0; i < laps; i++ {
		if sameLayout(currentLayout(), name) {
			return nil
		}
		if err := runQuiet("mmsg", "-d", "switch_keyboard_layout"); err != nil {
			return err
		}
		// The compositor applies the switch asynchronously; without this
		// the next read still returns the previous layout and the loop
		// overshoots.
		time.Sleep(60 * time.Millisecond)
	}
	if sameLayout(currentLayout(), name) {
		return nil
	}
	return fmt.Errorf("could not reach layout %q by cycling", name)
}

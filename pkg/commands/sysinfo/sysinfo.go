// Package sysinfo shows the detail BEHIND the waybar numbers: the bar's
// built-in cpu/memory/disk modules display one figure each, and their
// on-click points here for the processes and filesystems producing it.
package sysinfo

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/utils"
)

func init() {
	commands.Register(commands.Command{
		Name:        "sysinfo",
		Description: "System info (CPU, memory, disk)",
		Run:         Run,
	})
}

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfg := commands.DecodeConfig(ctx, "sysinfo", DefaultConfig())

	if !cfg.Enabled {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("sysinfo module is disabled in config"),
		}
	}

	notifCfg := ctx.Config().GetNotificationConfig()

	if args := ctx.Args(); len(args) > 0 {
		if err := show(args[0], &cfg); err != nil {
			utils.ShowErrorNotificationWithConfig(&notifCfg, "Sysinfo Error", err.Error())
			return commands.CommandResult{Success: false, Error: err}
		}
		return commands.CommandResult{Success: true}
	}

	options := []string{}
	if !ctx.IsDirectLaunch() {
		options = append(options, "← Back")
	}
	options = append(options, "CPU", "Memory", "Disk")

	choice, err := ctx.Show(options, "System Info")
	if err != nil {
		return commands.CommandResult{Success: false}
	}
	if choice == "← Back" {
		return commands.CommandResult{Success: false, Error: commands.ErrBack}
	}

	if err := show(strings.ToLower(choice), &cfg); err != nil {
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Sysinfo Error", err.Error())
		return commands.CommandResult{Success: false, Error: err}
	}
	return commands.CommandResult{Success: true}
}

func show(what string, cfg *Config) error {
	lines := strconv.Itoa(cfg.Lines)
	switch strings.ToLower(what) {
	case "cpu":
		return display("CPU", [][]string{
			{"uptime"},
			{"sh", "-c", "ps -eo pid,pcpu,pmem,comm --sort=-pcpu | head -" + lines},
		})
	case "mem", "memory":
		return display("Memory", [][]string{
			{"free", "-h"},
			{"sh", "-c", "ps -eo pid,pmem,pcpu,comm --sort=-pmem | head -" + lines},
		})
	case "disk":
		return display("Disk", [][]string{
			{"df", "-h", "-x", "tmpfs", "-x", "devtmpfs", "-x", "efivarfs"},
			{"lsblk", "-o", "NAME,SIZE,TYPE,FSTYPE,MOUNTPOINT"},
		})
	default:
		return fmt.Errorf("unknown sysinfo view: %s (use: cpu, mem, disk)", what)
	}
}

// display runs each command and shows their outputs as one page. A command
// that fails contributes its error text instead of sinking the whole view.
func display(title string, cmds [][]string) error {
	var b strings.Builder
	for i, argv := range cmds {
		if i > 0 {
			b.WriteString("\n" + strings.Repeat("─", 60) + "\n\n")
		}
		out, err := exec.Command(argv[0], argv[1:]...).CombinedOutput()
		if err != nil {
			fmt.Fprintf(&b, "%s: %v\n", argv[0], err)
			continue
		}
		b.Write(out)
	}
	return utils.DisplayTextGUI(title, b.String())
}

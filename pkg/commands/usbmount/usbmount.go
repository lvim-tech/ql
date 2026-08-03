// Package usbmount mounts and unmounts removable drives through udisksctl —
// the polkit-aware way that needs no sudo and no fstab entry.
package usbmount

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/utils"
)

func init() {
	commands.Register(commands.Command{
		Name:        "usbmount",
		Description: "Mount/unmount USB drives",
		Run:         Run,
	})
}

type device struct {
	Path       string
	Label      string
	Size       string
	FSType     string
	MountPoint string
}

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfg := commands.DecodeConfig(ctx, "usbmount", DefaultConfig())

	if !cfg.Enabled {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("usbmount module is disabled in config"),
		}
	}

	notifCfg := ctx.Config().GetNotificationConfig()

	if !utils.CommandExists("udisksctl") {
		err := fmt.Errorf("udisksctl is not installed (udisks2)")
		utils.ShowErrorNotificationWithConfig(&notifCfg, "USB Error", err.Error())
		return commands.CommandResult{Success: false, Error: err}
	}

	for {
		devices, err := removableDevices()
		if err != nil {
			utils.ShowErrorNotificationWithConfig(&notifCfg, "USB Error", err.Error())
			return commands.CommandResult{Success: false, Error: err}
		}

		var options []string
		actions := map[string]device{}

		if !ctx.IsDirectLaunch() {
			options = append(options, "← Back")
		}
		if len(devices) == 0 {
			options = append(options, "No removable drives found")
		}
		for _, d := range devices {
			label := d.Label
			if label == "" {
				label = d.FSType
			}
			var entry string
			if d.MountPoint == "" {
				entry = fmt.Sprintf("Mount   %s  (%s, %s)", d.Path, label, d.Size)
			} else {
				entry = fmt.Sprintf("Unmount %s  (%s, on %s)", d.Path, label, d.MountPoint)
			}
			options = append(options, entry)
			actions[entry] = d
		}

		choice, err := ctx.Show(options, "USB Drives")
		if err != nil {
			return commands.CommandResult{Success: false}
		}
		if choice == "← Back" {
			return commands.CommandResult{Success: false, Error: commands.ErrBack}
		}

		d, ok := actions[choice]
		if !ok {
			return commands.CommandResult{Success: true}
		}

		verb := "mount"
		if d.MountPoint != "" {
			verb = "unmount"
		}
		out, err := exec.Command("udisksctl", verb, "-b", d.Path).CombinedOutput()
		message := strings.TrimSpace(string(out))
		if err != nil {
			if message == "" {
				message = err.Error()
			}
			utils.ShowErrorNotificationWithConfig(&notifCfg, "USB Error", message)
			continue // the device list may have changed; re-render
		}
		utils.NotifyWithConfig(&notifCfg, "USB", message)
		return commands.CommandResult{Success: true}
	}
}

// removableDevices lists partitions on removable/hotplug media that carry a
// filesystem. lsblk's JSON is the one stable interface for this.
func removableDevices() ([]device, error) {
	out, err := exec.Command("lsblk", "--json", "-o", "PATH,SIZE,TYPE,MOUNTPOINT,RM,HOTPLUG,LABEL,FSTYPE").Output()
	if err != nil {
		return nil, fmt.Errorf("lsblk: %w", err)
	}

	var tree struct {
		Blockdevices []lsblkNode `json:"blockdevices"`
	}
	if err := json.Unmarshal(out, &tree); err != nil {
		return nil, fmt.Errorf("lsblk json: %w", err)
	}

	var devices []device
	var walk func(nodes []lsblkNode, parentRemovable bool)
	walk = func(nodes []lsblkNode, parentRemovable bool) {
		for _, n := range nodes {
			removable := parentRemovable || n.RM || n.Hotplug
			if n.Type == "part" && removable && n.FSType != "" {
				devices = append(devices, device{
					Path:       n.Path,
					Label:      n.Label,
					Size:       n.Size,
					FSType:     n.FSType,
					MountPoint: n.Mountpoint,
				})
			}
			walk(n.Children, removable)
		}
	}
	walk(tree.Blockdevices, false)
	return devices, nil
}

type lsblkNode struct {
	Path       string      `json:"path"`
	Size       string      `json:"size"`
	Type       string      `json:"type"`
	Mountpoint string      `json:"mountpoint"`
	RM         bool        `json:"rm"`
	Hotplug    bool        `json:"hotplug"`
	Label      string      `json:"label"`
	FSType     string      `json:"fstype"`
	Children   []lsblkNode `json:"children"`
}

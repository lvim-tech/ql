// Package kill provides process management and killing functionality for ql.
// It displays running processes and allows killing them with confirmation.
package kill

import (
	"fmt"
	"os/exec"
	"os/user"
	"strings"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/utils"
	"github.com/mitchellh/mapstructure"
)

func init() {
	commands.Register(commands.Command{
		Name:        "kill",
		Description: "Kill processes",
		Run:         Run,
	})
}

type Process struct {
	PID     string
	User    string
	CPU     string
	MEM     string
	Command string
	Display string
}

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfgInterface := ctx.Config().GetKillConfig()

	var cfg Config
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           &cfg,
	})
	if err != nil {
		cfg = DefaultConfig()
	} else {
		if decodeErr := decoder.Decode(cfgInterface); decodeErr != nil {
			cfg = DefaultConfig()
		}
	}

	if !cfg.Enabled {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("kill module is disabled in config"),
		}
	}

	if !utils.CommandExists("ps") {
		notifCfg := ctx.Config().GetNotificationConfig()
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Kill Error",
			"ps command not found")
		return commands.CommandResult{Success: false}
	}

	notifCfg := ctx.Config().GetNotificationConfig()

	// Check for direct command (kill by PID or process name)
	args := ctx.Args()
	if len(args) > 0 {
		return executeDirectKill(args[0], &cfg, &notifCfg)
	}

	processes, err := getProcesses(&cfg)
	if err != nil {
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Kill Error", err.Error())
		return commands.CommandResult{Success: false}
	}

	if len(processes) == 0 {
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Kill Error", "No processes found")
		return commands.CommandResult{Success: false}
	}

	var options []string

	if !ctx.IsDirectLaunch() {
		options = append(options, "← Back")
	}

	for _, proc := range processes {
		options = append(options, proc.Display)
	}

	selected, err := ctx.Show(options, "Kill Process")
	if err != nil {
		// ESC pressed - exit completely
		return commands.CommandResult{Success: false}
	}

	if selected == "← Back" || selected == "" {
		return commands.CommandResult{
			Success: false,
			Error:   commands.ErrBack,
		}
	}

	var selectedProc *Process
	for _, proc := range processes {
		if proc.Display == selected {
			selectedProc = &proc
			break
		}
	}

	if selectedProc == nil {
		return commands.CommandResult{Success: false, Error: commands.ErrBack}
	}

	if cfg.ConfirmKill {
		confirmOpts := []string{"← Back", "Yes", "No"}
		confirm, err := ctx.Show(confirmOpts, fmt.Sprintf("Kill process %s (PID:       %s)?    ", selectedProc.Command, selectedProc.PID))
		if err != nil {
			// ESC pressed - exit completely
			return commands.CommandResult{Success: false}
		}

		if confirm == "← Back" || confirm == "No" {
			return commands.CommandResult{Success: false, Error: commands.ErrBack}
		}

		if confirm != "Yes" {
			return commands.CommandResult{Success: false, Error: commands.ErrBack}
		}
	}

	if err := killProcess(selectedProc.PID); err != nil {
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Kill Error",
			fmt.Sprintf("Failed to kill process:  %v", err))
		return commands.CommandResult{Success: false}
	}

	utils.NotifyWithConfig(&notifCfg, "Process Killed",
		fmt.Sprintf("Killed %s (PID:    %s)", selectedProc.Command, selectedProc.PID))

	return commands.CommandResult{Success: true}
}

func executeDirectKill(target string, cfg *Config, notifCfg *config.NotificationConfig) commands.CommandResult {
	// Try to parse as PID (numeric)
	if isPID(target) {
		if err := killProcess(target); err != nil {
			return commands.CommandResult{
				Success: false,
				Error:   fmt.Errorf("failed to kill PID %s: %w", target, err),
			}
		}
		utils.NotifyWithConfig(notifCfg, "Process Killed", fmt.Sprintf("Killed PID:  %s", target))
		return commands.CommandResult{Success: true}
	}

	// Otherwise treat as process name
	processes, err := getProcesses(cfg)
	if err != nil {
		return commands.CommandResult{Success: false, Error: err}
	}

	// Find matching processes by name
	var matches []Process
	targetLower := strings.ToLower(target)
	for _, proc := range processes {
		if strings.Contains(strings.ToLower(proc.Command), targetLower) {
			matches = append(matches, proc)
		}
	}

	if len(matches) == 0 {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("no process found matching '%s'", target),
		}
	}

	// Kill all matching processes
	var killed []string
	for _, proc := range matches {
		if err := killProcess(proc.PID); err != nil {
			utils.ShowErrorNotificationWithConfig(notifCfg, "Kill Error",
				fmt.Sprintf("Failed to kill %s (PID:  %s): %v", proc.Command, proc.PID, err))
		} else {
			killed = append(killed, fmt.Sprintf("%s (PID: %s)", proc.Command, proc.PID))
		}
	}

	if len(killed) > 0 {
		utils.NotifyWithConfig(notifCfg, "Processes Killed", strings.Join(killed, "\n"))
		return commands.CommandResult{Success: true}
	}

	return commands.CommandResult{
		Success: false,
		Error:   fmt.Errorf("failed to kill any processes"),
	}
}

func isPID(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

func getProcesses(cfg *Config) ([]Process, error) {
	var cmd *exec.Cmd

	if cfg.ShowAllProcesses {
		cmd = exec.Command("ps", "aux", "--sort=-%cpu")
	} else {
		currentUser, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("failed to get current user:    %w", err)
		}
		cmd = exec.Command("ps", "-u", currentUser.Username, "-o", "pid,user,%cpu,%mem,comm", "--sort=-%cpu")
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get processes:    %w", err)
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("no processes found")
	}

	var processes []Process

	for i, line := range lines {
		if i == 0 || line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		pid := fields[0]
		userName := fields[1]
		cpu := fields[2]
		mem := fields[3]
		command := strings.Join(fields[4:], " ")

		if shouldExclude(command, cfg.ExcludeProcesses) {
			continue
		}

		proc := Process{
			PID:     pid,
			User:    userName,
			CPU:     cpu,
			MEM:     mem,
			Command: command,
			Display: fmt.Sprintf("PID:    %-7s | CPU: %-5s%% | MEM: %-5s%% | %s", pid, cpu, mem, command),
		}

		processes = append(processes, proc)
	}

	return processes, nil
}

func shouldExclude(command string, excludeList []string) bool {
	commandLower := strings.ToLower(command)
	for _, exclude := range excludeList {
		if strings.Contains(commandLower, strings.ToLower(exclude)) {
			return true
		}
	}
	return false
}

func killProcess(pid string) error {
	cmd := exec.Command("kill", "-9", pid)
	return cmd.Run()
}

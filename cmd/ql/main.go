package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/lvim-tech/ql/pkg/commands"
	_ "github.com/lvim-tech/ql/pkg/commands/audiorecord"
	_ "github.com/lvim-tech/ql/pkg/commands/mpc"
	_ "github.com/lvim-tech/ql/pkg/commands/power"
	_ "github.com/lvim-tech/ql/pkg/commands/radio"
	_ "github.com/lvim-tech/ql/pkg/commands/screenshot"
	_ "github.com/lvim-tech/ql/pkg/commands/videorecord"
	_ "github.com/lvim-tech/ql/pkg/commands/weather"
	_ "github.com/lvim-tech/ql/pkg/commands/wifi"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/launcher"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	initFlag := flag.Bool("init", false, "Initialize user config")
	versionFlag := flag.Bool("version", false, "Show version")
	helpFlag := flag.Bool("help", false, "Show help")
	flatFlag := flag.Bool("flat", false, "Use flat menu style")
	groupedFlag := flag.Bool("grouped", false, "Use grouped menu style")
	launcherFlag := flag.String("launcher", "", "Override launcher (rofi, dmenu, fzf, bemenu, fuzzel)")
	groupFlag := flag.String("group", "", "Show only commands from specific group")

	flag.Parse()

	if *initFlag {
		return handleInit()
	}

	if *versionFlag {
		fmt.Println("ql version 0.1.0")
		return nil
	}

	if *helpFlag {
		printHelp()
		return nil
	}

	if len(os.Args) > 1 && !flag.Parsed() {
		switch os.Args[1] {
		case "init":
			return handleInit()
		case "version":
			fmt.Println("ql version 0.1.0")
			return nil
		case "help":
			printHelp()
			return nil
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	launcherName := cfg.GetDefaultLauncher()

	if *launcherFlag != "" {
		launcherName = *launcherFlag
	} else if len(flag.Args()) > 0 {
		arg := flag.Args()[0]
		if arg != "init" && arg != "version" && arg != "help" {
			launcherName = arg
		}
	}

	ctx, err := launcher.New(launcherName, cfg)
	if err != nil {
		return fmt.Errorf("failed to create launcher: %w", err)
	}

	if *groupFlag != "" {
		return runSpecificGroup(ctx, cfg, *groupFlag)
	}

	menuStyle := cfg.GetMenuStyle()

	if *flatFlag {
		menuStyle = "flat"
	} else if *groupedFlag {
		menuStyle = "grouped"
	}

	if menuStyle == "grouped" {
		return runGroupedMenu(ctx, cfg)
	}

	return runFlatMenu(ctx, cfg)
}

func runSpecificGroup(ctx launcher.Launcher, cfg *config.Config, groupName string) error {
	groups := cfg.GetModuleGroups()

	var selectedGroup *config.ModuleGroup

	for key, group := range groups {
		if key == groupName || group.Name == groupName {
			selectedGroup = &group
			break
		}
	}

	if selectedGroup == nil {
		fmt.Fprintf(os.Stderr, "Error: Group '%s' not found\n\n", groupName)
		fmt.Fprintf(os.Stderr, "Available groups:\n")

		groupOrder := cfg.GetModuleGroupsOrder()
		for _, key := range groupOrder {
			if group, exists := groups[key]; exists {
				fmt.Fprintf(os.Stderr, "  %s (%s)\n", key, group.Name)
			}
		}

		return fmt.Errorf("group not found")
	}

	registeredCommands := commands.GetAll()
	commandMap := make(map[string]commands.Command)
	for _, cmd := range registeredCommands {
		commandMap[cmd.Name] = cmd
	}

	result := runModuleMenuDirect(ctx, cfg, *selectedGroup, commandMap)

	if !result.Success && result.Error != nil {
		return result.Error
	}

	return nil
}

func runFlatMenu(ctx launcher.Launcher, cfg *config.Config) error {
	registeredCommands := commands.GetAll()
	if len(registeredCommands) == 0 {
		return fmt.Errorf("no commands registered")
	}

	commandMap := make(map[string]commands.Command)
	for _, cmd := range registeredCommands {
		commandMap[cmd.Name] = cmd
	}

	moduleOrder := cfg.GetModuleOrder()
	if len(moduleOrder) == 0 {
		for _, cmd := range registeredCommands {
			moduleOrder = append(moduleOrder, cmd.Name)
		}
	}

	for {
		var options []string
		optionToCommand := make(map[string]commands.Command)

		for _, moduleName := range moduleOrder {
			cmd, exists := commandMap[moduleName]
			if !exists {
				continue
			}

			if !isCommandEnabled(cfg, cmd.Name) {
				continue
			}

			options = append(options, cmd.Description)
			optionToCommand[cmd.Description] = cmd
		}

		if len(options) == 0 {
			return fmt.Errorf("no enabled commands")
		}

		choice, err := ctx.Show(options, "ql")
		if err != nil {
			// ESC pressed - exit
			return nil
		}

		cmd, ok := optionToCommand[choice]
		if !ok {
			showErrorNotification("Error", fmt.Sprintf("Unknown command: %s", choice))
			continue
		}

		_ = cmd.Run(ctx)

		// Command finished - exit
		return nil
	}
}

func runGroupedMenu(ctx launcher.Launcher, cfg *config.Config) error {
	registeredCommands := commands.GetAll()
	if len(registeredCommands) == 0 {
		return fmt.Errorf("no commands registered")
	}

	commandMap := make(map[string]commands.Command)
	for _, cmd := range registeredCommands {
		commandMap[cmd.Name] = cmd
	}

	groups := cfg.GetModuleGroups()
	if len(groups) == 0 {
		return runFlatMenu(ctx, cfg)
	}

	groupOrder := cfg.GetModuleGroupsOrder()

	for {
		var groupOptions []string
		groupMap := make(map[string]config.ModuleGroup)

		for _, groupKey := range groupOrder {
			group, exists := groups[groupKey]
			if !exists {
				continue
			}

			hasEnabled := false
			for _, moduleName := range group.Modules {
				if isCommandEnabled(cfg, moduleName) {
					hasEnabled = true
					break
				}
			}

			if hasEnabled {
				groupOptions = append(groupOptions, group.Name)
				groupMap[group.Name] = group
			}
		}

		if len(groupOptions) == 0 {
			return fmt.Errorf("no enabled command groups")
		}

		groupChoice, err := ctx.Show(groupOptions, "ql")
		if err != nil {
			// ESC at group level - exit
			return nil
		}

		selectedGroup, exists := groupMap[groupChoice]
		if !exists {
			showErrorNotification("Error", fmt.Sprintf("Unknown group: %s", groupChoice))
			continue
		}

		result := runModuleMenuWithBack(ctx, cfg, selectedGroup, commandMap)

		if result.Success {
			// Command succeeded - exit
			return nil
		}

		// Check error type
		if errors.Is(result.Error, commands.ErrBack) {
			// User pressed "← Back" at module level - continue loop (show groups again)
			continue
		}

		// Any other case (ESC, error, nil) - exit
		return nil
	}
}

func runModuleMenuDirect(ctx launcher.Launcher, cfg *config.Config, group config.ModuleGroup, commandMap map[string]commands.Command) commands.CommandResult {
	for {
		var moduleOptions []string
		moduleToCommand := make(map[string]commands.Command)

		for _, moduleName := range group.Modules {
			cmd, exists := commandMap[moduleName]
			if !exists {
				continue
			}

			if !isCommandEnabled(cfg, cmd.Name) {
				continue
			}

			moduleOptions = append(moduleOptions, cmd.Description)
			moduleToCommand[cmd.Description] = cmd
		}

		if len(moduleOptions) == 0 {
			return commands.CommandResult{
				Success: false,
				Error:   fmt.Errorf("no enabled commands in group"),
			}
		}

		moduleChoice, err := ctx.Show(moduleOptions, group.Name)
		if err != nil {
			// ESC pressed - exit
			return commands.CommandResult{Success: false}
		}

		cmd, ok := moduleToCommand[moduleChoice]
		if !ok {
			showErrorNotification("Error", fmt.Sprintf("Unknown command: %s", moduleChoice))
			continue
		}

		result := cmd.Run(ctx)

		// Command finished - exit (don't loop back)
		return result
	}
}

func runModuleMenuWithBack(ctx launcher.Launcher, cfg *config.Config, group config.ModuleGroup, commandMap map[string]commands.Command) commands.CommandResult {
	for {
		var moduleOptions []string
		moduleToCommand := make(map[string]commands.Command)

		moduleOptions = append(moduleOptions, "← Back")

		for _, moduleName := range group.Modules {
			cmd, exists := commandMap[moduleName]
			if !exists {
				continue
			}

			if !isCommandEnabled(cfg, cmd.Name) {
				continue
			}

			moduleOptions = append(moduleOptions, cmd.Description)
			moduleToCommand[cmd.Description] = cmd
		}

		if len(moduleOptions) == 1 {
			return commands.CommandResult{
				Success: false,
				Error:   fmt.Errorf("no enabled commands in group"),
			}
		}

		moduleChoice, err := ctx.Show(moduleOptions, group.Name)
		if err != nil {
			// ESC pressed - exit completely
			return commands.CommandResult{Success: false, Error: commands.ErrCancelled}
		}

		if moduleChoice == "← Back" {
			// Back button at module level - return to group menu
			return commands.CommandResult{
				Success: false,
				Error:   commands.ErrBack,
			}
		}

		cmd, ok := moduleToCommand[moduleChoice]
		if !ok {
			showErrorNotification("Error", fmt.Sprintf("Unknown command: %s", moduleChoice))
			continue
		}

		result := cmd.Run(ctx)

		// If command succeeded, exit
		if result.Success {
			return result
		}

		// If command returned ErrBack, loop back to module menu (System)
		if errors.Is(result.Error, commands.ErrBack) {
			continue
		}

		// Any other error (cancelled, etc.) - exit completely
		return result
	}
}

func isCommandEnabled(cfg *config.Config, cmdName string) bool {
	commandCfg, exists := cfg.Commands[cmdName]
	if !exists {
		return true
	}

	if enabledVal, ok := commandCfg["enabled"]; ok {
		if enabled, ok := enabledVal.(bool); ok {
			return enabled
		}
	}

	return true
}

func showErrorNotification(title, message string) {
	if _, err := exec.LookPath("dunstify"); err == nil {
		cmd := exec.Command("dunstify",
			"-u", "critical",
			"-t", "5000",
			title,
			message)
		cmd.Env = os.Environ()
		cmd.Start()
		return
	}

	if _, err := exec.LookPath("notify-send"); err == nil {
		cmd := exec.Command("notify-send",
			"-u", "critical",
			"-t", "5000",
			title,
			message)
		cmd.Env = os.Environ()
		cmd.Start()
		return
	}
}

func handleInit() error {
	if err := config.InitUserConfig(); err != nil {
		return err
	}

	configPath := config.GetUserConfigPath()
	fmt.Printf("Config initialized at: %s\n", configPath)
	fmt.Println("\nYou can now edit the config file to customize ql.")
	fmt.Println("Run 'ql' to start using it!")

	return nil
}

func printHelp() {
	fmt.Println("ql - Quick Launcher")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ql [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --init              Initialize user config (~/.config/ql/config.toml)")
	fmt.Println("  --version           Show version information")
	fmt.Println("  --help              Show this help message")
	fmt.Println("  --flat              Use flat menu style")
	fmt.Println("  --grouped           Use grouped menu style")
	fmt.Println("  --launcher NAME     Override launcher (rofi, dmenu, fzf, bemenu, fuzzel)")
	fmt.Println("  --group NAME        Show only commands from specific group")
	fmt.Println()
	fmt.Println("Available groups:")
	fmt.Println("  system, network, media, info")
	fmt.Println()
	fmt.Println("Legacy usage (still supported):")
	fmt.Println("  ql [launcher]       Run ql with specified launcher")
	fmt.Println("  ql init             Initialize config")
	fmt.Println("  ql version          Show version")
	fmt.Println("  ql help             Show help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ql --flat --launcher rofi")
	fmt.Println("  ql --grouped")
	fmt.Println("  ql --group media")
	fmt.Println("  ql --group system --launcher fuzzel")
	fmt.Println("  ql rofi")
	fmt.Println()
	fmt.Println("Config file: ~/.config/ql/config. toml")
}

package main

import (
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
	// Define flags
	initFlag := flag.Bool("init", false, "Initialize user config")
	versionFlag := flag.Bool("version", false, "Show version")
	helpFlag := flag.Bool("help", false, "Show help")
	flatFlag := flag.Bool("flat", false, "Use flat menu style")
	groupedFlag := flag.Bool("grouped", false, "Use grouped menu style")
	launcherFlag := flag.String("launcher", "", "Override launcher (rofi, dmenu, fzf, bemenu, fuzzel)")
	groupFlag := flag.String("group", "", "Show only commands from specific group (system, media, network, info, etc.)")

	flag.Parse()

	// Handle flags
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

	// Legacy positional argument support (for backward compatibility)
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

	// Determine launcher:  --launcher > positional arg > config
	launcherName := cfg.GetDefaultLauncher()

	if *launcherFlag != "" {
		launcherName = *launcherFlag
	} else if len(flag.Args()) > 0 {
		// Support legacy:  ql rofi
		arg := flag.Args()[0]
		if arg != "init" && arg != "version" && arg != "help" {
			launcherName = arg
		}
	}

	ctx, err := launcher.New(launcherName, cfg)
	if err != nil {
		return fmt.Errorf("failed to create launcher: %w", err)
	}

	// If --group is specified, show only that group's commands
	if *groupFlag != "" {
		return runSpecificGroup(ctx, cfg, *groupFlag)
	}

	// Determine menu style: --flat/--grouped > config
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

	// Find the group (case-insensitive)
	var selectedGroup *config.ModuleGroup

	for key, group := range groups {
		if key == groupName || group.Name == groupName {
			selectedGroup = &group
			break
		}
	}

	if selectedGroup == nil {
		// List available groups
		fmt.Fprintf(os.Stderr, "Error: Group '%s' not found\n\n", groupName)
		fmt.Fprintf(os.Stderr, "Available groups:\n")
		for key, group := range groups {
			fmt.Fprintf(os.Stderr, "  %s (%s)\n", key, group.Name)
		}
		return fmt.Errorf("group not found")
	}

	registeredCommands := commands.GetAll()
	commandMap := make(map[string]commands.Command)
	for _, cmd := range registeredCommands {
		commandMap[cmd.Name] = cmd
	}

	// Run the group menu directly WITHOUT back button
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
			return nil
		}

		cmd, ok := optionToCommand[choice]
		if !ok {
			showErrorNotification("Error", fmt.Sprintf("Unknown command: %s", choice))
			continue
		}

		result := cmd.Run(ctx)

		if result.Success {
			return nil
		}

		if result.Error != nil {
			showErrorNotification("Error", result.Error.Error())
		}
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

	for {
		var groupOptions []string
		groupMap := make(map[string]config.ModuleGroup)

		for _, group := range groups {
			hasEnabled := false
			for _, moduleName := range group.Modules {
				if isCommandEnabled(cfg, moduleName) {
					hasEnabled = true
					break
				}
			}

			if hasEnabled {
				label := fmt.Sprintf("%s %s", group.Icon, group.Name)
				groupOptions = append(groupOptions, label)
				groupMap[label] = group
			}
		}

		if len(groupOptions) == 0 {
			return fmt.Errorf("no enabled command groups")
		}

		groupChoice, err := ctx.Show(groupOptions, "ql")
		if err != nil {
			return nil
		}

		selectedGroup := groupMap[groupChoice]

		// Run with back button (navigates back to group selection)
		result := runModuleMenuWithBack(ctx, cfg, selectedGroup, commandMap)

		if result.Success {
			return nil
		}

		if result.Error != nil {
			showErrorNotification("Error", result.Error.Error())
		}

		// Loop continues - shows group menu again
	}
}

// runModuleMenuDirect shows module menu WITHOUT back button (for direct group access)
func runModuleMenuDirect(ctx launcher.Launcher, cfg *config.Config, group config.ModuleGroup, commandMap map[string]commands.Command) commands.CommandResult {
	for {
		var moduleOptions []string
		moduleToCommand := make(map[string]commands.Command)

		// NO back button

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
			// User cancelled - exit
			return commands.CommandResult{Success: false}
		}

		cmd, ok := moduleToCommand[moduleChoice]
		if !ok {
			showErrorNotification("Error", fmt.Sprintf("Unknown command: %s", moduleChoice))
			continue
		}

		result := cmd.Run(ctx)

		if result.Success {
			return result
		}

		if result.Error != nil {
			showErrorNotification("Error", result.Error.Error())
		}

		// Loop continues - shows same group menu again
	}
}

// runModuleMenuWithBack shows module menu WITH back button (for grouped menu navigation)
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
			return commands.CommandResult{Success: false}
		}

		if moduleChoice == "← Back" {
			return commands.CommandResult{Success: false}
		}

		cmd, ok := moduleToCommand[moduleChoice]
		if !ok {
			showErrorNotification("Error", fmt.Sprintf("Unknown command: %s", moduleChoice))
			continue
		}

		result := cmd.Run(ctx)

		if result.Success {
			return result
		}

		if result.Error != nil {
			showErrorNotification("Error", result.Error.Error())
		}
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
	fmt.Printf("Config initialized at:  %s\n", configPath)
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
	fmt.Println("  --init              Initialize user config (~/.config/ql/config. toml)")
	fmt.Println("  --version           Show version information")
	fmt.Println("  --help              Show this help message")
	fmt.Println("  --flat              Use flat menu style")
	fmt.Println("  --grouped           Use grouped menu style")
	fmt.Println("  --launcher NAME     Override launcher (rofi, dmenu, fzf, bemenu, fuzzel)")
	fmt.Println("  --group NAME        Show only commands from specific group")
	fmt.Println()
	fmt.Println("Available groups:")
	fmt.Println("  system, network, media, info, files, productivity, appearance, dev, security")
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
	fmt.Println("  ql --group media           # Show only media commands")
	fmt.Println("  ql --group system --launcher fuzzel")
	fmt.Println("  ql rofi                     # Legacy style")
	fmt.Println()
	fmt.Println("Config file: ~/.config/ql/config.toml")
}

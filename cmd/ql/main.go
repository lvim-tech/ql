package main

import (
	"fmt"
	"os"

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
	// Handle special commands
	if len(os.Args) > 1 {
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

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Determine launcher name
	launcherName := cfg.GetDefaultLauncher()
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg != "init" && arg != "version" && arg != "help" {
			launcherName = arg
		}
	}

	// Create launcher context
	ctx, err := launcher.New(launcherName, cfg)
	if err != nil {
		return fmt.Errorf("failed to create launcher:  %w", err)
	}

	// Get menu style
	menuStyle := cfg.GetMenuStyle()

	if menuStyle == "grouped" {
		return runGroupedMenu(ctx, cfg)
	}

	return runFlatMenu(ctx, cfg)
}

func runFlatMenu(ctx launcher.Launcher, cfg *config.Config) error {
	// Get all registered commands
	registeredCommands := commands.GetAll()
	if len(registeredCommands) == 0 {
		return fmt.Errorf("no commands registered")
	}

	// Create command map
	commandMap := make(map[string]commands.Command)
	for _, cmd := range registeredCommands {
		commandMap[cmd.Name] = cmd
	}

	// Get module order
	moduleOrder := cfg.GetModuleOrder()
	if len(moduleOrder) == 0 {
		// Fallback to all registered commands
		for _, cmd := range registeredCommands {
			moduleOrder = append(moduleOrder, cmd.Name)
		}
	}

	// Build options in specified order
	var options []string
	optionToCommand := make(map[string]commands.Command)

	for _, moduleName := range moduleOrder {
		cmd, exists := commandMap[moduleName]
		if !exists {
			continue
		}

		// Check if enabled
		if !isCommandEnabled(cfg, cmd.Name) {
			continue
		}

		options = append(options, cmd.Description)
		optionToCommand[cmd.Description] = cmd
	}

	if len(options) == 0 {
		return fmt.Errorf("no enabled commands")
	}

	// Show menu
	choice, err := ctx.Show(options, "ql")
	if err != nil {
		return err
	}

	// Execute selected command
	cmd, ok := optionToCommand[choice]
	if !ok {
		return fmt.Errorf("unknown command: %s", choice)
	}

	return cmd.Run(ctx)
}

func runGroupedMenu(ctx launcher.Launcher, cfg *config.Config) error {
	// Get all registered commands
	registeredCommands := commands.GetAll()
	if len(registeredCommands) == 0 {
		return fmt.Errorf("no commands registered")
	}

	// Create command map
	commandMap := make(map[string]commands.Command)
	for _, cmd := range registeredCommands {
		commandMap[cmd.Name] = cmd
	}

	// Get module groups
	groups := cfg.GetModuleGroups()
	if len(groups) == 0 {
		// Fallback to flat menu
		return runFlatMenu(ctx, cfg)
	}

	// Build group options
	var groupOptions []string
	groupMap := make(map[string]config.ModuleGroup)

	for _, group := range groups {
		// Check if group has any enabled modules
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

	// Show group menu
	groupChoice, err := ctx.Show(groupOptions, "ql")
	if err != nil {
		return err
	}

	selectedGroup := groupMap[groupChoice]

	// Build module options for selected group
	var moduleOptions []string
	moduleToCommand := make(map[string]commands.Command)

	for _, moduleName := range selectedGroup.Modules {
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
		return fmt.Errorf("no enabled commands in group")
	}

	// Show module menu
	moduleChoice, err := ctx.Show(moduleOptions, selectedGroup.Name)
	if err != nil {
		return err
	}

	// Execute selected command
	cmd, ok := moduleToCommand[moduleChoice]
	if !ok {
		return fmt.Errorf("unknown command: %s", moduleChoice)
	}

	return cmd.Run(ctx)
}

// isCommandEnabled checks if a module is enabled in config
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
	fmt.Println("  ql [launcher]    - Run ql with specified launcher (default: rofi)")
	fmt.Println("  ql init          - Initialize user config (~/.config/ql/config. toml)")
	fmt.Println("  ql version       - Show version information")
	fmt.Println("  ql help          - Show this help message")
	fmt.Println()
	fmt.Println("Available launchers:")
	fmt.Println("  rofi, dmenu, fzf, bemenu, fuzzel")
	fmt.Println()
	fmt.Println("Config file: ~/.config/ql/config.toml")
}

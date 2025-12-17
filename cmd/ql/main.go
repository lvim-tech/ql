package main

import (
	"fmt"
	"os"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/launcher"

	// Import commands
	_ "github.com/lvim-tech/ql/pkg/commands/audiorecord"
	_ "github.com/lvim-tech/ql/pkg/commands/mpc"
	_ "github.com/lvim-tech/ql/pkg/commands/power"
	_ "github.com/lvim-tech/ql/pkg/commands/radio"
	_ "github.com/lvim-tech/ql/pkg/commands/screenshot"
	_ "github.com/lvim-tech/ql/pkg/commands/videorecord"
	_ "github.com/lvim-tech/ql/pkg/commands/wifi"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Check for subcommands
	if len(os.Args) > 1 {
		cmdName := os.Args[1]

		// Handle special commands
		switch cmdName {
		case "version":
			fmt.Println("ql version 0.1.0")
			return
		case "help", "-h", "--help":
			showHelp()
			return
		case "init":
			if err := config.InitUserConfig(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Config initialized at:", config.GetUserConfigPath())
			return
		}

		// Check if command is enabled in config
		if !isCommandEnabled(cmdName, cfg) {
			fmt.Fprintf(os.Stderr, "Error: %s module is disabled in config\n", cmdName)
			os.Exit(1)
		}

		// Try to run the command
		cmd := commands.Find(cmdName)
		if cmd == nil {
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmdName)
			fmt.Fprintf(os.Stderr, "Run 'ql help' for usage\n")
			os.Exit(1)
		}

		// Parse flags (simple implementation)
		flags := parseFlags(os.Args[2:])

		// Create launcher context
		ctx := launcher.NewContextFromFlags(flags)

		// Run command
		if err := cmd.Run(ctx); err != nil {
			if launcher.IsCancelled(err) {
				os.Exit(0)
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// No subcommand - show main menu
	showMainMenu(cfg)
}

// isCommandEnabled checks if a command is enabled in config
func isCommandEnabled(cmdName string, cfg *config.Config) bool {
	switch cmdName {
	case "power":
		return cfg.Commands.Power.Enabled
	case "screenshot":
		return cfg.Commands.Screenshot.Enabled
	case "radio":
		return cfg.Commands.Radio.Enabled
	case "wifi":
		return cfg.Commands.Wifi.Enabled
	case "mpc":
		return cfg.Commands.Mpc.Enabled
	default:
		return true // Unknown commands are allowed (will fail later)
	}
}

func showMainMenu(cfg *config.Config) {
	// Get all registered commands
	allCommands := commands.List()

	// Filter enabled commands
	var enabledCommands []string
	var enabledCommandNames []string
	for _, cmd := range allCommands {
		if isCommandEnabled(cmd.Name, cfg) {
			enabledCommands = append(enabledCommands, cmd.Description)
			enabledCommandNames = append(enabledCommandNames, cmd.Name)
		}
	}

	if len(enabledCommands) == 0 {
		fmt.Fprintf(os.Stderr, "No commands enabled in config\n")
		os.Exit(1)
	}

	// Create launcher context
	flags := parseFlags(os.Args[1:])
	ctx := launcher.NewContextFromFlags(flags)

	// Show menu
	choice, err := ctx.Show(enabledCommands, "ql")
	if err != nil {
		if launcher.IsCancelled(err) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Find and run the selected command
	for i, desc := range enabledCommands {
		if desc == choice {
			cmdName := enabledCommandNames[i]
			cmd := commands.Find(cmdName)
			if cmd == nil {
				fmt.Fprintf(os.Stderr, "Command not found: %s\n", cmdName)
				os.Exit(1)
			}

			if err := cmd.Run(ctx); err != nil {
				if launcher.IsCancelled(err) {
					os.Exit(0)
				}
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	fmt.Fprintf(os.Stderr, "Command not found\n")
	os.Exit(1)
}

func showHelp() {
	fmt.Println("ql - Quick Launcher for Linux")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ql [command] [flags]")
	fmt.Println()
	fmt.Println("Available Commands:")

	for _, cmd := range commands.List() {
		fmt.Printf("  %-12s %s\n", cmd.Name, cmd.Description)
	}

	fmt.Println()
	fmt.Println("Special Commands:")
	fmt.Println("  version      Show version")
	fmt.Println("  help         Show this help")
	fmt.Println("  init         Initialize user config")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -r           Use rofi launcher")
	fmt.Println("  -d           Use dmenu launcher")
	fmt.Println("  -f           Use fzf launcher")
	fmt.Println("  -b           Use bemenu launcher")
	fmt.Println("  -z           Use fuzzel launcher")
}

func parseFlags(args []string) map[string]bool {
	flags := make(map[string]bool)
	for _, arg := range args {
		if len(arg) > 1 && arg[0] == '-' {
			flag := arg[1:]
			flags[flag] = true
		}
	}
	return flags
}

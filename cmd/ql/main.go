package main

import (
	"fmt"
	"os"

	"github.com/lvim-tech/ql/pkg/commands"
	_ "github.com/lvim-tech/ql/pkg/commands/hub"
	_ "github.com/lvim-tech/ql/pkg/commands/power"
	_ "github.com/lvim-tech/ql/pkg/commands/screenshot"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/launcher"
	"github.com/spf13/cobra"
)

var (
	useDmenu  bool
	useRofi   bool
	useFzf    bool
	useBemenu bool
	useFuzzel bool
)

var rootCmd = &cobra.Command{
	Use:   "ql",
	Short: "Quick Launch - Modern launcher scripts in Go",
	Long: `ql is a modern rewrite of dmscripts with support for multiple launchers.  
It provides quick access to common tasks like power management, screenshots, and more.   

Supported launchers:  dmenu, rofi, fzf, bemenu, fuzzel`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := getLauncherContext()

		hubCmd := commands.Find("hub")
		if hubCmd == nil {
			fmt.Fprintln(os.Stderr, "Error: hub command not found")
			os.Exit(1)
		}

		if err := hubCmd.Run(ctx); err != nil {
			// Graceful exit при cancel
			if launcher.IsCancelled(err) {
				os.Exit(0)
			}

			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var initConfigCmd = &cobra.Command{
	Use:   "init-config",
	Short: "Initialize user configuration file",
	Long: `Creates a default configuration file in ~/.config/ql/config. toml
You can then edit this file to customize ql's behavior. 

Example: 
  ql init-config              # Create config
  ql init-config --force      # Overwrite existing config`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.InitUserConfig(); err != nil {
			// Провери дали е заради съществуващ файл
			forceFlag, _ := cmd.Flags().GetBool("force")
			if forceFlag {
				// Force overwrite
				userConfigPath := config.GetUserConfigPath()
				content := config.GetDefaultConfigContent()
				if err := os.WriteFile(userConfigPath, []byte(content), 0644); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to overwrite config: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("✓ Configuration file overwritten:  %s\n", userConfigPath)
				fmt.Println("  Edit this file to customize ql's behavior")
				return
			}

			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintln(os.Stderr, "\nUse --force to overwrite existing config")
			os.Exit(1)
		}

		fmt.Printf("✓ Configuration file created:  %s\n", config.GetUserConfigPath())
		fmt.Println("  Edit this file to customize ql's behavior")
	},
}

var showConfigCmd = &cobra.Command{
	Use:   "show-config",
	Short: "Show current configuration",
	Long:  `Displays the merged configuration (defaults + user overrides)`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Get()

		fmt.Println("ql Configuration")
		fmt.Println("================")
		fmt.Printf("\nDefault Launcher: %s\n", cfg.DefaultLauncher)

		fmt.Println("\nLauncher Configurations:")
		fmt.Println("------------------------")
		launchers := []string{"dmenu", "rofi", "fzf", "bemenu", "fuzzel"}
		for _, name := range launchers {
			launcherCmd := cfg.GetLauncherCommand(name)
			if launcherCmd != nil {
				fmt.Printf("  %s:\n", name)
				fmt.Printf("    command: %s\n", launcherCmd.Command)
				fmt.Printf("    args: %v\n", launcherCmd.Args)
			}
		}

		fmt.Println("\nCommand Settings:")
		fmt.Println("-----------------")

		// Power settings
		fmt.Printf("  power:\n")
		fmt.Printf("    enabled: %v\n", cfg.Commands.Power.Enabled)
		fmt.Printf("    show:     logout=%v suspend=%v hibernate=%v reboot=%v shutdown=%v\n",
			cfg.Commands.Power.ShowLogout,
			cfg.Commands.Power.ShowSuspend,
			cfg.Commands.Power.ShowHibernate,
			cfg.Commands.Power.ShowReboot,
			cfg.Commands.Power.ShowShutdown)
		fmt.Printf("    confirm: logout=%v suspend=%v hibernate=%v reboot=%v shutdown=%v\n",
			cfg.Commands.Power.ConfirmLogout,
			cfg.Commands.Power.ConfirmSuspend,
			cfg.Commands.Power.ConfirmHibernate,
			cfg.Commands.Power.ConfirmReboot,
			cfg.Commands.Power.ConfirmShutdown)
		fmt.Printf("    commands:\n")
		fmt.Printf("      logout:     %s\n", cfg.Commands.Power.LogoutCommand)
		fmt.Printf("      suspend:   %s\n", cfg.Commands.Power.SuspendCommand)
		fmt.Printf("      hibernate: %s\n", cfg.Commands.Power.HibernateCommand)
		fmt.Printf("      reboot:    %s\n", cfg.Commands.Power.RebootCommand)
		fmt.Printf("      shutdown:  %s\n", cfg.Commands.Power.ShutdownCommand)

		// Screenshot settings
		fmt.Printf("\n  screenshot:\n")
		fmt.Printf("    enabled:     %v\n", cfg.Commands.Screenshot.Enabled)
		fmt.Printf("    save_dir:    %s\n", cfg.Commands.Screenshot.SaveDir)
		fmt.Printf("    file_prefix: %s\n", cfg.Commands.Screenshot.FilePrefix)

		fmt.Printf("\nConfig file location: %s\n", config.GetUserConfigPath())
		if _, err := os.Stat(config.GetUserConfigPath()); os.IsNotExist(err) {
			fmt.Println("(using defaults - no user config found)")
			fmt.Println("\nRun 'ql init-config' to create a config file")
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("ql version 0.1.0")
		fmt.Println("Modern launcher scripts in Go")
		fmt.Println("https://github.com/lvim-tech/ql")
	},
}

// getLauncherContext създава launcher context от флаговете
func getLauncherContext() *launcher.Context {
	flags := map[string]bool{
		"d": useDmenu,
		"r": useRofi,
		"f": useFzf,
		"b": useBemenu,
		"z": useFuzzel,
	}
	return launcher.NewContextFromFlags(flags)
}

// registerDynamicCommands добавя всички регистрирани команди към Cobra
func registerDynamicCommands() {
	for _, cmd := range commands.List() {
		// Skip hub - той се вика от rootCmd директно
		if cmd.Name == "hub" {
			continue
		}

		// Създай Cobra команда за всяка регистрирана команда
		command := cmd // Capture loop variable
		cobraCmd := &cobra.Command{
			Use:   command.Name,
			Short: command.Description,
			Run: func(cobraCmd *cobra.Command, args []string) {
				ctx := getLauncherContext()

				if err := command.Run(ctx); err != nil {
					// Graceful exit при cancel
					if launcher.IsCancelled(err) {
						os.Exit(0)
					}

					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(1)
				}
			},
		}

		rootCmd.AddCommand(cobraCmd)
	}
}

func init() {
	// Зареди config при startup
	if _, err := config.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		fmt.Fprintln(os.Stderr, "Using default configuration")
	}

	// Регистрирай launcher flags
	rootCmd.PersistentFlags().BoolVarP(&useDmenu, "dmenu", "d", false, "Use dmenu launcher")
	rootCmd.PersistentFlags().BoolVarP(&useRofi, "rofi", "r", false, "Use rofi launcher")
	rootCmd.PersistentFlags().BoolVarP(&useFzf, "fzf", "f", false, "Use fzf launcher")
	rootCmd.PersistentFlags().BoolVarP(&useBemenu, "bemenu", "b", false, "Use bemenu launcher")
	rootCmd.PersistentFlags().BoolVarP(&useFuzzel, "fuzzel", "z", false, "Use fuzzel launcher")

	// Добави флагове за init-config
	initConfigCmd.Flags().Bool("force", false, "Force overwrite existing config")

	// Добави статични команди
	rootCmd.AddCommand(initConfigCmd)
	rootCmd.AddCommand(showConfigCmd)
	rootCmd.AddCommand(versionCmd)

	// Добави динамично регистрираните команди
	registerDynamicCommands()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

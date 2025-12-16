package test

import (
	"fmt"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/launcher"
)

func init() {
	commands.Register(commands.Command{
		Name:        "test",
		Description: "Test launcher functionality",
		Run:         Run,
	})
}

// Run показва тестово menu
func Run(ctx *launcher.Context) error {
	options := []string{
		"Option 1 - Hello",
		"Option 2 - World",
		"Option 3 - Launcher Test",
		"Option 4 - Exit",
	}

	choice, err := ctx.Show(options, "Test Menu")
	if err != nil {
		return err
	}

	fmt.Printf("✓ You selected: %s\n", choice)
	fmt.Printf("  Using launcher: %s\n", ctx.LauncherName())

	return nil
}

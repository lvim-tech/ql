package launcher

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/utils"
)

// external runs one of the menu programs. The five backends differ only in
// the binary name and how the prompt is spelled, so they are entries in a
// table — not five copies of the pipe-and-scan dance this file does once.
type external struct {
	baseLauncher
	name       string
	promptArgs func(prompt string) []string
	// fzf draws its whole interface on the terminal through stderr; the GUI
	// menus must not inherit it.
	wantStderr bool
}

var externals = map[string]struct {
	promptArgs func(string) []string
	wantStderr bool
}{
	"rofi":   {func(p string) []string { return []string{"-p", p} }, false},
	"dmenu":  {func(p string) []string { return []string{"-p", p} }, false},
	"bemenu": {func(p string) []string { return []string{"-p", p} }, false},
	"fzf":    {func(p string) []string { return []string{"--prompt", p + "> "} }, true},
	"fuzzel": {func(p string) []string { return []string{"--prompt", p + "> "} }, false},
}

func newExternal(name string, cfg *config.Config) *external {
	spec := externals[name]
	return &external{
		baseLauncher: baseLauncher{cfg: cfg},
		name:         name,
		promptArgs:   spec.promptArgs,
		wantStderr:   spec.wantStderr,
	}
}

func (e *external) Show(options []string, prompt string) (string, error) {
	launcherCfg := e.cfg.GetLauncherConfig(e.name)
	// Concat, not append: append would scribble into the config's backing
	// array once anything holds a slice with spare capacity.
	args := slices.Concat(launcherCfg.Args, e.promptArgs(prompt))

	cmd := exec.Command(e.name, args...)
	if e.wantStderr {
		cmd.Stderr = os.Stderr
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		// Callers treat any Show error as "user cancelled", so a missing
		// menu binary is invisible without this line.
		utils.Logf("[launcher %s] failed to start: %v", e.name, err)
		return "", fmt.Errorf("failed to start %s: %w", e.name, err)
	}

	for _, option := range options {
		fmt.Fprintln(stdin, option)
	}
	stdin.Close()

	scanner := bufio.NewScanner(stdout)
	var choice string
	if scanner.Scan() {
		choice = strings.TrimSpace(scanner.Text())
	}

	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("%s exited with error: %w", e.name, err)
	}
	if choice == "" {
		return "", fmt.Errorf("no selection made")
	}
	return choice, nil
}

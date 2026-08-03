// Package pass fronts the standard unix password store. The division of
// labour follows keyforge's principles: pass IS the store (no new
// cryptography), keyforge generates and audits, gpg/pinentry unlock — ql is
// only the menu. No secret ever crosses a command line or appears in a
// menu; passwords travel by stdin (wtype) or through pass's own clipboard
// handling, and the log only ever sees entry NAMES.
package pass

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/utils"
)

func init() {
	commands.Register(commands.Command{
		Name:        "pass",
		Description: "Passwords (pass)",
		Run:         Run,
	})
}

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfg := commands.DecodeConfig(ctx, "pass", DefaultConfig())

	if !cfg.Enabled {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("pass module is disabled in config"),
		}
	}

	notifCfg := ctx.Config().GetNotificationConfig()

	if !utils.CommandExists("pass") {
		err := fmt.Errorf("pass is not installed")
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Pass Error", err.Error())
		return commands.CommandResult{Success: false, Error: err}
	}

	// Direct verbs for keybindings: `ql pass type` / `ql pass copy` go
	// straight to the entry list and perform that one action.
	verb := ""
	if args := ctx.Args(); len(args) > 0 {
		verb = strings.ToLower(args[0])
	}

	entries, err := listEntries(&cfg)
	if err != nil {
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Pass Error", err.Error())
		return commands.CommandResult{Success: false, Error: err}
	}
	if len(entries) == 0 {
		err := fmt.Errorf("the password store is empty")
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Pass Error", err.Error())
		return commands.CommandResult{Success: false, Error: err}
	}

	// QL_PASS_FILTER narrows the list to one site — the qutebrowser
	// userscript sets it to the page's domain, so the menu offers the two
	// logins for THIS site instead of the whole store. A filter that
	// matches nothing is ignored rather than leaving an empty menu.
	if filter := strings.ToLower(os.Getenv("QL_PASS_FILTER")); filter != "" {
		var matched []string
		for _, e := range entries {
			if strings.Contains(strings.ToLower(e), filter) {
				matched = append(matched, e)
			}
		}
		if len(matched) > 0 {
			entries = matched
		}
	}

	for {
		var options []string
		if !ctx.IsDirectLaunch() {
			options = append(options, "← Back")
		}
		options = append(options, entries...)

		entry, err := ctx.Show(options, "Passwords")
		if err != nil {
			return commands.CommandResult{Success: false}
		}
		if entry == "← Back" {
			return commands.CommandResult{Success: false, Error: commands.ErrBack}
		}

		var action string
		if verb != "" {
			action = verb
		} else {
			choice, err := ctx.Show(
				[]string{"← Back", "Copy password", "Type password", "Copy OTP", "Open keyforge"},
				entry)
			if err != nil {
				return commands.CommandResult{Success: false}
			}
			if choice == "← Back" {
				continue // back to the entry list
			}
			action = map[string]string{
				"Copy password": "copy",
				"Type password": "type",
				"Copy OTP":      "otp",
				"Open keyforge": "keyforge",
			}[choice]
		}

		if err := perform(action, entry, &cfg, &notifCfg); err != nil {
			utils.ShowErrorNotificationWithConfig(&notifCfg, "Pass Error", err.Error())
			return commands.CommandResult{Success: false, Error: err}
		}
		return commands.CommandResult{Success: true}
	}
}

func perform(action, entry string, cfg *Config, notifCfg *config.NotificationConfig) error {
	switch action {
	case "copy":
		// pass owns the clipboard dance, including the 45-second clear.
		if out, err := exec.Command("pass", "show", "-c", entry).CombinedOutput(); err != nil {
			return fmt.Errorf("pass show -c %s: %s", entry, strings.TrimSpace(string(out)))
		}
		utils.NotifyWithConfig(notifCfg, "Pass", entry+" copied — clipboard clears in 45s")
		return nil

	case "type":
		if !utils.CommandExists("wtype") {
			return fmt.Errorf("wtype is not installed (needed for typing)")
		}
		out, err := exec.Command("pass", "show", entry).Output()
		if err != nil {
			return fmt.Errorf("pass show %s failed", entry)
		}
		password, _, _ := strings.Cut(string(out), "\n")
		// Let the menu window close and focus return to the target field —
		// typing into the dying menu is the classic autotype failure.
		time.Sleep(time.Duration(cfg.TypeDelayMs) * time.Millisecond)
		cmd := exec.Command("wtype", "-")
		cmd.Stdin = strings.NewReader(password)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("wtype failed: %w", err)
		}
		return nil

	case "otp":
		if out, err := exec.Command("pass", "otp", "-c", entry).CombinedOutput(); err != nil {
			return fmt.Errorf("pass otp %s: %s", entry, strings.TrimSpace(string(out)))
		}
		utils.NotifyWithConfig(notifCfg, "Pass", "OTP for "+entry+" copied")
		return nil

	case "keyforge":
		terminal := utils.DetectTerminal()
		if terminal == "" {
			return fmt.Errorf("no terminal emulator found")
		}
		cmd := exec.Command(terminal, "-e", cfg.Keyforge)
		cmd.Env = os.Environ()
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to open %s: %w", cfg.Keyforge, err)
		}
		go cmd.Wait()
		return nil

	default:
		return fmt.Errorf("unknown pass action: %s (use: copy, type, otp, keyforge)", action)
	}
}

// listEntries walks the store for *.gpg files — names only, no decryption.
func listEntries(cfg *Config) ([]string, error) {
	store := utils.ExpandHomeDir(cfg.Store)
	var entries []string
	err := filepath.WalkDir(store, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == ".extensions" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".gpg") {
			rel, err := filepath.Rel(store, path)
			if err != nil {
				return nil
			}
			entries = append(entries, strings.TrimSuffix(rel, ".gpg"))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", store, err)
	}
	sort.Strings(entries)
	return entries, nil
}

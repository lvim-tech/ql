package launcher

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lvim-tech/ql/pkg/config"
)

// The prompt flag comes from the table, never from the config args — the
// shipped defaults used to list it there too, producing `dmenu -i -p -p ql`
// and an fzf --prompt that swallowed --height=40%.
func TestPromptArgsPerBackend(t *testing.T) {
	want := map[string][]string{
		"rofi":   {"-p", "ql"},
		"dmenu":  {"-p", "ql"},
		"bemenu": {"-p", "ql"},
		"fzf":    {"--prompt", "ql> "},
		"fuzzel": {"--prompt", "ql> "},
	}
	for name, expected := range want {
		spec, ok := externals[name]
		if !ok {
			t.Fatalf("backend %s missing from the table", name)
		}
		got := spec.promptArgs("ql")
		if len(got) != len(expected) {
			t.Fatalf("%s: prompt args %v, want %v", name, got, expected)
		}
		for i := range got {
			if got[i] != expected[i] {
				t.Errorf("%s: prompt args %v, want %v", name, got, expected)
			}
		}
	}
}

// An unknown launcher is an error — the old default arm silently ran rofi,
// which is also why "auto" appeared to work while not existing at all.
func TestUnknownLauncherErrors(t *testing.T) {
	if _, err := New("wleave", &config.Config{}); err == nil {
		t.Error("unknown launcher name did not error")
	}
}

// Auto resolves to something usable everywhere: with no menu binaries on
// PATH and no terminal it still names a launcher rather than failing.
func TestAutoAlwaysResolves(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("WAYLAND_DISPLAY", "")
	if _, err := New("auto", &config.Config{}); err != nil {
		t.Errorf("auto did not resolve: %v", err)
	}
}

// The TUI model: enter returns the highlighted option, escape returns
// nothing — Show turns that into the cancel error every caller expects.
func TestTUIModelSelectsAndCancels(t *testing.T) {
	model := tuiModel{list: newTUIList([]string{"first", "second"}, "ql")}
	next, _ := model.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	next, _ = next.(tuiModel).Update(tea.KeyMsg{Type: tea.KeyDown})
	next, _ = next.(tuiModel).Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := next.(tuiModel).choice; got != "second" {
		t.Errorf("enter selected %q, want second", got)
	}

	model = tuiModel{list: newTUIList([]string{"first", "second"}, "ql")}
	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if got := next.(tuiModel).choice; got != "" {
		t.Errorf("escape still chose %q", got)
	}
}

// Under Wayland with everything installed, auto picks rofi: it runs on both
// display servers and carries the user's theming, so the Wayland-native
// menus are a fallback, not a silent substitution.
func TestAutoPrefersRofiOnWayland(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"rofi", "fuzzel", "bemenu", "dmenu"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")

	if got := pickAuto(nil); got != "rofi" && got != "tui" {
		t.Errorf("auto picked %q, want rofi (tui is fine only when the test runs on a TTY)", got)
	}
}

// A launcher installed outside the session's PATH is reachable only by path,
// and BOTH the run and the auto-detection have to honour it. The detection is
// the half that was missing: pickAuto asked PATH, found nothing, and chose a
// menu the user had configured against.
func TestConfiguredCommandIsFoundAndPreferred(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(dir, "elsewhere")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	rofi := filepath.Join(outside, "rofi")
	if err := os.WriteFile(rofi, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A PATH with a fuzzel in it and no rofi: exactly the case that misfired.
	onpath := filepath.Join(dir, "bin")
	if err := os.MkdirAll(onpath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(onpath, "fuzzel"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", onpath)
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")

	cfg := &config.Config{Launchers: map[string]config.LauncherConfig{
		"rofi": {Command: rofi},
	}}

	if got := binaryFor(cfg, "rofi"); got != rofi {
		t.Errorf("binaryFor = %q, want the configured path %q", got, rofi)
	}
	if !available(rofi) {
		t.Errorf("available(%q) = false for a file that is there and executable", rofi)
	}
	if got := pickAuto(cfg); got != "rofi" && got != "tui" {
		t.Errorf("auto picked %q; the configured rofi should win over the fuzzel on PATH", got)
	}
	// Without the config it must still fall back to what PATH actually holds.
	if got := pickAuto(nil); got != "fuzzel" && got != "tui" {
		t.Errorf("auto without config picked %q, want fuzzel — the only one on PATH", got)
	}
}

package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A compositor keybinding or a waybar module runs without the interactive shell's PATH, and
// clipack's bin is deliberately kept out of the session's. Before this, DetectTerminal simply
// skipped anything installed there and returned the next terminal down its list.
func TestLookFindsWhatPathDoesNot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", filepath.Join(home, "empty"))

	bin := filepath.Join(home, "clipack", "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	term := filepath.Join(bin, "kitty")
	if err := os.WriteFile(term, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := Look("kitty")
	if got != term {
		t.Fatalf("Look = %q, want %q", got, term)
	}
	if !filepath.IsAbs(got) {
		t.Error("the path must be absolute — the caller execs it in the same poor environment")
	}
	if d := DetectTerminal(); !strings.HasSuffix(d, "/kitty") {
		t.Errorf("DetectTerminal = %q, want the kitty it can actually reach", d)
	}
}

// A non-executable file of the right name is not a terminal.
func TestLookIgnoresANonExecutable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", filepath.Join(home, "empty"))
	bin := filepath.Join(home, "clipack", "bin")
	os.MkdirAll(bin, 0o755)
	os.WriteFile(filepath.Join(bin, "kitty"), []byte("not a program"), 0o644)

	if got := Look("kitty"); got != "" {
		t.Errorf("Look = %q, want empty", got)
	}
}

// neomutt's colour commands emit #rrggbb, and xterm-kitty's terminfo declares 256 colours — so the
// direct-colour entry has to be asked for explicitly. It cannot be done through the environment:
// kitty sets its child's TERM from its own `term` option and overwrites whatever it inherited.
func TestKittyIsGivenItsDirectColourTerm(t *testing.T) {
	args := TerminalArgs("/home/someone/clipack/bin/kitty")
	want := []string{"-o", "term=kitty-direct"}
	if len(args) != len(want) || args[0] != want[0] || args[1] != want[1] {
		t.Errorf("TerminalArgs = %v, want %v", args, want)
	}

	// Matched on the BASE name: the detector returns an absolute path now, so a check against
	// the whole string would never fire.
	if len(TerminalArgs("kitty")) != 2 {
		t.Error("a bare name is the same terminal")
	}

	// Every other terminal is left alone rather than guessed at.
	for _, other := range []string{"/usr/bin/foot", "alacritty", "/usr/bin/wezterm"} {
		if got := TerminalArgs(other); got != nil {
			t.Errorf("TerminalArgs(%q) = %v, want nothing", other, got)
		}
	}
}

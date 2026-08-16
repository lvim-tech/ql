package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestShellQuoteNeutralisesExpansion is the point of the helper: what comes
// back must survive `sh -c` as literal text, whatever it contains.
func TestShellQuoteNeutralisesExpansion(t *testing.T) {
	cases := []string{
		"plain",
		"with space",
		"$(touch /tmp/should-not-exist)",
		"`id`",
		"$HOME",
		"back\\slash",
		"it's",
		"'; id; '",
		`double"quote`,
		"新しい",
		"",
	}

	for _, in := range cases {
		out, err := exec.Command("sh", "-c", "printf %s "+ShellQuote(in)).Output()
		if err != nil {
			t.Fatalf("ShellQuote(%q): sh refused the result %s: %v", in, ShellQuote(in), err)
		}
		if string(out) != in {
			t.Errorf("ShellQuote(%q): shell saw %q, want it unchanged", in, string(out))
		}
	}
}

// TestShellQuoteBeatsGoQuoting pins the reason %q was not good enough.
func TestShellQuoteBeatsGoQuoting(t *testing.T) {
	const payload = "$(echo pwned)"

	out, err := exec.Command("sh", "-c", "printf %s "+ShellQuote(payload)).Output()
	if err != nil {
		t.Fatalf("sh failed: %v", err)
	}
	if string(out) == "pwned" {
		t.Fatal("command substitution still ran — the value was not quoted for the shell")
	}
	if string(out) != payload {
		t.Errorf("got %q, want %q", string(out), payload)
	}
}

func TestRuntimeStatePathIsPrivateAndUnderRuntimeDir(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", base)

	path := RuntimeStatePath("audiorecord.pid")
	if path == "" {
		t.Fatal("RuntimeStatePath returned empty with a usable XDG_RUNTIME_DIR")
	}
	if want := filepath.Join(base, "ql", "audiorecord.pid"); path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	if strings.HasPrefix(path, "/tmp/ql_") {
		t.Error("state is still at a fixed, world-writable /tmp name")
	}

	info, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("runtime dir not created: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0700 {
		t.Errorf("runtime dir mode = %#o, want 0700 — other users must not reach it", perm)
	}
}

// Without XDG_RUNTIME_DIR the directory still has to be private and ours,
// rather than a predictable name anybody can pre-create.
func TestRuntimeDirFallbackIsStillPrivate(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "")
	t.Setenv("TMPDIR", t.TempDir())

	dir := RuntimeDir()
	if dir == "" {
		t.Fatal("RuntimeDir returned empty")
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("fallback dir not created: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0700 {
		t.Errorf("fallback dir mode = %#o, want 0700", perm)
	}
}

package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeUserConfig points Load() at a throwaway config via $HOME.
func writeUserConfig(t *testing.T, body string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".config", "ql")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The old merge computed enabled = user || default with a true default, so
// notifications could never be turned off.
func TestNotificationsCanBeDisabled(t *testing.T) {
	writeUserConfig(t, "[notifications]\nenabled = false\n")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Notifications.Enabled {
		t.Error("enabled = false in user config did not disable notifications")
	}
}

// Omitting the keys keeps the defaults — the old merge copied
// show_in_terminal unconditionally, forcing it false when absent.
func TestOmittedNotificationKeysKeepDefaults(t *testing.T) {
	writeUserConfig(t, "menu_style = \"flat\"\n")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Notifications.Enabled {
		t.Error("default enabled = true was lost with no user notification block")
	}
}

// A group override naming ONE field must not wipe the others: the old
// whole-value maps.Copy made [module_groups.system] with only a name erase
// the group's modules and disable it.
func TestPartialGroupOverrideKeepsModules(t *testing.T) {
	writeUserConfig(t, "[module_groups.system]\nname = \"Sys\"\n")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	group, ok := cfg.ModuleGroups["system"]
	if !ok {
		t.Fatal("system group vanished")
	}
	if group.Name != "Sys" {
		t.Errorf("name override lost: %q", group.Name)
	}
	if !group.Enabled {
		t.Error("enabled was wiped by a partial override")
	}
	if len(group.Modules) == 0 {
		t.Error("modules were wiped by a partial override")
	}
}

// A malformed user config must not brick the launcher — it falls back to
// the defaults instead of refusing to start.
func TestMalformedUserConfigFallsBackToDefaults(t *testing.T) {
	writeUserConfig(t, "this is [not toml")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("malformed config bricked Load: %v", err)
	}
	if cfg.DefaultLauncher != "auto" {
		t.Errorf("defaults not in effect, default_launcher = %q", cfg.DefaultLauncher)
	}
}

package kb

import (
	"os"
	"path/filepath"
	"testing"
)

// The layouts come from the session, never from a hardcoded list: mango
// writes them in its own config as xkb_rules_layout=us,bg.
func TestDetectLayoutsReadsMangoConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".config", "mango")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# keyboard\nrepeat_rate=25\nxkb_rules_layout=us,bg,de\n"
	if err := os.WriteFile(filepath.Join(dir, "keyboard.conf"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	got := detectLayouts()
	if len(got) != 3 || got[0] != "us" || got[1] != "bg" || got[2] != "de" {
		t.Fatalf("detected %v, want [us bg de] from the mango config", got)
	}
}

// "next" walks the detected list; en and us are the same layout under two
// spellings, so the current layout must be recognised through the alias or
// next would always land back on the first entry.
func TestNeighbourLayoutWrapsAndKnowsAliases(t *testing.T) {
	layouts := []string{"us", "bg"}

	if !sameLayout("en", "us") {
		t.Error("en and us are the same layout, the alias table missed it")
	}

	for _, tc := range []struct{ current, want string }{
		{"us", "bg"},
		{"bg", "us"},
	} {
		index := 0
		for i, l := range layouts {
			if sameLayout(l, tc.current) {
				index = i
			}
		}
		if got := layouts[(index+1)%len(layouts)]; got != tc.want {
			t.Errorf("next from %s = %s, want %s", tc.current, got, tc.want)
		}
	}
}

// The index backends (Hyprland, niri, sway) need the position, and a name
// they do not know must not silently switch to something random.
func TestLayoutIndex(t *testing.T) {
	layouts := []string{"us", "bg"}
	if got := layoutIndex("bg", layouts); got != 1 {
		t.Errorf("bg is at %d, want 1", got)
	}
	if got := layoutIndex("en", layouts); got != 0 {
		t.Errorf("en (alias of us) is at %d, want 0", got)
	}
}

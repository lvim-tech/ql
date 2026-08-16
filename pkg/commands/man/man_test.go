package man

import (
	"os/exec"
	"strings"
	"testing"
)

// A manpage NAME comes from whatever packages are installed, so it is not
// trusted input. It used to be interpolated with %q, which leaves $() and
// backticks live inside the double quotes it produces.
func TestManScriptNeutralisesPageName(t *testing.T) {
	hostile := []string{
		"$(touch pwned)",
		"`touch pwned`",
		"foo; touch pwned",
		"foo && touch pwned",
		"foo | touch pwned",
	}

	for _, name := range hostile {
		script := manScript(name, "less")

		// `man` will not find these pages; what matters is that the shell
		// runs exactly one command and hands the whole name to it.
		out, err := exec.Command("sh", "-c", "printf %s "+strings.TrimSuffix(strings.TrimPrefix(script, "man "), " | less")).Output()
		if err != nil {
			t.Fatalf("%q: quoted name is not a valid shell word: %v", name, err)
		}
		if string(out) != name {
			t.Errorf("%q: shell resolved the name to %q", name, string(out))
		}
	}
}

func TestManScriptPagerStaysExpandable(t *testing.T) {
	// The pager is the user's own config and may carry flags.
	if got, want := manScript("ls", "less"), "man 'ls' | less"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := manScript("ls", "nvimpager"), "man 'ls' | nvimpager -p"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatManpageHonoursShowDescriptions(t *testing.T) {
	const line = "printf (3) - formatted output conversion"

	withDesc := formatManpage(line, true)
	if !strings.Contains(withDesc, "-") || !strings.Contains(withDesc, "output") {
		t.Errorf("description dropped when it was asked for: %q", withDesc)
	}

	// show_descriptions = false used to be decoded and then ignored.
	without := formatManpage(line, false)
	if strings.Contains(without, "output") {
		t.Errorf("description kept when show_descriptions is false: %q", without)
	}
	if got, want := without, "printf (3)"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatManpageEdgeCases(t *testing.T) {
	if got := formatManpage("", true); got != "" {
		t.Errorf("empty line produced %q", got)
	}
	if got := formatManpage("   ", true); got != "" {
		t.Errorf("blank line produced %q", got)
	}
	// A single field has no section to strip; it comes back untouched.
	if got, want := formatManpage("lonely", true), "lonely"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := formatManpage("ls (1)", true), "ls (1)"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

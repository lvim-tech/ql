package screenshot

import (
	"slices"
	"strings"
	"testing"

	"github.com/lvim-tech/ql/pkg/utils"
)

// hostilePath is what a careless (or malicious) save_dir / file_prefix looks
// like. The builders used to interpolate it into `sh -c` with %q, which does
// not stop the shell expanding $() or backticks.
const hostilePath = "/tmp/ql-test/$(touch pwned)`id`_shot.png"

func assertNoShell(t *testing.T, args []string) {
	t.Helper()
	if len(args) == 0 {
		t.Fatal("command has no argv")
	}
	if slices.Contains([]string{"sh", "bash", "/bin/sh", "/bin/bash"}, args[0]) {
		t.Fatalf("command goes through a shell: %v", args)
	}
	if slices.Contains(args, "-c") {
		t.Fatalf("command looks like a shell invocation: %v", args)
	}
}

// The output path must arrive as one argv entry, byte for byte — no quoting,
// no splitting, and nothing left for a shell to expand.
func assertPathIntact(t *testing.T, args []string, want string) {
	t.Helper()
	if !slices.Contains(args, want) {
		t.Fatalf("output path not passed as a single argument: %v", args)
	}
}

func TestBuildWaylandCommandFullscreenPassesPathAsOneArgument(t *testing.T) {
	if !utils.CommandExists("grim") {
		t.Skip("grim not installed")
	}

	cmd, err := buildWaylandCommand("Fullscreen", hostilePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertNoShell(t, cmd.Args)
	assertPathIntact(t, cmd.Args, hostilePath)
}

func TestBuildX11CommandPassesPathAsOneArgument(t *testing.T) {
	if !utils.CommandExists("maim") && !utils.CommandExists("scrot") {
		t.Skip("neither maim nor scrot installed")
	}

	// "Active Window" is left out on purpose: it now resolves the window id
	// by running xdotool, which needs a live X11 session.
	for _, mode := range []string{"Fullscreen", "Select Region"} {
		cmd, err := buildX11Command(mode, hostilePath)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", mode, err)
		}

		assertNoShell(t, cmd.Args)
		assertPathIntact(t, cmd.Args, hostilePath)
	}
}

func TestBuildCommandsRejectUnknownMode(t *testing.T) {
	if _, err := buildWaylandCommand("Telepathy", "/tmp/x.png"); err == nil {
		t.Error("buildWaylandCommand accepted an unknown mode")
	} else if !strings.Contains(err.Error(), "unknown mode") {
		t.Errorf("unexpected error: %v", err)
	}

	if !utils.CommandExists("maim") && !utils.CommandExists("scrot") {
		return
	}
	if _, err := buildX11Command("Telepathy", "/tmp/x.png"); err == nil {
		t.Error("buildX11Command accepted an unknown mode")
	}
}

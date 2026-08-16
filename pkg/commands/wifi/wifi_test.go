package wifi

import (
	"io"
	"slices"
	"strings"
	"testing"
)

const passphrase = "correct horse battery staple"

// The whole point of connectCmd: /proc/<pid>/cmdline is world-readable, so a
// passphrase in argv is a passphrase every local user can read.
func TestConnectCmdKeepsPasswordOutOfArgv(t *testing.T) {
	cmd := connectCmd("HomeNet", passphrase)

	for _, arg := range cmd.Args {
		if strings.Contains(arg, passphrase) {
			t.Fatalf("passphrase appears in argv: %v", cmd.Args)
		}
	}
	if slices.Contains(cmd.Args, "password") {
		t.Errorf("still using the `password <psk>` argv form: %v", cmd.Args)
	}
	if !slices.Contains(cmd.Args, "HomeNet") {
		t.Errorf("SSID missing from argv: %v", cmd.Args)
	}
}

// Out of argv is only half of it — nmcli still has to receive the secret.
func TestConnectCmdFeedsPasswordOnStdin(t *testing.T) {
	cmd := connectCmd("HomeNet", passphrase)

	if cmd.Stdin == nil {
		t.Fatal("no stdin attached — nmcli would never get the passphrase")
	}
	if !slices.Contains(cmd.Args, "--ask") {
		t.Errorf("nmcli was not asked to act as the secret agent: %v", cmd.Args)
	}

	fed, err := io.ReadAll(cmd.Stdin)
	if err != nil {
		t.Fatalf("reading stdin: %v", err)
	}
	if got, want := string(fed), passphrase+"\n"; got != want {
		t.Errorf("stdin = %q, want %q", got, want)
	}
}

// An open network needs no secret agent and no stdin.
func TestConnectCmdWithoutPassword(t *testing.T) {
	cmd := connectCmd("OpenNet", "")

	if cmd.Stdin != nil {
		t.Error("stdin attached for a passwordless connection")
	}
	if slices.Contains(cmd.Args, "--ask") {
		t.Errorf("--ask used with no passphrase to supply: %v", cmd.Args)
	}
	if !slices.Contains(cmd.Args, "OpenNet") {
		t.Errorf("SSID missing from argv: %v", cmd.Args)
	}
}

// An SSID may contain anything, including what looks like an nmcli flag; it
// must stay one argv entry either way.
func TestConnectCmdSSIDStaysOneArgument(t *testing.T) {
	for _, ssid := range []string{"my network", "--ask", "$(id)", "a'b\"c"} {
		cmd := connectCmd(ssid, passphrase)
		if !slices.Contains(cmd.Args, ssid) {
			t.Errorf("SSID %q was not passed intact: %v", ssid, cmd.Args)
		}
	}
}

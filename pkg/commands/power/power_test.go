package power

import (
	"testing"

	"github.com/lvim-tech/ql/pkg/config"
)

func testConfig() *Config {
	cfg := DefaultConfig()
	return &cfg
}

// An empty argument used to reach action[:1] and panic with a slice bounds
// error. `ql power ""` produces exactly that, so it is the case worth pinning.
func TestLookupPowerCommandEmptyAction(t *testing.T) {
	if _, _, ok := lookupPowerCommand(testConfig(), ""); ok {
		t.Fatal("empty action matched a power command")
	}
}

func TestExecuteDirectCommandEmptyActionDoesNotPanic(t *testing.T) {
	// Notifications stay disabled: this path must fail before it reaches
	// the desktop, and a test has no business raising a real toast.
	notifCfg := &config.NotificationConfig{}

	result := executeDirectCommand("", testConfig(), notifCfg)

	if result.Success {
		t.Error("empty action reported success")
	}
	if result.Error == nil {
		t.Fatal("empty action returned no error")
	}
}

func TestLookupPowerCommandIsCaseInsensitive(t *testing.T) {
	cfg := testConfig()

	for _, action := range []string{"shutdown", "Shutdown", "SHUTDOWN", "ShUtDoWn"} {
		label, spec, ok := lookupPowerCommand(cfg, action)
		if !ok {
			t.Fatalf("%q did not resolve", action)
		}
		if label != "Shutdown" {
			t.Errorf("%q resolved to label %q, want %q", action, label, "Shutdown")
		}
		if spec.Command != cfg.ShutdownCommand {
			t.Errorf("%q resolved to command %q, want %q", action, spec.Command, cfg.ShutdownCommand)
		}
	}
}

func TestLookupPowerCommandKnownActions(t *testing.T) {
	cfg := testConfig()

	for _, want := range []string{"Logout", "Suspend", "Hibernate", "Reboot", "Shutdown"} {
		if _, _, ok := lookupPowerCommand(cfg, want); !ok {
			t.Errorf("%q did not resolve", want)
		}
	}
}

func TestLookupPowerCommandUnknownAction(t *testing.T) {
	for _, action := range []string{"selfdestruct", " shutdown", "shutdown ", "shut down", "-"} {
		if _, _, ok := lookupPowerCommand(testConfig(), action); ok {
			t.Errorf("%q resolved but should not have", action)
		}
	}
}

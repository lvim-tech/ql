package commands

import (
	"testing"

	"github.com/lvim-tech/ql/pkg/config"
)

type fakeCtx struct{ cfg *config.Config }

func (f *fakeCtx) Show([]string, string) (string, error) { return "", nil }
func (f *fakeCtx) Config() *config.Config                { return f.cfg }
func (f *fakeCtx) IsDirectLaunch() bool                  { return false }
func (f *fakeCtx) Args() []string                        { return nil }

type demoConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	SaveDir string `mapstructure:"save_dir"`
	Retries int    `mapstructure:"retries"`
}

// A key the user did not write keeps its default — DecodeConfig starts from
// the defaults instead of a zero struct.
func TestDecodeConfigKeepsDefaultsForMissingKeys(t *testing.T) {
	ctx := &fakeCtx{cfg: &config.Config{Commands: map[string]map[string]any{
		"demo": {"save_dir": "/elsewhere"},
	}}}
	got := DecodeConfig(ctx, "demo", demoConfig{Enabled: true, SaveDir: "/default", Retries: 3})
	if !got.Enabled {
		t.Error("Enabled default lost")
	}
	if got.SaveDir != "/elsewhere" {
		t.Errorf("override not applied: %q", got.SaveDir)
	}
	if got.Retries != 3 {
		t.Errorf("Retries default lost: %d", got.Retries)
	}
}

// Weak typing survives the extraction: TOML integers arrive as any and used
// to be decoded with WeaklyTypedInput.
func TestDecodeConfigIsWeaklyTyped(t *testing.T) {
	ctx := &fakeCtx{cfg: &config.Config{Commands: map[string]map[string]any{
		"demo": {"retries": "7", "enabled": "false"},
	}}}
	got := DecodeConfig(ctx, "demo", demoConfig{Enabled: true})
	if got.Retries != 7 {
		t.Errorf("string 7 did not decode into int: %d", got.Retries)
	}
	if got.Enabled {
		t.Error("string false did not decode into bool")
	}
}

// No block at all returns the defaults untouched.
func TestDecodeConfigWithoutBlockReturnsDefaults(t *testing.T) {
	ctx := &fakeCtx{cfg: &config.Config{}}
	got := DecodeConfig(ctx, "demo", demoConfig{Enabled: true, Retries: 9})
	if !got.Enabled || got.Retries != 9 {
		t.Errorf("defaults mangled: %+v", got)
	}
}

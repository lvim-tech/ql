package commands

import (
	"github.com/mitchellh/mapstructure"
)

// DecodeConfig fills a module's typed config from the raw [commands.<name>]
// table, starting from the module's defaults — a key the user did not write
// keeps its default instead of collapsing to the zero value. Thirteen
// modules used to carry this decoder dance as byte-identical copies; a
// failed decode falls back to the defaults, as those copies did.
func DecodeConfig[T any](ctx LauncherContext, name string, defaults T) T {
	raw, ok := ctx.Config().Commands[name]
	if !ok || raw == nil {
		return defaults
	}

	cfg := defaults
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           &cfg,
	})
	if err != nil {
		return defaults
	}
	if err := decoder.Decode(raw); err != nil {
		return defaults
	}
	return cfg
}

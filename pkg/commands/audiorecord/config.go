package audiorecord

// Config за audio recording
type Config struct {
	Enabled    bool   `toml:"enabled" mapstructure:"enabled"`
	SaveDir    string `toml:"save_dir" mapstructure:"save_dir"`
	FilePrefix string `toml:"file_prefix" mapstructure:"file_prefix"`
	Format     string `toml:"format" mapstructure:"format"`
	Quality    string `toml:"quality" mapstructure:"quality"`
}

// DefaultConfig връща default настройки
func DefaultConfig() Config {
	return Config{
		Enabled:    true,
		SaveDir:    "~/Music/Recordings",
		FilePrefix: "audio",
		Format:     "mp3",
		Quality:    "2",
	}
}

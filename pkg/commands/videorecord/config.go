package videorecord

// Config за videorecord модула
type Config struct {
	Enabled     bool   `toml:"enabled"`
	SaveDir     string `toml:"save_dir"`
	FilePrefix  string `toml:"file_prefix"`
	Format      string `toml:"format"`
	Quality     string `toml:"quality"`
	RecordAudio bool   `toml:"record_audio"`
	ShowNotify  bool   `toml:"show_notify"`
}

// ConfigFile за четене от TOML
type ConfigFile struct {
	Enabled     *bool   `toml:"enabled"`
	SaveDir     *string `toml:"save_dir"`
	FilePrefix  *string `toml:"file_prefix"`
	Format      *string `toml:"format"`
	Quality     *string `toml:"quality"`
	RecordAudio *bool   `toml:"record_audio"`
	ShowNotify  *bool   `toml:"show_notify"`
}

package videorecord

// Config за video recording
type Config struct {
	Enabled     bool          `toml:"enabled" mapstructure:"enabled"`
	SaveDir     string        `toml:"save_dir" mapstructure:"save_dir"`
	FilePrefix  string        `toml:"file_prefix" mapstructure:"file_prefix"`
	Format      string        `toml:"format" mapstructure:"format"`
	Quality     string        `toml:"quality" mapstructure:"quality"`
	RecordAudio bool          `toml:"record_audio" mapstructure:"record_audio"`
	ShowNotify  bool          `toml:"show_notify" mapstructure:"show_notify"`
	X11         X11Config     `toml:"x11" mapstructure:"x11"`
	Wayland     WaylandConfig `toml:"wayland" mapstructure:"wayland"`
}

type X11Config struct {
	Framerate  int64  `toml:"framerate" mapstructure:"framerate"`
	OutputFPS  int64  `toml:"output_fps" mapstructure:"output_fps"`
	Preset     string `toml:"preset" mapstructure:"preset"`
	VideoCodec string `toml:"video_codec" mapstructure:"video_codec"`
	AudioCodec string `toml:"audio_codec" mapstructure:"audio_codec"`
}

type WaylandConfig struct {
	Framerate  int64  `toml:"framerate" mapstructure:"framerate"`
	OutputFPS  int64  `toml:"output_fps" mapstructure:"output_fps"`
	Preset     string `toml:"preset" mapstructure:"preset"`
	VideoCodec string `toml:"video_codec" mapstructure:"video_codec"`
	AudioCodec string `toml:"audio_codec" mapstructure:"audio_codec"`
}

// DefaultConfig връща default настройки
func DefaultConfig() Config {
	return Config{
		Enabled:     true,
		SaveDir:     "~/Videos/Recordings",
		FilePrefix:  "screencast",
		Format:      "mp4",
		Quality:     "23",
		RecordAudio: true,
		ShowNotify:  true,
		X11: X11Config{
			Framerate:  60,
			OutputFPS:  30,
			Preset:     "ultrafast",
			VideoCodec: "libx264",
			AudioCodec: "aac",
		},
		Wayland: WaylandConfig{
			Framerate:  60,
			OutputFPS:  30,
			Preset:     "ultrafast",
			VideoCodec: "libx264",
			AudioCodec: "aac",
		},
	}
}

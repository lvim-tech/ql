package config

// ConfigProvider е interface за достъп до конфигурация
// Всеки модул достъпва само своята конфигурация през този interface
type ConfigProvider interface {
	GetPowerConfig() interface{}
	GetScreenshotConfig() interface{}
	GetRadioConfig() interface{}
	GetWifiConfig() interface{}
	GetMpcConfig() interface{}
	GetAudioRecordConfig() interface{}
	GetVideoRecordConfig() interface{}
	GetDefaultLauncher() string
	GetLauncherArgs(name string) []string
}

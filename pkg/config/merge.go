// Package config merge utilities
package config

// ConfigMerger interface за модули които искат да merge-ват config
type ConfigMerger interface {
	Merge(user any)
}

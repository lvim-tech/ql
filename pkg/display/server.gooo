// Package display provides utilities for detecting the current display server type (X11 or Wayland).
// It checks environment variables to determine which display server is running and provides
// helper methods for type checking and information retrieval.
package display

import (
	"os"
)

// ServerType представлява типа display server
type ServerType int

const (
	Unknown ServerType = iota
	X11
	Wayland
)

// Detect открива текущия display server
func Detect() ServerType {
	// Провери за Wayland
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return Wayland
	}

	// Провери за X11
	if os.Getenv("DISPLAY") != "" {
		return X11
	}

	return Unknown
}

// String връща string представяне на ServerType
func (s ServerType) String() string {
	switch s {
	case X11:
		return "X11"
	case Wayland:
		return "Wayland"
	default:
		return "Unknown"
	}
}

// IsX11 проверява дали е X11
func (s ServerType) IsX11() bool {
	return s == X11
}

// IsWayland проверява дали е Wayland
func (s ServerType) IsWayland() bool {
	return s == Wayland
}

// IsUnknown проверява дали е неизвестен
func (s ServerType) IsUnknown() bool {
	return s == Unknown
}

// GetSessionType връща XDG_SESSION_TYPE ако е зададен
func GetSessionType() string {
	return os.Getenv("XDG_SESSION_TYPE")
}

// GetCurrentDesktop връща XDG_CURRENT_DESKTOP ако е зададен
func GetCurrentDesktop() string {
	return os.Getenv("XDG_CURRENT_DESKTOP")
}

package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LogPath is where ql appends its error log. XDG state, not cache: cache is
// allowed to vanish, and this file exists precisely to be read after
// something went wrong.
func LogPath() string {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "ql", "ql.log")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "ql", "ql.log")
}

// Logf appends one timestamped line to the error log. Failures to log are
// swallowed on purpose — logging must never take the launcher down, and
// there is nowhere further to report them anyway.
func Logf(format string, args ...any) {
	path := LogPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
}

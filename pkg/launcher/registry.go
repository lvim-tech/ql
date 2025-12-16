package launcher

import "os/exec"

var registry = make(map[string]Launcher)
var flagMap = make(map[string]Launcher)

// Register добавя launcher в registry
func Register(l Launcher) {
	registry[l.Name()] = l
	flagMap[l.Flag()] = l
}

// GetByName връща launcher по име
func GetByName(name string) Launcher {
	return registry[name]
}

// GetByFlag връща launcher по флаг
func GetByFlag(flag string) Launcher {
	return flagMap[flag]
}

// All връща всички регистрирани launchers
func All() []Launcher {
	var launchers []Launcher
	for _, l := range registry {
		launchers = append(launchers, l)
	}
	return launchers
}

// DetectAvailable намира първия наличен launcher
func DetectAvailable() Launcher {
	// Приоритет: rofi > dmenu > fzf > bemenu > fuzzel
	priority := []string{"rofi", "dmenu", "fzf", "bemenu", "fuzzel"}

	for _, name := range priority {
		if l := GetByName(name); l != nil && l.IsAvailable() {
			return l
		}
	}

	return nil
}

// commandExists проверява дали команда съществува
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

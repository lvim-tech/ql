package launcher

// Launcher interface за различни menu системи
type Launcher interface {
	Name() string                                         // "dmenu", "rofi", etc.
	Flag() string                                         // "d", "r", "f" - единична буква за флаг
	Description() string                                  // "Use dmenu launcher"
	IsAvailable() bool                                    // Проверка дали е инсталиран
	Show(options []string, prompt string) (string, error) // Показва menu

	// НОВ метод за set на custom command
	SetCommand(command string, args []string)
}

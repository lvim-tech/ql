package launcher

import "errors"

var (
	// ErrCancelled се връща когато потребителят натисне ESC/Cancel
	ErrCancelled = errors.New("cancelled by user")
)

// IsCancelled проверява дали грешката е от cancel
func IsCancelled(err error) bool {
	if err == nil {
		return false
	}
	// Провери за exit status 1 (common за dmenu/rofi при ESC)
	return err.Error() == "exit status 1" || errors.Is(err, ErrCancelled)
}

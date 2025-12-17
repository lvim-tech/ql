package commands

var registeredCommands []Command

// Register регистрира команда
func Register(cmd Command) {
	registeredCommands = append(registeredCommands, cmd)
}

// GetAll връща всички регистрирани команди
func GetAll() []Command {
	return registeredCommands
}

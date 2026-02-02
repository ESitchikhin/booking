package app

import (
	"mts/booking_service/internal/ws/server"
)

// Run запускает приложение.
func Run(configPath string) {
	server.Run(configPath)
}

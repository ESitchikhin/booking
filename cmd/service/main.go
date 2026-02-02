package main

import (
	"os"

	"mts/booking_service/internal/app"
)

func main() {
	// Путь к файлу конфигурации можно передавать через флаги или переменные окружения.
	// Для простоты используем значение по умолчанию.
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/local.yml"
	}

	// Запускаем приложение
	app.Run(configPath)
}

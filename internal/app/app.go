package app

import (
	"log"
	"mts/booking_service/internal/config"
	"mts/booking_service/internal/repository/supabase"
	"mts/booking_service/internal/services/standservice"
	"mts/booking_service/internal/ws/handlers"
	"mts/booking_service/internal/ws/server"
)

// Run запускает приложение.
func Run(configPath string) {
	// 1. Инициализация конфигурации
	cfg, err := config.NewConfig(configPath)
	if err != nil {
		log.Fatalf("Ошибка при загрузке конфигурации: %v", err)
	}

	// 2. Инициализация зависимостей
	standsRepo := supabase.NewStandsRepository(&cfg.Supabase)
	hub := handlers.NewHub()
	standSvc := standservice.NewStandService(standsRepo, hub)
	hub.SetService(standSvc)
	go hub.Run()

	standsHandler := handlers.NewStandsHandler(standsRepo)

	// 3. Создание и запуск сервера
	srv := server.New(":"+cfg.Server.Port, hub, standsHandler)
	srv.Run()
}

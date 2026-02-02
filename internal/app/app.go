package app

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mts/booking_service/internal/config"
	"mts/booking_service/internal/repository/supabase"
	"mts/booking_service/internal/services/standservice"
	ws "mts/booking_service/internal/transport/websocket"
)

// Run запускает приложение.
func Run(configPath string) {
	// 1. Инициализация конфигурации
	cfg, err := config.NewConfig(configPath)
	if err != nil {
		log.Fatalf("Ошибка при загрузке конфигурации: %v", err)
	}

	// 2. Инициализация зависимостей
	// Репозиторий
	standsRepo := supabase.NewStandsRepository(&cfg.Supabase)

	// WebSocket Hub в качестве Notifier
	hub := ws.NewHub()

	// Сервис бизнес-логики
	standSvc := standservice.NewStandService(standsRepo, hub)

	// Устанавливаем сервис в Hub
	hub.SetService(standSvc)

	// Запускаем Hub в отдельной горутине
	go hub.Run()

	// 3. Настройка HTTP-сервера и роутинга
	mux := http.NewServeMux()
	mux.Handle("/ws", hub)
	mux.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"success": true}`))
		if err != nil {
			log.Printf("Ошибка записи ответа healthcheck: %v", err)
		}
	})

	server := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: mux,
	}

	// 4. Запуск сервера с Graceful Shutdown
	go func() {
		log.Printf("Сервер запускается на порту %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка при запуске сервера: %v", err)
		}
	}()

	// Ожидание сигнала для завершения работы
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Сервер останавливается...")

	// Контекст с таймаутом для завершения активных соединений
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Ошибка при остановке сервера: %v", err)
	}

	log.Println("Сервер успешно остановлен.")
}

package server

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
	"mts/booking_service/internal/ws/handlers"
)

func Run(configPath string) {
	cfg, err := config.NewConfig(configPath)
	if err != nil {
		log.Fatalf("Ошибка при загрузке конфигурации: %v", err)
	}

	standsRepo := supabase.NewStandsRepository(&cfg.Supabase)
	hub := handlers.NewHub()
	standSvc := standservice.NewStandService(standsRepo, hub)
	hub.SetService(standSvc)
	go hub.Run()

	mux := http.NewServeMux()
	mux.Handle("/ws", hub)
	standsHandler := handlers.NewStandsHandler(standsRepo)
	mux.Handle("/stands", standsHandler)
	mux.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	})

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: mux,
	}

	go func() {
		log.Printf("Сервер запускается на порту %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка при запуске сервера: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Сервер останавливается...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Ошибка при остановке сервера: %v", err)
	}

	log.Println("Сервер успешно остановлен.")
}

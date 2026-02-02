package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Server представляет HTTP-сервер.
type Server struct {
	httpServer *http.Server
}

// New создает новый экземпляр Server.
func New(addr string, wsHandler, standsHandler http.Handler) *Server {
	mux := http.NewServeMux()
	mux.Handle("/ws", wsHandler)
	mux.Handle("/stands", standsHandler)
	mux.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	})

	return &Server{
		httpServer: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}
}

// Run запускает сервер и настраивает graceful shutdown.
func (s *Server) Run() {
	go func() {
		log.Printf("Сервер запускается на %s", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка при запуске сервера: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Сервер останавливается...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Ошибка при graceful shutdown сервера: %v", err)
	}

	log.Println("Сервер успешно остановлен.")
}

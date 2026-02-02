package websocket

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// StandUpdater определяет интерфейс для обновления данных о стендах.
type StandUpdater interface {
	UpdateAndNotify(ctx context.Context, standsData []byte) error
	GetInitialStands(ctx context.Context) ([]byte, error)
}

// upgrader настраивает параметры для обновления HTTP-соединения до WebSocket.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// В проде здесь должна быть проверка на разрешенные домены.
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Hub управляет пулом WebSocket-клиентов.
type Hub struct {
	clients    map[*websocket.Conn]bool
	mu         sync.Mutex
	service    StandUpdater
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
}

// NewHub создает новый Hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

// SetService устанавливает сервис для хаба.
func (h *Hub) SetService(service StandUpdater) {
	h.service = service
}

// Run запускает главный цикл Hub для обработки событий.
func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()
			log.Println("Новый клиент подключен.")
			// Отправляем новому клиенту актуальное состояние
			initialStands, err := h.service.GetInitialStands(context.Background())
			if err != nil {
				log.Printf("Ошибка получения начального состояния стендов: %v", err)
			} else {
				if err := conn.WriteMessage(websocket.TextMessage, initialStands); err != nil {
					log.Printf("Ошибка отправки начального состояния клиенту: %v", err)
				}
			}

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
				log.Println("Клиент отключен.")
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.Lock()
			for conn := range h.clients {
				if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
					log.Printf("Ошибка отправки сообщения клиенту: %v", err)
				}
			}
			h.mu.Unlock()
			log.Println("Сообщение разослано всем клиентам.")
		}
	}
}

// Broadcast реализует интерфейс Notifier для StandService.
func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}

// ServeHTTP обрабатывает входящие HTTP-запросы и обновляет их до WebSocket.
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Ошибка обновления до WebSocket: %v", err)
		return
	}

	h.register <- conn

	// Горутина для чтения сообщений от данного клиента
	go h.handleClientMessages(conn)
}

// handleClientMessages читает сообщения от клиента и передает их в сервис.
func (h *Hub) handleClientMessages(conn *websocket.Conn) {
	defer func() {
		h.unregister <- conn
	}()

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Неожиданная ошибка закрытия соединения: %v", err)
			}
			break
		}

		if messageType == websocket.TextMessage {
			// Передаем полученные данные в сервис для обработки
			if err := h.service.UpdateAndNotify(context.Background(), p); err != nil {
				log.Printf("Ошибка при обработке сообщения от клиента: %v", err)
				// Опционально: отправить клиенту сообщение об ошибке
				errorMsg := []byte(`{"error": "Не удалось обновить данные."}`)
				if writeErr := conn.WriteMessage(websocket.TextMessage, errorMsg); writeErr != nil {
					log.Printf("Ошибка отправки сообщения об ошибке клиенту: %v", writeErr)
				}
			}
		}
	}
}

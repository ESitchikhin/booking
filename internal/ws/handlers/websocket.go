package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"mts/booking_service/internal/ws/dto"
)

// StandUpdater определяет интерфейс для сервиса стендов.
type StandUpdater interface {
	UpdateStand(ctx context.Context, id string, data []byte) error
	GetInitialStands(ctx context.Context) ([]byte, error)
}

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
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
			h.sendInitialStands(conn)

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
				updateMsg := dto.WsMessage{
					Type:    "UPDATE",
					Payload: message,
				}
				if err := conn.WriteJSON(updateMsg); err != nil {
					log.Printf("Ошибка отправки сообщения клиенту: %v", err)
				}
			}
			h.mu.Unlock()
			log.Println("Сообщение 'UPDATE' разослано всем клиентам.")
		}
	}
}

func (h *Hub) sendInitialStands(conn *websocket.Conn) {
	initialStands, err := h.service.GetInitialStands(context.Background())
	if err != nil {
		log.Printf("Ошибка получения начального состояния стендов: %v", err)
		h.sendError(conn, "Не удалось получить начальное состояние стендов.")
		return
	}

	updateMsg := dto.WsMessage{
		Type:    "UPDATE",
		Payload: initialStands,
	}
	if err := conn.WriteJSON(updateMsg); err != nil {
		log.Printf("Ошибка отправки начального состояния клиенту: %v", err)
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
	go h.handleClientMessages(conn)
}

func (h *Hub) handleClientMessages(conn *websocket.Conn) {
	defer func() {
		h.unregister <- conn
	}()

	for {
		var msg dto.WsMessage
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("Ошибка чтения JSON сообщения: %v", err)
			h.sendError(conn, "Некорректный формат сообщения.")
			// В случае ошибки чтения, соединение, вероятно, невалидно, поэтому выходим из цикла.
			break
		}

		switch msg.Type {
		case "PATCH":
			h.handlePatch(conn, msg.Payload)
		default:
			log.Printf("Получен неизвестный тип сообщения: %s", msg.Type)
			h.sendError(conn, "Неизвестный тип сообщения.")
		}
	}
}

func (h *Hub) handlePatch(conn *websocket.Conn, payload json.RawMessage) {
	var patchPayload dto.PatchPayload
	if err := json.Unmarshal(payload, &patchPayload); err != nil {
		log.Printf("Ошибка парсинга PATCH payload: %v", err)
		h.sendError(conn, "Некорректный payload для PATCH сообщения.")
		return
	}

	if err := h.service.UpdateStand(context.Background(), patchPayload.ID, patchPayload.UpdateData); err != nil {
		log.Printf("Ошибка при обработке PATCH сообщения от клиента: %v", err)
		h.sendError(conn, "Не удалось обновить данные.")
		return
	}
}

func (h *Hub) sendError(conn *websocket.Conn, message string) {
	errorPayload := dto.ErrorPayload{Message: message}
	payloadBytes, _ := json.Marshal(errorPayload)
	errorMsg := dto.WsMessage{
		Type:    "ERROR",
		Payload: payloadBytes,
	}
	if err := conn.WriteJSON(errorMsg); err != nil {
		log.Printf("Ошибка отправки сообщения об ошибке клиенту: %v", err)
	}
}

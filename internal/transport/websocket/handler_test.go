package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
)

// MockStandUpdater - это мок для сервиса стендов.
type MockStandUpdater struct {
	UpdateStandFunc      func(ctx context.Context, id string, data []byte) error
	GetInitialStandsFunc func(ctx context.Context) ([]byte, error)
}

func (m *MockStandUpdater) UpdateStand(ctx context.Context, id string, data []byte) error {
	if m.UpdateStandFunc != nil {
		return m.UpdateStandFunc(ctx, id, data)
	}
	return nil
}

func (m *MockStandUpdater) GetInitialStands(ctx context.Context) ([]byte, error) {
	if m.GetInitialStandsFunc != nil {
		return m.GetInitialStandsFunc(ctx)
	}
	return []byte(`[{"id":"initial"}]`), nil
}

// Helper для создания тестового websocket клиента
func newTestWsClient(t *testing.T, serverURL string) *websocket.Conn {
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Не удалось подключиться к WebSocket: %v", err)
	}
	return conn
}

func TestHub_ClientCommunication(t *testing.T) {
	service := &MockStandUpdater{}
	hub := NewHub()
	hub.SetService(service)
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(hub.ServeHTTP))
	defer server.Close()

	t.Run("получение начального состояния", func(t *testing.T) {
		conn := newTestWsClient(t, server.URL)
		defer conn.Close()

		var receivedMsg WsMessage
		err := conn.ReadJSON(&receivedMsg)
		if err != nil {
			t.Fatalf("Не удалось прочитать JSON сообщение: %v", err)
		}

		if receivedMsg.Type != "UPDATE" {
			t.Errorf("Ожидался тип 'UPDATE', получено '%s'", receivedMsg.Type)
		}
		if string(receivedMsg.Payload) != `[{"id":"initial"}]` {
			t.Errorf("Получен неожиданный payload: %s", string(receivedMsg.Payload))
		}
	})

	t.Run("успешная обработка PATCH сообщения", func(t *testing.T) {
		conn := newTestWsClient(t, server.URL)
		defer conn.Close()
		conn.ReadJSON(new(WsMessage)) // Пропускаем начальное сообщение

		var wg sync.WaitGroup
		wg.Add(1)

		service.UpdateStandFunc = func(ctx context.Context, id string, data []byte) error {
			if id != "stand1" {
				t.Errorf("Ожидался id 'stand1', получено '%s'", id)
			}
			if string(data) != `{"status":"occupied"}` {
				t.Errorf("Получены неожиданные данные: %s", string(data))
			}
			return nil
		}

		patchPayload, _ := json.Marshal(PatchPayload{
			ID:         "stand1",
			UpdateData: json.RawMessage(`{"status":"occupied"}`),
		})
		patchMsg := WsMessage{Type: "PATCH", Payload: patchPayload}

		if err := conn.WriteJSON(patchMsg); err != nil {
			t.Fatalf("Не удалось отправить JSON сообщение: %v", err)
		}

		wg.Wait()
	})

	t.Run("обработка неизвестного типа сообщения", func(t *testing.T) {
		conn := newTestWsClient(t, server.URL)
		defer conn.Close()
		conn.ReadJSON(new(WsMessage)) // Пропускаем начальное сообщение

		unknownMsg := WsMessage{Type: "UNKNOWN", Payload: []byte(`{}`)}
		if err := conn.WriteJSON(unknownMsg); err != nil {
			t.Fatalf("Не удалось отправить JSON сообщение: %v", err)
		}

		var errRsp WsMessage
		conn.ReadJSON(&errRsp)
		if errRsp.Type != "ERROR" {
			t.Errorf("Ожидался тип 'ERROR', получено '%s'", errRsp.Type)
		}
	})

	t.Run("обработка ошибки от сервиса", func(t *testing.T) {
		conn := newTestWsClient(t, server.URL)
		defer conn.Close()
		conn.ReadJSON(new(WsMessage)) // Пропускаем начальное сообщение

		service.UpdateStandFunc = func(ctx context.Context, id string, data []byte) error {
			return errors.New("service error")
		}

		patchPayload, _ := json.Marshal(PatchPayload{ID: "stand1"})
		patchMsg := WsMessage{Type: "PATCH", Payload: patchPayload}
		conn.WriteJSON(patchMsg)

		var errRsp WsMessage
		conn.ReadJSON(&errRsp)
		if errRsp.Type != "ERROR" {
			t.Errorf("Ожидался тип 'ERROR', получено '%s'", errRsp.Type)
		}
	})
}

package websocket

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// MockStandUpdater - это мок для сервиса стендов.
type MockStandUpdater struct {
	UpdateAndNotifyFunc  func(ctx context.Context, data []byte) error
	GetInitialStandsFunc func(ctx context.Context) ([]byte, error)
}

func (m *MockStandUpdater) UpdateAndNotify(ctx context.Context, data []byte) error {
	if m.UpdateAndNotifyFunc != nil {
		return m.UpdateAndNotifyFunc(ctx, data)
	}
	return nil
}

func (m *MockStandUpdater) GetInitialStands(ctx context.Context) ([]byte, error) {
	if m.GetInitialStandsFunc != nil {
		return m.GetInitialStandsFunc(ctx)
	}
	return []byte("initial"), nil
}

func TestHub_Run(t *testing.T) {
	service := &MockStandUpdater{}
	hub := NewHub()
	hub.SetService(service)
	go hub.Run()

	// Проверка регистрации, дерегистрации и рассылки
	t.Run("регистрация и дерегистрация", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(hub.ServeHTTP))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Не удалось подключиться к WebSocket: %v", err)
		}

		// Даем время на регистрацию
		time.Sleep(100 * time.Millisecond)
		hub.mu.Lock()
		if len(hub.clients) != 1 {
			t.Errorf("Ожидался 1 клиент, получено %d", len(hub.clients))
		}
		hub.mu.Unlock()

		conn.Close()
		// Даем время на дерегистрацию
		time.Sleep(100 * time.Millisecond)
		hub.mu.Lock()
		if len(hub.clients) != 0 {
			t.Errorf("Ожидалось 0 клиентов, получено %d", len(hub.clients))
		}
		hub.mu.Unlock()
	})

	t.Run("рассылка сообщений", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(hub.ServeHTTP))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Не удалось подключиться (conn1): %v", err)
		}
		defer conn1.Close()

		conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Не удалось подключиться (conn2): %v", err)
		}
		defer conn2.Close()

		// Даем время на регистрацию
		time.Sleep(100 * time.Millisecond)

		var wg sync.WaitGroup
		wg.Add(2)

		// Клиент 1
		go func() {
			defer wg.Done()
			// Сначала читаем и игнорируем начальное сообщение
			if _, _, err := conn1.ReadMessage(); err != nil {
				t.Errorf("Ошибка чтения начального сообщения (conn1): %v", err)
				return
			}
			// Теперь читаем широковещательное сообщение
			_, msg, err := conn1.ReadMessage()
			if err != nil {
				t.Errorf("Ошибка чтения широковещательного сообщения (conn1): %v", err)
				return
			}
			if string(msg) != "broadcast message" {
				t.Errorf("Ожидалось 'broadcast message', получено '%s'", string(msg))
			}
		}()

		// Клиент 2
		go func() {
			defer wg.Done()
			// Сначала читаем и игнорируем начальное сообщение
			if _, _, err := conn2.ReadMessage(); err != nil {
				t.Errorf("Ошибка чтения начального сообщения (conn2): %v", err)
				return
			}
			// Теперь читаем широковещательное сообщение
			_, msg, err := conn2.ReadMessage()
			if err != nil {
				t.Errorf("Ошибка чтения широковещательного сообщения (conn2): %v", err)
				return
			}
			if string(msg) != "broadcast message" {
				t.Errorf("Ожидалось 'broadcast message', получено '%s'", string(msg))
			}
		}()

		// Даем горутинам время на запуск и начало чтения
		time.Sleep(50 * time.Millisecond)

		hub.Broadcast([]byte("broadcast message"))

		// Ждем завершения всех горутин
		wg.Wait()
	})
}

func TestHub_ClientCommunication(t *testing.T) {
	t.Run("получение начального состояния", func(t *testing.T) {
		service := &MockStandUpdater{
			GetInitialStandsFunc: func(ctx context.Context) ([]byte, error) {
				return []byte("initial state"), nil
			},
		}
		hub := NewHub()
		hub.SetService(service)
		go hub.Run()

		server := httptest.NewServer(http.HandlerFunc(hub.ServeHTTP))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Не удалось подключиться: %v", err)
		}
		defer conn.Close()

		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("Не удалось прочитать сообщение: %v", err)
		}

		if string(msg) != "initial state" {
			t.Errorf("Ожидалось 'initial state', получено '%s'", string(msg))
		}
	})

	t.Run("обработка сообщения от клиента", func(t *testing.T) {
		var serviceCallData string
		service := &MockStandUpdater{
			UpdateAndNotifyFunc: func(ctx context.Context, data []byte) error {
				serviceCallData = string(data)
				return nil
			},
		}
		hub := NewHub()
		hub.SetService(service)
		go hub.Run()

		server := httptest.NewServer(http.HandlerFunc(hub.ServeHTTP))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Не удалось подключиться: %v", err)
		}
		defer conn.Close()

		conn.ReadMessage() // Пропускаем начальное сообщение

		if err := conn.WriteMessage(websocket.TextMessage, []byte("client message")); err != nil {
			t.Fatalf("Не удалось отправить сообщение: %v", err)
		}

		time.Sleep(100 * time.Millisecond) // время на обработку

		if serviceCallData != "client message" {
			t.Errorf("Сервис был вызван с '%s', ожидалось 'client message'", serviceCallData)
		}
	})

	t.Run("обработка ошибки от сервиса", func(t *testing.T) {
		service := &MockStandUpdater{
			UpdateAndNotifyFunc: func(ctx context.Context, data []byte) error {
				return errors.New("service error")
			},
		}
		hub := NewHub()
		hub.SetService(service)
		go hub.Run()

		server := httptest.NewServer(http.HandlerFunc(hub.ServeHTTP))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Не удалось подключиться: %v", err)
		}
		defer conn.Close()

		conn.ReadMessage() // Пропускаем начальное сообщение

		if err := conn.WriteMessage(websocket.TextMessage, []byte("test")); err != nil {
			t.Fatalf("Не удалось отправить сообщение: %v", err)
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("Не удалось прочитать сообщение об ошибке: %v", err)
		}

		expectedError := `{"error": "Не удалось обновить данные."}`
		if string(msg) != expectedError {
			t.Errorf("Ожидалось сообщение об ошибке '%s', получено '%s'", expectedError, string(msg))
		}
	})
}

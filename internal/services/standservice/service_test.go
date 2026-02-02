package standservice

import (
	"context"
	"errors"
	"testing"
)

// MockRepository - это мок для репозитория.
type MockRepository struct {
	UpdateStandsFunc func(ctx context.Context, id string, data []byte) error
	GetStandsFunc    func(ctx context.Context) ([]byte, error)
}

func (m *MockRepository) UpdateStands(ctx context.Context, id string, data []byte) error {
	if m.UpdateStandsFunc != nil {
		return m.UpdateStandsFunc(ctx, id, data)
	}
	return nil
}

func (m *MockRepository) GetStands(ctx context.Context) ([]byte, error) {
	if m.GetStandsFunc != nil {
		return m.GetStandsFunc(ctx)
	}
	return []byte("initial data"), nil
}

// MockNotifier - это мок для уведомителя.
type MockNotifier struct {
	BroadcastFunc   func(message []byte)
	broadcastCalled bool
	lastMessage     []byte
}

func (m *MockNotifier) Broadcast(message []byte) {
	m.broadcastCalled = true
	m.lastMessage = message
	if m.BroadcastFunc != nil {
		m.BroadcastFunc(message)
	}
}

func TestStandService_UpdateAndNotify(t *testing.T) {
	t.Run("успешное обновление и уведомление", func(t *testing.T) {
		repo := &MockRepository{
			GetStandsFunc: func(ctx context.Context) ([]byte, error) {
				return []byte("updated data"), nil
			},
		}
		notifier := &MockNotifier{}
		service := NewStandService(repo, notifier)

		patchMsg := `{"id": "stand1", "updateData": {"status": "occupied"}}`
		err := service.UpdateAndNotify(context.Background(), []byte(patchMsg))

		if err != nil {
			t.Errorf("Ожидалась ошибка nil, получено %v", err)
		}

		if !notifier.broadcastCalled {
			t.Error("Ожидался вызов Broadcast, но он не был вызван")
		}

		if string(notifier.lastMessage) != "updated data" {
			t.Errorf("Ожидалось сообщение 'updated data', получено '%s'", string(notifier.lastMessage))
		}
	})

	t.Run("ошибка парсинга", func(t *testing.T) {
		repo := &MockRepository{}
		notifier := &MockNotifier{}
		service := NewStandService(repo, notifier)

		err := service.UpdateAndNotify(context.Background(), []byte("invalid json"))

		if err == nil {
			t.Error("Ожидалась ошибка парсинга, но получено nil")
		}
	})

	t.Run("ошибка при обновлении в репозитории", func(t *testing.T) {
		repo := &MockRepository{
			UpdateStandsFunc: func(ctx context.Context, id string, data []byte) error {
				return errors.New("repo update error")
			},
		}
		notifier := &MockNotifier{}
		service := NewStandService(repo, notifier)

		patchMsg := `{"id": "stand1", "updateData": {"status": "occupied"}}`
		err := service.UpdateAndNotify(context.Background(), []byte(patchMsg))

		if err == nil {
			t.Error("Ожидалась ошибка, но получено nil")
		}

		if notifier.broadcastCalled {
			t.Error("Broadcast не должен был вызываться при ошибке обновления")
		}
	})
}

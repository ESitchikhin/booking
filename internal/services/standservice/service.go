package standservice

import (
	"context"
	"log"
)

// Repository определяет интерфейс для работы с хранилищем стендов.
type Repository interface {
	UpdateStands(ctx context.Context, standsData []byte) error
	GetStands(ctx context.Context) ([]byte, error)
}

// Notifier определяет интерфейс для отправки уведомлений.
// Это позволяет отделить сервис от конкретной реализации транспорта (WebSocket).
type Notifier interface {
	Broadcast(message []byte)
}

// StandService предоставляет бизнес-логику для управления стендами.
type StandService struct {
	repo     Repository
	notifier Notifier
}

// NewStandService создает новый экземпляр StandService.
func NewStandService(repo Repository, notifier Notifier) *StandService {
	return &StandService{
		repo:     repo,
		notifier: notifier,
	}
}

// UpdateAndNotify обновляет данные о стендах и уведомляет клиентов.
// Этот метод является основной точкой входа для бизнес-логики.
func (s *StandService) UpdateAndNotify(ctx context.Context, standsData []byte) error {
	// 1. Обновить данные в репозитории (Supabase).
	if err := s.repo.UpdateStands(ctx, standsData); err != nil {
		log.Printf("Ошибка обновления стендов в репозитории: %v", err)
		return err
	}

	// 2. Получить актуальное состояние (на случай, если в БД есть триггеры или доп. логика).
	// Это также гарантирует, что мы рассылаем консистентные данные.
	latestStands, err := s.repo.GetStands(ctx)
	if err != nil {
		log.Printf("Ошибка получения актуального состояния стендов: %v", err)
		return err
	}

	// 3. Разослать уведомление всем клиентам через Notifier.
	if s.notifier != nil {
		s.notifier.Broadcast(latestStands)
	}

	log.Println("Стенды успешно обновлены и уведомления разосланы.")
	return nil
}

// GetInitialStands возвращает начальное состояние стендов для нового клиента.
func (s *StandService) GetInitialStands(ctx context.Context) ([]byte, error) {
	return s.repo.GetStands(ctx)
}

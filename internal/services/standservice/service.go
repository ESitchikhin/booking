package standservice

import (
	"context"
	"log"
)

// Repository определяет интерфейс для работы с хранилищем стендов.
type Repository interface {
	Patch(ctx context.Context, id string, standsData []byte) error
	GetStands(ctx context.Context) ([]byte, error)
}

// Notifier определяет интерфейс для отправки уведомлений.
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

// UpdateStand обновляет данные о стенде и уведомляет всех клиентов.
func (s *StandService) UpdateStand(ctx context.Context, id string, data []byte) error {
	if err := s.repo.Patch(ctx, id, data); err != nil {
		log.Printf("Ошибка обновления стенда в репозитории: %v", err)
		return err
	}

	latestStands, err := s.repo.GetStands(ctx)
	if err != nil {
		log.Printf("Ошибка получения актуального состояния стендов: %v", err)
		return err
	}

	if s.notifier != nil {
		s.notifier.Broadcast(latestStands)
	}

	log.Println("Стенд успешно обновлен и уведомления разосланы.")
	return nil
}

// GetInitialStands возвращает начальное состояние стендов для нового клиента.
func (s *StandService) GetInitialStands(ctx context.Context) ([]byte, error) {
	return s.repo.GetStands(ctx)
}

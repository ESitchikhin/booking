package standservice

import (
	"context"
	"encoding/json"
	"log"
)

// PatchMessage определяет структуру для частичного обновления.
type PatchMessage struct {
	ID         string          `json:"id"`
	UpdateData json.RawMessage `json:"updateData"`
}

// Repository определяет интерфейс для работы с хранилищем стендов.
type Repository interface {
	UpdateStands(ctx context.Context, id string, standsData []byte) error
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

// UpdateAndNotify обновляет данные о стендах и уведомляет клиентов.
func (s *StandService) UpdateAndNotify(ctx context.Context, updateData []byte) error {
	id, standData, err := s.parseUpdateData(updateData)
	if err != nil {
		log.Printf("Ошибка парсинга данных для обновления: %v", err)
		return err
	}

	// 1. Обновить данные в репозитории (Supabase).
	if err := s.repo.UpdateStands(ctx, id, standData); err != nil {
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

// parseUpdateData извлекает id и данные для обновления из входящего сообщения.
func (s *StandService) parseUpdateData(updateData []byte) (string, []byte, error) {
	var msg PatchMessage
	if err := json.Unmarshal(updateData, &msg); err != nil {
		return "", nil, err
	}
	return msg.ID, msg.UpdateData, nil
}

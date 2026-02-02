package supabase

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"mts/booking_service/internal/config"
)

// StandsRepository представляет собой репозиторий для работы со стендами в Supabase.
type StandsRepository struct {
	client *http.Client
	cfg    *config.SupabaseConfig
}

// NewStandsRepository создает новый экземпляр репозитория.
func NewStandsRepository(cfg *config.SupabaseConfig) *StandsRepository {
	return &StandsRepository{
		client: &http.Client{},
		cfg:    cfg,
	}
}

// UpdateStands обновляет данные о стендах в Supabase.
// Вместо прямого изменения, он перезаписывает все данные (согласно ТЗ).
func (r *StandsRepository) UpdateStands(ctx context.Context, standsData []byte) error {
	// В Supabase для перезаписи обычно используется POST с заголовком "Prefer: resolution=merge-duplicates"
	// или PATCH, если есть уникальный ключ.
	// Для простоты примера, будем считать что мы полностью заменяем данные.
	// Предположим, что у нас есть таблица 'stands' с одним столбцом 'data' типа jsonb.
	// Для этого нужно использовать RPC или кастомный API-роут.
	// Здесь для примера используем гипотетический REST-эндпоинт.
	// Формат запроса может сильно отличаться в зависимости от структуры БД.

	// В реальном проекте, URL и структура запроса должны быть более гибкими.
	reqURL := fmt.Sprintf("%s/rest/v1/stands", r.cfg.URL) // Пример, может потребоваться адаптация

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewBuffer(standsData))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("apikey", r.cfg.APIKey)
	req.Header.Set("Authorization", "Bearer "+r.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	// Этот заголовок важен для Upsert
	req.Header.Set("Prefer", "resolution=merge-duplicates")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка при выполнении запроса к Supabase: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Supabase вернул ошибку: статус %d, тело %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetStands получает актуальное состояние стендов из Supabase.
func (r *StandsRepository) GetStands(ctx context.Context) ([]byte, error) {
	reqURL := fmt.Sprintf("%s/rest/v1/stands", r.cfg.URL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("apikey", r.cfg.APIKey)
	req.Header.Set("Authorization", "Bearer "+r.cfg.APIKey)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса к Supabase: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Supabase вернул ошибку: статус %d, тело %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа от Supabase: %w", err)
	}

	return body, nil
}

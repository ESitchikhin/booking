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

// Patch обновляет данные о стенде в Supabase.
func (r *StandsRepository) Patch(ctx context.Context, id string, data []byte) error {
	reqURL := fmt.Sprintf("%s/rest/v1/stands?id=%s", r.cfg.URL, id)

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, reqURL, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Apikey", r.cfg.APIKey)
	req.Header.Set("Authorization", "Bearer "+r.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
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
	reqURL := fmt.Sprintf("%s/rest/v1/stands?select=*", r.cfg.URL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Apikey", r.cfg.APIKey)
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

package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"mts/booking_service/internal/services/standservice"
)

type StandsHandler struct {
	repo standservice.Repository
}

func NewStandsHandler(repo standservice.Repository) *StandsHandler {
	return &StandsHandler{repo: repo}
}

func (h *StandsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r)
	case http.MethodPatch:
		h.handlePatch(w, r)
	default:
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
	}
}

func (h *StandsHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	stands, err := h.repo.GetStands(r.Context())
	if err != nil {
		http.Error(w, "Ошибка получения данных", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(stands)
}

func (h *StandsHandler) handlePatch(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Ошибка чтения тела запроса", http.StatusBadRequest)
		return
	}

	var patchData struct {
		ID         string          `json:"id"`
		UpdateData json.RawMessage `json:"updateData"`
	}

	if err := json.Unmarshal(body, &patchData); err != nil {
		http.Error(w, "Некорректный формат JSON", http.StatusBadRequest)
		return
	}

	if err := h.repo.Patch(r.Context(), patchData.ID, patchData.UpdateData); err != nil {
		log.Printf("Ошибка обновления стенда: %v", err)
		http.Error(w, "Ошибка обновления данных", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

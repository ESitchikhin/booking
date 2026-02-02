package dto

import "encoding/json"

// WsMessage - это общая структура для всех WebSocket сообщений.
type WsMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// PatchPayload - это структура для payload'а PATCH сообщения.
type PatchPayload struct {
	ID         string          `json:"id"`
	UpdateData json.RawMessage `json:"updateData"`
}

// ErrorPayload - это структура для payload'а ERROR сообщения.
type ErrorPayload struct {
	Message string `json:"message"`
}

package handler

import (
	"encoding/json"
	"errors"
	"log-parser/internal/service"
	"net/http"
)

type ParserHandler struct {
	service *service.ParserService
}

func NewParserHandler(s *service.ParserService) *ParserHandler {
	return &ParserHandler{service: s}
}

func (h *ParserHandler) Parse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		Log string `json:"log"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	err := h.service.ParseAndSave(r.Context(), input.Log)
	if err != nil {
		// Проверяем тип ошибки
		switch {
		case errors.Is(err, service.ErrEmptyLog), errors.Is(err, service.ErrInvalidFormat):
			sendError(w, err.Error(), http.StatusBadRequest)
		default:
			sendError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// функция для ответов с ошибками
func sendError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func (h *ParserHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logs, err := h.service.GetLogs(r.Context())
	if err != nil {
		sendError(w, "Failed to fetch logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

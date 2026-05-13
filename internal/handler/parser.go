package handler

import (
	"encoding/json"
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		Log string `json:"log"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.ParseAndSave(r.Context(), input.Log); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

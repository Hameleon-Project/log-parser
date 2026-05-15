package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log-parser/internal/service"
	"net/http"
	"strconv"
)

type ParserHandler struct {
	service *service.ParserService
}

func NewParserHandler(s *service.ParserService) *ParserHandler {
	return &ParserHandler{service: s}
}

func (h *ParserHandler) Parse(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	if input.Path == "" {
		sendError(w, "path is required", http.StatusBadRequest)
		return
	}

	logID, err := h.service.ParseAndSave(r.Context(), input.Path)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidDataPath):
			sendError(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, service.ErrParseLog):
			sendError(w, err.Error(), http.StatusBadRequest)
		default:
			sendError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]int{"log_id": logID})
}

func (h *ParserHandler) GetTopology(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("log_id")
	logID, err := strconv.Atoi(idStr)
	if err != nil {
		sendError(w, "Invalid log ID", http.StatusBadRequest)
		return
	}

	topo, err := h.service.GetTopology(r.Context(), logID)
	if err != nil {
		sendError(w, "Failed to fetch topology", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(topo)
}

func (h *ParserHandler) GetNode(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("node_id")
	nodeID, err := strconv.Atoi(idStr)
	if err != nil {
		sendError(w, "Invalid node ID", http.StatusBadRequest)
		return
	}

	node, err := h.service.GetNodeDetails(r.Context(), nodeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			sendError(w, "Node not found", http.StatusNotFound)
			return
		}
		sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(node)
}

func (h *ParserHandler) GetPorts(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("node_id")
	nodeID, err := strconv.Atoi(idStr)
	if err != nil {
		sendError(w, "Invalid node ID", http.StatusBadRequest)
		return
	}

	ports, err := h.service.GetPortsByNode(r.Context(), nodeID)
	if err != nil {
		sendError(w, "Failed to fetch ports", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ports)
}

func (h *ParserHandler) GetLogMeta(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("log_id")
	logID, err := strconv.Atoi(idStr)
	if err != nil {
		sendError(w, "Invalid log ID", http.StatusBadRequest)
		return
	}

	meta, err := h.service.GetLogMeta(r.Context(), logID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			sendError(w, "Log not found", http.StatusNotFound)
			return
		}
		sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(meta)
}

func sendError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
